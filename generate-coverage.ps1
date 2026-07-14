$dirname = Split-Path -Parent $MyInvocation.MyCommand.Path
$dirname = Split-Path -Leaf $dirname

$DOCKER_COMPOSE_FILE = "./test-databases.docker-compose.yml"
$CHECKPOINT_FILE = "./.test_checkpoint"

# Ensure gocovmerge is available for the final merge step
Write-Host "Ensuring gocovmerge is installed..."
go install github.com/wadey/gocovmerge@latest

# Databases to test (these translate to Go build tags)
$Databases = @(
    "sqlite",
    "mysql_local",
    "mariadb",
    "mysql",
    "postgres"
)

# Databases defined in docker-compose.yml
$DockerDatabases = @{
    "mysql" = $true   
    "mariadb"  = $true
    "postgres" = $true
}

# Empty test array will be built out
$testsToRun = @()
$coverFiles = @()

# Flags for the test run
$flags = @{
    verbose = $false
    failslow = $false
    runTests = $true
    resume = $false
}

# Check script arguments that were passed in
foreach ($arg in $args) {
    switch ($arg) {
        "verbose" {
            $flags.verbose = $true
            continue
        }
        "failslow" {
            $flags.failslow = $true
            continue
        }
        "fake" {
            $flags.runTests = $false
            continue
        }
        "resume" {
            $flags.resume = $true
            continue
        }
        "down" {
            # If the argument is "down", tear everything down
            docker-compose -f $DOCKER_COMPOSE_FILE down
            foreach ($db in $DockerDatabases.Keys) {
                docker volume rm "${dirname}_go-django_${db}_data" -f
            }
            if (Test-Path $CHECKPOINT_FILE) { Remove-Item $CHECKPOINT_FILE }
            exit 0
        }
        default {
            $testsToRun += $arg
        }
    }
}

if ($testsToRun.Count -eq 0) {
    # If no specific tests were provided, run all databases
    $testsToRun = $Databases
}

# ---------------------------------------------------------
# RESUME STATE LOGIC
# ---------------------------------------------------------
$skipUntilDb = ""
$skipUntilRunIndex = -1

if ($flags.resume -and (Test-Path $CHECKPOINT_FILE)) {
    $checkpoint = Get-Content $CHECKPOINT_FILE
    $parts = $checkpoint -split ":"
    $skipUntilDb = $parts[0]
    $skipUntilRunIndex = [int]$parts[1]
    Write-Host ">>> RESUMING RUN from Database: $skipUntilDb, Step: $skipUntilRunIndex"
} else {
    # If we aren't resuming, start fresh by taking everything down
    docker-compose -f $DOCKER_COMPOSE_FILE down
}

if (-not $flags.runTests) {
    Write-Host "Skipping test execution as per the 'fake' flag."
    exit 0
}

# Define the exact 4 test permutations you provided
$pkgQueries = "github.com/Nigel2392/go-django/queries/..."
$pkgSrc = "github.com/Nigel2392/go-django/src/..."
$dirQueries = "./queries/..."
$dirSrc = "./src/..."

# Run tests for each database type
foreach ($Database in $testsToRun) {
    
    # Check if we are fast-forwarding to a checkpointed DB
    if ($flags.resume -and $skipUntilDb -ne "" -and $skipUntilDb -ne $Database) {
        Write-Host "Skipping $Database (already passed)..."
        continue
    }
    $skipUntilDb = "" # Clear checkpoint lock once we reach the correct DB

    Write-Host "========================================"
    Write-Host "Running 4-way tests for: $Database"
    Write-Host "========================================"

    # Array of the 4 manual runs per database
    $runs = @(
        @{ name = "Queries -> Queries"; coverpkg = $pkgQueries; target = $dirQueries; out = "cover.$Database.queries.out" },
        @{ name = "Src -> Src";         coverpkg = $pkgSrc;     target = $dirSrc;     out = "cover.$Database.src.out" },
        @{ name = "Queries -> Src";     coverpkg = $pkgSrc;     target = $dirQueries; out = "cover.$Database.queries.src.out" },
        @{ name = "Src -> Queries";     coverpkg = $pkgQueries; target = $dirSrc;     out = "cover.$Database.src.queries.out" }
    )

    for ($i = 0; $i -lt $runs.Count; $i++) {
        $run = $runs[$i]

        # Check if we are fast-forwarding to a specific test step
        if ($flags.resume -and $skipUntilRunIndex -gt -1 -and $i -lt $skipUntilRunIndex) {
            Write-Host "  Skipping step '$($run.name)' (already passed)..."
            # We still need to track the coverfile since it exists from the previous run
            $coverFiles += $run.out
            continue 
        }
        $skipUntilRunIndex = -1 # Clear checkpoint lock once we reach the correct step

        # Save Checkpoint
        "${Database}:$i" | Out-File $CHECKPOINT_FILE

        # WIPE AND RESTART DATABASE (Clean state for every step)
        if ($DockerDatabases.ContainsKey($Database)) {
            Write-Host "  --> Wiping volume and restarting $Database container for clean state..."
            docker-compose -f $DOCKER_COMPOSE_FILE stop $Database 2>&1 | Out-Null
            docker-compose -f $DOCKER_COMPOSE_FILE rm -f $Database 2>&1 | Out-Null
            docker volume rm "${dirname}_go-django_${Database}_data" -f 2>&1 | Out-Null
            docker-compose -f $DOCKER_COMPOSE_FILE up -d $Database 2>&1 | Out-Null
            
            # Wait a few seconds for the new database to accept connections
            Start-Sleep -Seconds 5
        }

        # RUN TEST
        $cmd = "go test -p=1 -tags=`"testing_auth test $Database`" -coverpkg=`"$($run.coverpkg)`" $($run.target) -coverprofile=`"$($run.out)`" --timeout=30s"
        
        if ($flags.verbose) { $cmd += " -v" }
        if ($flags.failslow -ne $true) { $cmd += " -failfast" }

        Write-Host "--- $($run.name) ---"
        Write-Host "Command: $cmd"
        
        Invoke-Expression $cmd
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Tests failed during '$($run.name)' on $Database."
            Write-Host "You can fix the error and run '.\generate-coverage.ps1 resume' to pick up from this exact spot."
            exit $LASTEXITCODE
        }
        
        # Save the file name to merge later
        $coverFiles += $run.out
    }
}

# Clear checkpoint on complete success
if (Test-Path $CHECKPOINT_FILE) { Remove-Item $CHECKPOINT_FILE }

# ---------------------------------------------------------
# FINAL REPORT MERGE
# ---------------------------------------------------------
if ($coverFiles.Count -gt 0) {
    Write-Host "========================================"
    Write-Host "Merging $($coverFiles.Count) coverage files..."
    
    # cmd.exe is used here to avoid a PowerShell file-encoding bug with the '>' operator
    $filesToMerge = $coverFiles -join " "
    $mergeCmd = "cmd.exe /c `"gocovmerge $filesToMerge > cover.all.out`""
    Invoke-Expression $mergeCmd

    Write-Host "========================================"
    Write-Host "FINAL COMBINED COVERAGE:"
    go tool cover -func="cover.all.out" | Select-String "total:"
    Write-Host "========================================"
    
    # # Clean up the individual output files to keep your directory clean
    # foreach ($file in $coverFiles) {
    #     Remove-Item -Path $file -ErrorAction SilentlyContinue
    # }
}