param()
$timer=[Diagnostics.Stopwatch]::StartNew()
$capturedAt=(Get-Date).ToUniversalTime()
$ErrorActionPreference="Stop"
$root=(Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
$backend=Join-Path $root "apps/backend"
$qa=Join-Path $backend "data/synthetic-demo"
$stamp=Get-Date -Format "yyyyMMddHHmmss"
$backup=Join-Path $qa "backup-$stamp"

$vars=@{};Get-Content (Join-Path $root "deploy/demo/.env")|ForEach-Object {if($_ -match '^([^=]+)=(.*)$'){$vars[$Matches[1]]=$Matches[2]}}
$env:DEMO_CONFIRM="synthetic-demo";$env:GOTOOLCHAIN="auto"
if(-not $vars.BACKUP_ENCRYPTION_KEY){
 $vars.BACKUP_ENCRYPTION_KEY=[Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
 Add-Content -LiteralPath (Join-Path $root "deploy/demo/.env") -Value "BACKUP_ENCRYPTION_KEY=$($vars.BACKUP_ENCRYPTION_KEY)"
}
$env:BACKUP_ENCRYPTION_KEY=$vars.BACKUP_ENCRYPTION_KEY
$env:MAIL_OUTBOX_KEY=$vars.MAIL_OUTBOX_KEY
$env:DEMO_S3_ACCESS_KEY=$vars.DEMO_S3_ACCESS_KEY;$env:DEMO_S3_SECRET_KEY=$vars.DEMO_S3_SECRET_KEY
if($vars.DEMO_DATASET){
 if($vars.DEMO_DATASET -notmatch '^[a-z0-9-]{1,40}$'){throw "Dataset inválido"}
 $qa=Join-Path $qa $vars.DEMO_DATASET;$backup=Join-Path $qa "backup-$stamp"
}
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
 @("MAIL_OUTBOX_KEY=$($vars.MAIL_OUTBOX_KEY)","DEMO_S3_ACCESS_KEY=$($vars.DEMO_S3_ACCESS_KEY)","DEMO_S3_SECRET_KEY=$($vars.DEMO_S3_SECRET_KEY)")|Set-Content -LiteralPath (Join-Path $backup "recovery.env")
 $restoreFiles=Join-Path $backup "private-restore"
 New-Item -ItemType Directory -Path $restoreFiles | Out-Null
 foreach($file in @("postgres.dump","objects.tar","recovery.env")){
   go -C $backend run ./cmd/backup-crypt encrypt (Join-Path $backup $file) (Join-Path $backup "$file.enc");Assert-Native
   go -C $backend run ./cmd/backup-crypt decrypt (Join-Path $backup "$file.enc") (Join-Path $restoreFiles $file);Assert-Native
   if((Get-FileHash (Join-Path $backup $file)).Hash -cne (Get-FileHash (Join-Path $restoreFiles $file)).Hash){throw "Falló integridad de descifrado"}
 }
 # Restaurar exclusivamente desde los archivos descifrados, no del original aún disponible.
 docker cp (Join-Path $restoreFiles "postgres.dump") framework-backend-demo-postgres-1:/tmp/synthetic-demo-restore.dump;Assert-Native
 $db="synthetic_restore_$stamp";$volume="synthetic-restore-$stamp";$container="synthetic-restore-$stamp"
 docker exec framework-backend-demo-postgres-1 createdb -U postgres $db;Assert-Native
 docker exec framework-backend-demo-postgres-1 pg_restore -U postgres -d $db --exit-on-error /tmp/synthetic-demo-restore.dump;Assert-Native
 docker volume create --label data.origin=synthetic-demo $volume | Out-Null;Assert-Native
 docker run --rm -v "${volume}:/objects" -v "${restoreFiles}:/backup:ro" postgres:18-alpine tar -C /objects -xf /backup/objects.tar;Assert-Native
 # Secretos por archivo ignorado; nunca mediante argumentos visibles de docker.
 $runtimeEnv=Join-Path $backup "rustfs.env"
 @("RUSTFS_VOLUMES=/data","RUSTFS_ACCESS_KEY=$($vars.DEMO_S3_ACCESS_KEY)","RUSTFS_SECRET_KEY=$($vars.DEMO_S3_SECRET_KEY)","RUSTFS_ADDRESS=0.0.0.0:9000","RUSTFS_CONSOLE_ENABLE=false")|Set-Content $runtimeEnv
 docker run -d --name $container --label data.origin=synthetic-demo --env-file $runtimeEnv -p 127.0.0.1:19010:9000 -v "${volume}:/data" rustfs/rustfs:latest | Out-Null;Assert-Native
 Start-Sleep -Seconds 8
 Verify-Demo "postgres://postgres:$($vars.DEMO_DB_PASSWORD)@localhost:15432/${db}?sslmode=disable" "http://localhost:19010/synthetic-demo" (Join-Path $backup "restored.json")
 $before=Get-Content (Join-Path $backup "before.json") -Raw;$after=Get-Content (Join-Path $backup "restored.json") -Raw
 if($before -cne $after){throw "Difieren los manifiestos originales/restaurados"}
 # Después de comparar el snapshot íntegro, invalidar las sesiones restauradas.
 # Una revocación posterior a la captura no puede reconstruirse desde un backup antiguo.
 docker exec framework-backend-demo-postgres-1 psql -U postgres -d $db -v ON_ERROR_STOP=1 -c "DELETE FROM sessions" | Out-Null;Assert-Native
 $remaining=docker exec framework-backend-demo-postgres-1 psql -U postgres -d $db -Atc "SELECT count(*) FROM sessions";Assert-Native
 if($remaining -ne '0'){throw "La restauración no invalidó sesiones"}
 @("origin=synthetic-demo","warning=DATOS SINTÉTICOS - NO VÁLIDO","restoredDatabase=$db","restoredContainer=$container","restoredVolume=$volume","status=verified")|Set-Content (Join-Path $backup "restore-evidence.txt")
 Get-FileHash (Join-Path $backup "postgres.dump.enc"),(Join-Path $backup "objects.tar.enc"),(Join-Path $backup "recovery.env.enc") -Algorithm SHA256 | Select-Object @{n="file";e={Split-Path $_.Path -Leaf}},Hash | ConvertTo-Json | Set-Content (Join-Path $backup "archive-sha256.json")
 $timer.Stop()
 $binaryHash=(Get-FileHash (Join-Path $backend "data/synthetic-demo/api") -Algorithm SHA256).Hash
 $revision=git -C $root rev-parse HEAD
 $dirty=[bool](git -C $root status --porcelain)

 @{origin="synthetic-demo";profile="tecnico-provisional-v1";revision=$revision;workingTreeDirty=$dirty;apiBinarySha256=$binaryHash;capturedAt=$capturedAt.ToString("o");recoveredAt=(Get-Date).ToUniversalTime().ToString("o");rtoSeconds=[math]::Round($timer.Elapsed.TotalSeconds,3);rtoTargetSeconds=1800;rpoTargetSeconds=86400;scheduledRpoMeasured=$false;encryption="AES-256-GCM por bloques, subclave HMAC-SHA256 por archivo";restoreFromEncrypted=$true;restoredSessionsInvalidated=$true;status="verified"}|ConvertTo-Json|Set-Content (Join-Path $backup "recovery-timing.json")
 # Borrar sólo archivos temporales creados por este simulacro tras autenticar y restaurar.
 $resolvedBackup=(Resolve-Path -LiteralPath $backup).Path
 $qaRoot=(Resolve-Path -LiteralPath $qa).Path
 if(-not $resolvedBackup.StartsWith($qaRoot+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw "Respaldo fuera de QA"}
 foreach($file in @("postgres.dump","objects.tar","recovery.env","rustfs.env")){
  Remove-Item -LiteralPath (Join-Path $backup $file)
 }
 foreach($file in @("postgres.dump","objects.tar","recovery.env")){Remove-Item -LiteralPath (Join-Path $restoreFiles $file)}
 Remove-Item -LiteralPath $restoreFiles
 docker exec framework-backend-demo-postgres-1 rm -f /tmp/synthetic-demo.dump /tmp/synthetic-demo-restore.dump;Assert-Native
 Write-Host "Backup cifrado y restauración aislada verificados. Evidencia privada: $backup"
} finally {docker start framework-backend-demo-api-1 | Out-Null}
