param([switch]$NoBuild)
$ErrorActionPreference = "Stop"
$root = (Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
$backend = Join-Path $root "apps/backend"
$demo = Join-Path $backend "data/synthetic-demo"
New-Item -ItemType Directory -Force -Path $demo | Out-Null
$envFile = Join-Path $root "deploy/demo/.env"
if (-not (Test-Path -LiteralPath $envFile)) {
  function New-DemoSecret { [Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(24)).ToLowerInvariant() }
  $key = [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
  @("DEMO_DB_PASSWORD=$(New-DemoSecret)", "DEMO_S3_ACCESS_KEY=syntheticdemo", "DEMO_S3_SECRET_KEY=$(New-DemoSecret)", "MAIL_OUTBOX_KEY=$key", "DEMO_DATASET=demo", "DEMO_POSTGRES_VOLUME=framework-backend-demo_postgres_demo", "DEMO_OBJECT_VOLUME=framework-backend-demo_objects_demo", "DEMO_FAST_EXPIRY=1") | Set-Content -LiteralPath $envFile -Encoding utf8
}
$values=@{};Get-Content $envFile|ForEach-Object {if($_ -match '^([^#=]+)=(.*)$'){$values[$matches[1]]=$matches[2]}}
foreach($entry in $values.GetEnumerator()){[Environment]::SetEnvironmentVariable($entry.Key,$entry.Value,"Process")}
$dataset=$demo
if($values.DEMO_DATASET){if($values.DEMO_DATASET -notmatch '^[a-z0-9-]{1,40}$'){throw "Dataset inválido"};$dataset=Join-Path $demo $values.DEMO_DATASET}
New-Item -ItemType Directory -Force -Path $dataset | Out-Null
if (-not (Test-Path -LiteralPath (Join-Path $dataset "eligibility.json"))) { '{}' | Set-Content -LiteralPath (Join-Path $dataset "eligibility.json") -Encoding utf8 }
if (-not $NoBuild) {
  $oldOS,$oldArch,$oldCGO = $env:GOOS,$env:GOARCH,$env:CGO_ENABLED
  try {
    $env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; $env:GOTOOLCHAIN="auto"
    go -C $backend build -trimpath -o (Join-Path $demo "api") ./cmd/api
    if ($LASTEXITCODE -ne 0) { throw "No se pudo compilar la API" }
    go -C $backend build -trimpath -o (Join-Path $demo "migrate") ./cmd/migrate
    if ($LASTEXITCODE -ne 0) { throw "No se pudo compilar migraciones" }
  } finally { $env:GOOS=$oldOS; $env:GOARCH=$oldArch; $env:CGO_ENABLED=$oldCGO }
}
docker image inspect framework-examen-api:latest *> $null
if ($LASTEXITCODE -ne 0) { throw "Se requiere la imagen local framework-examen-api:latest; construir con docker compose build api" }
$compose = Join-Path $root "deploy/demo/compose.yaml"
$running= docker ps --filter name=framework-backend-demo-api-1 --format '{{.Names}}'
if($running -contains 'framework-backend-demo-api-1'){docker stop framework-backend-demo-api-1 | Out-Null}
docker compose --env-file $envFile -f $compose up -d postgres redis rustfs mailpit
if($LASTEXITCODE -ne 0){throw "No se pudieron preparar las dependencias sintéticas"}
$env:DEMO_CONFIRM="synthetic-demo";$env:DATABASE_URL="postgres://postgres:$($values.DEMO_DB_PASSWORD)@localhost:15432/framework_synthetic_demo?sslmode=disable"
for($n=0;$n -lt 30;$n++){docker exec framework-backend-demo-postgres-1 pg_isready -h 127.0.0.1 -U postgres -d framework_synthetic_demo *> $null;if($LASTEXITCODE -eq 0){break};Start-Sleep 1}
go -C $backend run ./cmd/demo-bootstrap
if($LASTEXITCODE -ne 0){throw "Bootstrap sintético incompleto"}
docker compose --env-file $envFile -f $compose up -d --build
if ($LASTEXITCODE -ne 0) { throw "No se pudo iniciar el stack sintético" }
Write-Host "Stack synthetic-demo: API http://localhost:18080 ; proxy http://localhost:18081 ; Mailpit http://localhost:18025"

# El contrato real S3 aprovisiona y comprueba únicamente el bucket de QA.
$env:STORAGE_TEST_ENDPOINT="http://localhost:19000/synthetic-demo"
$env:STORAGE_TEST_ACCESS_KEY=$values.DEMO_S3_ACCESS_KEY;$env:STORAGE_TEST_SECRET_KEY=$values.DEMO_S3_SECRET_KEY
$env:GOTOOLCHAIN="auto"
go -C $backend test ./internal/platform/storage -run '^TestS3RuntimeRoundTrip$' -count=1
if($LASTEXITCODE -ne 0){throw "RustFS sintético no pasó su aprovisionamiento y round-trip"}
