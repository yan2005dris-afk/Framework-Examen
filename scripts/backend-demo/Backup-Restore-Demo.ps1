param()
$ErrorActionPreference="Stop"
$root=(Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
$backend=Join-Path $root "apps/backend"
$qa=Join-Path $backend "data/synthetic-demo"
$stamp=Get-Date -Format "yyyyMMddHHmmss"
$backup=Join-Path $qa "backup-$stamp"

$vars=@{};Get-Content (Join-Path $root "deploy/demo/.env")|ForEach-Object {if($_ -match '^([^=]+)=(.*)$'){$vars[$Matches[1]]=$Matches[2]}}
$env:DEMO_CONFIRM="synthetic-demo";$env:GOTOOLCHAIN="auto"
$env:MAIL_OUTBOX_KEY=$vars.MAIL_OUTBOX_KEY
$env:DEMO_S3_ACCESS_KEY=$vars.DEMO_S3_ACCESS_KEY;$env:DEMO_S3_SECRET_KEY=$vars.DEMO_S3_SECRET_KEY
if($vars.DEMO_DATASET){$qa=Join-Path $qa $vars.DEMO_DATASET;$backup=Join-Path $qa "backup-$stamp"}
New-Item -ItemType Directory -Path $backup | Out-Null
$objectsVolume="framework-backend-demo_objects";if($vars.DEMO_OBJECT_VOLUME){$objectsVolume=$vars.DEMO_OBJECT_VOLUME}
$source="postgres://postgres:$($vars.DEMO_DB_PASSWORD)@localhost:15432/framework_synthetic_demo?sslmode=disable"
function Assert-Native {if($LASTEXITCODE -ne 0){throw "Falló un paso de backup/restauración; se preservan los archivos"}}
function Verify-Demo($database,$endpoint,$file){$env:DATABASE_URL=$database;$env:VERIFY_S3_ENDPOINT=$endpoint;$env:VERIFY_OUTPUT=$file;go -C $backend run ./cmd/demo-verify;Assert-Native}
# Congelar sólo la API sintética durante el snapshot, no las bases ni el frontend compartido.
docker stop framework-backend-demo-api-1 | Out-Null;Assert-Native
try {
 Verify-Demo $source "http://localhost:19000/synthetic-demo" (Join-Path $backup "before.json")
 docker exec framework-backend-demo-postgres-1 pg_dump -U postgres -d framework_synthetic_demo -Fc -f /tmp/synthetic-demo.dump;Assert-Native
 docker cp framework-backend-demo-postgres-1:/tmp/synthetic-demo.dump (Join-Path $backup "postgres.dump");Assert-Native
 docker stop framework-backend-demo-rustfs-1 | Out-Null;Assert-Native
 try {docker run --rm -v "${objectsVolume}:/objects:ro" -v "${backup}:/backup" postgres:18-alpine tar -C /objects -cf /backup/objects.tar .;Assert-Native}
 finally {docker start framework-backend-demo-rustfs-1 | Out-Null}
 # La clave de outbox es necesaria para recuperar las cargas cifradas. Copia privada, nunca versionable.
 Copy-Item -LiteralPath (Join-Path $root "deploy/demo/.env") -Destination (Join-Path $backup "recovery.env")
 $db="synthetic_restore_$stamp";$volume="synthetic-restore-$stamp";$container="synthetic-restore-$stamp"
 docker exec framework-backend-demo-postgres-1 createdb -U postgres $db;Assert-Native
 docker exec framework-backend-demo-postgres-1 pg_restore -U postgres -d $db --exit-on-error /tmp/synthetic-demo.dump;Assert-Native
 docker volume create --label data.origin=synthetic-demo $volume | Out-Null;Assert-Native
 docker run --rm -v "${volume}:/objects" -v "${backup}:/backup:ro" postgres:18-alpine tar -C /objects -xf /backup/objects.tar;Assert-Native
 # Secretos por archivo ignorado; nunca mediante argumentos visibles de docker.
 $runtimeEnv=Join-Path $backup "rustfs.env"
 @("RUSTFS_VOLUMES=/data","RUSTFS_ACCESS_KEY=$($vars.DEMO_S3_ACCESS_KEY)","RUSTFS_SECRET_KEY=$($vars.DEMO_S3_SECRET_KEY)","RUSTFS_ADDRESS=0.0.0.0:9000","RUSTFS_CONSOLE_ENABLE=false")|Set-Content $runtimeEnv
 docker run -d --name $container --label data.origin=synthetic-demo --env-file $runtimeEnv -p 127.0.0.1:19010:9000 -v "${volume}:/data" rustfs/rustfs:latest | Out-Null;Assert-Native
 Start-Sleep -Seconds 8
 Verify-Demo "postgres://postgres:$($vars.DEMO_DB_PASSWORD)@localhost:15432/${db}?sslmode=disable" "http://localhost:19010/synthetic-demo" (Join-Path $backup "restored.json")
 $before=Get-Content (Join-Path $backup "before.json") -Raw;$after=Get-Content (Join-Path $backup "restored.json") -Raw
 if($before -cne $after){throw "Difieren los manifiestos originales/restaurados"}
 @("origin=synthetic-demo","warning=DATOS SINTÉTICOS - NO VÁLIDO","restoredDatabase=$db","restoredContainer=$container","restoredVolume=$volume","status=verified")|Set-Content (Join-Path $backup "restore-evidence.txt")
 Get-FileHash (Join-Path $backup "postgres.dump"),(Join-Path $backup "objects.tar") -Algorithm SHA256 | Select-Object @{n="file";e={Split-Path $_.Path -Leaf}},Hash | ConvertTo-Json | Set-Content (Join-Path $backup "archive-sha256.json")
 Write-Host "Backup y restauración aislada verificados. Evidencia privada: $backup"
} finally {docker start framework-backend-demo-api-1 | Out-Null}
