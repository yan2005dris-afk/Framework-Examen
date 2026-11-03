param([switch]$Faults)
$ErrorActionPreference="Stop"
$root=(Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
foreach($line in Get-Content -LiteralPath (Join-Path $root "deploy/demo/.env")){if($line -match '^([^#=]+)=(.*)$'){[Environment]::SetEnvironmentVariable($matches[1],$matches[2],"Process")}}
$env:DATABASE_URL="postgres://postgres:$($env:DEMO_DB_PASSWORD)@localhost:15432/framework_synthetic_demo?sslmode=disable"
$env:DEMO_CONFIRM="synthetic-demo";$env:DEMO_ROOT=$root;$env:GOTOOLCHAIN="auto"
if($Faults){go -C (Join-Path $root "apps/backend") run ./cmd/demo faults} else {go -C (Join-Path $root "apps/backend") run ./cmd/demo}
if($LASTEXITCODE -ne 0){throw "Demostración incompleta; estado preservado para reanudar"}
