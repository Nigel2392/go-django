$dirname = Split-Path -Parent $MyInvocation.MyCommand.Path
$dirname = Split-Path -Leaf $dirname

$DOCKER_COMPOSE_FILE = "./test-databases.docker-compose.yml"

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

$upString = ""
foreach ($Database in $testsToRun) {
    # Check if the argument is a valid Docker database type
    # if it is, reset the corresponding Docker volume and start the container
    if ($DockerDatabases.ContainsKey($Database)) {
        docker volume rm "${dirname}_go-django_${Database}_data"
        $upString += " $Database"
    }
}

if ($upString -ne "") {
    # Start the Docker containers for the specified databases
    Write-Host "Starting Docker containers for:$upString"
    Invoke-Expression "docker-compose -f $DOCKER_COMPOSE_FILE up -d$upString"
} else {
    Write-Host "No Docker databases specified, skipping container start."
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
    
    Write-Host "========================================"
    Write-Host "Running tests for: $Database"
    Write-Host "========================================"

    # RUN TEST
    $file = "coverage/cover.${Database}.out"
    $cmd = "go test -p=1 -tags=`"testing_auth test $Database`" -coverpkg=`"github.com/Nigel2392/go-django/...`" ./... ./queries/... ./djester/... -coverprofile=`"$file`" --timeout=30s"
    
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
    $coverFiles += $file
}

# ---------------------------------------------------------
# FINAL REPORT MERGE
# ---------------------------------------------------------
if ($coverFiles.Count -gt 0) {
    Write-Host "========================================"
    Write-Host "Merging $($coverFiles.Count) coverage files..."
    
    # cmd.exe is used here to avoid a PowerShell file-encoding bug with the '>' operator
    $filesToMerge = $coverFiles -join " "
    $mergeCmd = "cmd.exe /c `"gocovmerge $filesToMerge > coverage/cover.all.out`""
    Invoke-Expression $mergeCmd

    Write-Host "========================================"
    Write-Host "FINAL COMBINED COVERAGE:"
    go tool cover -func="coverage/cover.all.out" | Select-String "total:"
    Write-Host "========================================"
    
    # # Clean up the individual output files to keep your directory clean
    # foreach ($file in $coverFiles) {
    #     Remove-Item -Path $file -ErrorAction SilentlyContinue
    # }
}