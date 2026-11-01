param()
$ErrorActionPreference="Stop"
$root=(Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
$backend=Join-Path $root "apps/backend";$qa=Join-Path $backend "data/synthetic-demo"
foreach($line in Get-Content (Join-Path $root "deploy/demo/.env")){if($line -match '^([^#=]+)=(.*)$'){[Environment]::SetEnvironmentVariable($matches[1],$matches[2],"Process")}}
if($env:DEMO_DATASET){$qa=Join-Path $qa $env:DEMO_DATASET}
$env:TEST_DATABASE_URL="postgres://postgres:$($env:DEMO_DB_PASSWORD)@localhost:15432/framework_synthetic_demo?sslmode=disable"
$env:REDIS_TEST_URL="redis://localhost:16379/15"
$env:STORAGE_TEST_ENDPOINT="http://localhost:19000/synthetic-demo"
$env:STORAGE_TEST_ACCESS_KEY=$env:DEMO_S3_ACCESS_KEY;$env:STORAGE_TEST_SECRET_KEY=$env:DEMO_S3_SECRET_KEY
$env:S3_SIMULATION_ENDPOINT="embedded";$env:GOTOOLCHAIN="auto"
# No se modifica ningún archivo ajeno con el verificador; informar si gofmt haría cambios.
$unformatted=@(gofmt -l $backend)
if($unformatted.Count){throw "Hay archivos Go pendientes de formato"}
$results=Join-Path $qa "go-test.jsonl"
go -C $backend test ./... -count=1 -json > $results
if($LASTEXITCODE -ne 0){throw "Fallaron pruebas; revisar go-test.jsonl privado"}
go -C $backend vet ./...;if($LASTEXITCODE -ne 0){throw "Falló vet"}
go -C $backend build ./...;if($LASTEXITCODE -ne 0){throw "Falló build"}
$events=Get-Content $results|ForEach-Object {$_|ConvertFrom-Json}
$passed=@($events|Where-Object {$_.Action -eq 'pass' -and $_.Test}).Count
$skipped=@($events|Where-Object {$_.Action -eq 'skip' -and $_.Test}).Count
@{origin="synthetic-demo";warning="DATOS SINTÉTICOS - NO VÁLIDO";passed=$passed;skipped=$skipped;vet="pass";build="pass";format="pass";time=(Get-Date).ToUniversalTime().ToString("o")}|ConvertTo-Json|Set-Content (Join-Path $qa "quality.json")
Write-Host "Go: $passed pruebas aprobadas, $skipped omitidas; vet/build/formato aprobados."
