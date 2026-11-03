param()
$ErrorActionPreference="Stop"
$root=(Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
$backend=Join-Path $root "apps/backend"
$vars=@{};Get-Content (Join-Path $root "deploy/demo/.env")|ForEach-Object{if($_ -match '^([^=]+)=(.*)$'){$vars[$matches[1]]=$matches[2]}}
$qa=Join-Path $backend "data/synthetic-demo";if($vars.DEMO_DATASET){if($vars.DEMO_DATASET -notmatch '^[a-z0-9-]{1,40}$'){throw "Dataset inválido"};$qa=Join-Path $qa $vars.DEMO_DATASET}
$env:TEST_DATABASE_URL="postgres://postgres:$($vars.DEMO_DB_PASSWORD)@localhost:15432/framework_synthetic_demo?sslmode=disable"
$env:REDIS_TEST_URL="redis://localhost:16379/15"
$env:RUNTIME_API_URL="http://localhost:18080";$env:RUNTIME_MAILPIT_URL="http://localhost:18025";$env:RUNTIME_ORIGIN="http://localhost:18081"
$env:GOTOOLCHAIN="auto"
go -C $backend test ./internal/exam/postgres ./internal/privacy/... ./internal/identity/postgres ./internal/document/postgres ./internal/results/postgres ./internal/platform/cache ./internal/platform/httpx ./internal/platform/mail ./internal/platform/telemetry ./internal/platform/backup -count=1 -json > (Join-Path $qa "privacy-tests.jsonl")
if($LASTEXITCODE -ne 0){throw "Falló una categoría: revisar privacy-tests.jsonl privado"}
$logs=(docker logs --since 15m framework-backend-demo-api-1 2>&1|Out-String)
if($LASTEXITCODE -ne 0){throw "No se pudieron revisar logs sintéticos"}
$violations=0
foreach($entry in $vars.GetEnumerator()){if($entry.Key -match 'PASSWORD|SECRET|KEY' -and $entry.Value.Length -ge 12 -and $logs.Contains($entry.Value)){$violations++}}
if($logs -match 'X-Amz-Signature=|Authorization: Bearer |Cookie: session=|%PDF-|[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'){$violations++}
if($violations){throw "Los logs contienen material no permitido; no se imprime su contenido"}
$tests=Get-Content (Join-Path $qa "privacy-tests.jsonl")|ForEach-Object{$_|ConvertFrom-Json}
@{origin="synthetic-demo";profile="tecnico-provisional-v1";testsPassed=@($tests|Where-Object{$_.Action -eq 'pass' -and $_.Test}).Count;testsSkipped=@($tests|Where-Object{$_.Action -eq 'skip' -and $_.Test}).Count;logViolations=$violations;categories=@('sesiones','otp','documentos','expedientes','auditoria','exportaciones','urls','nombres','notificaciones','backups','logs');legalApproval=$false;time=(Get-Date).ToUniversalTime().ToString('o')}|ConvertTo-Json|Set-Content (Join-Path $qa "privacy-evidence.json")
Write-Host "Privacidad: categorías técnicas verificadas y logs sintéticos sin secretos detectados; política legal no certificada."
