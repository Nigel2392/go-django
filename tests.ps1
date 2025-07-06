$dirname = Split-Path -Parent $MyInvocation.MyCommand.Path
$dirname = Split-Path -Leaf $dirname

$BASE_TEST_DIR = "./src/... ./queries/..."
$DOCKER_COMPOSE_FILE = "./test-databases.docker-compose.yml"

# Reset test docker containers
docker-compose -f "$DOCKER_COMPOSE_FILE" down

# Databases to test (these translate to Go build tags)
$Databases = @(
    "sqlite",
    "mysql_local",
    "mysql",
    "mariadb",
    "postgres"
)

# Databases defined in docker-compose.yml
$DockerDatabases = @{
    "mysql" = $true   
    "mariadb"  = $true
    "postgres" = $true
}

# Empty test array will be built out
$testsToRun = @(

)

# Flags for the test run
$flags = @{
    verbose = $true
    failslow = $false
}

# Check script arguments that were passed in
foreach ($arg in $args) {
    switch ($arg) {
        "silent" {
            $flags.verbose = $false
            continue
        }
        "failslow" {
            $flags.failslow = $true
            continue
        }
        "down" {
            # If the argument is "down", remove all volumes and exit
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

# Run tests for each database type
foreach ($Database in $testsToRun) {

    $cmd = "go test -tags=`"testing_auth test $Database`" --timeout=30s"
    if ($flags.verbose) {
        $cmd += " -v"
    }
    if ($flags.failslow -ne $true) {
        $cmd += " --failfast"
    }
    
    $cmd += " $BASE_TEST_DIR"

    Write-Host "Running tests for $Database"
    Write-Host "Command: $cmd"
    Write-Host "----------------------------------------"
    Invoke-Expression $cmd
    if ($LASTEXITCODE -ne 0) {
        Write-Host "----------------------------------------"
        Write-Host "Tests failed for $Database"
        exit $LASTEXITCODE
    }
    Write-Host "Tests passed for $Database"
    Write-Host "----------------------------------------"
}