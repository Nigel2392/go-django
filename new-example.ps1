param(
    [Parameter(Mandatory = $true)]
    [string]$Name
)

$WHERE = "./examples"
$GOMODULE = "github.com/Nigel2392/go-django/examples/$Name"

Invoke-Expression "go-django startproject -d $WHERE/$Name -m $GOMODULE $Name"
Invoke-Expression "go work use $WHERE/$Name"