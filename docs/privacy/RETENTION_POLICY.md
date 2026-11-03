# Perfil provisional de conservación y eliminación

**Ámbito:** desarrollo y demostraciones sintéticas; confirmado por el usuario el
2026-11-01. No sustituye aprobación institucional ni fija plazos legales. No se
activa una purga de expedientes reales por añadir este documento.

## Reglas por categoría

| Categoría | Plazo del perfil | Acción al vencer / excepción | Evidencia verificable |
|---|---|---|---|
| Sesiones | TTL de sesión por entorno; conservar expiradas 30 días | Logout/login/reset revocan; purga de expiradas >30 días en lotes de 1000/h. Una restauración invalida todas las sesiones antes de abrir tráfico | `TestAuthenticatedSessionRotationIsAtomic`, `TestPurgeSessionsPreservesActiveAndRecentExpired`, manifiesto de restauración |
| OTP | 10 min, tres intentos, tres emisiones por cédula y correo/finalidad | Redis expira el desafío; no recuperar OTP desde backups. No reintentos automáticos de scripts con resultado ambiguo | `TestRedisSecurityIntegrationTTLAndAtomicity`, `TestEmailSendQuotaSurvivesDifferentSubjectsAndSeparatesPurposes` |
| Borradores / documentos | Revisión a 365 días en demo | Sin eliminación automática por antigüedad. Baja autorizada de borrador retira metadatos/referencias y encola objetos; el historial sobrevive | `TestPostgresSealBlocksDocumentsAndCascadesQueueObjects` |
| Expedientes sellados / resultados / PDFs | Ventanas de acceso de la versión aceptada, demo 365 días; revisión humana | Expiración restringe salida, no borra sello ni calificaciones. Cualquier reclamación, orden de conservación o incompatibilidad impide purga hasta decisión documentada | `TestPolicyChannelWindowsAndRestrictions`, `TestPrivacyVersionBindingAndAuditRollback` |
| Auditoría | Mínimo técnico provisional 730 días; sin purga automática | Append-only. A los 730 días se revisa; no eliminar mientras exista expediente/reclamación, investigación o excepción pendiente | `TestStateJournalCapturesActorBeforeAfterAndRollsBackOnFailure` |
| Exportaciones | No persistencia en navegador; ventana de acceso del expediente, demo 365 días | Blob/URL de descarga sólo en memoria y revocada al terminar; archivo descargado queda bajo custodia del operador, no es revocable por la API | Pruebas de cliente `api.test.ts` y `vistas-revision.test.ts`; revisión del exportador |
| URLs firmadas | 5 min; emisión autorizada y no-store | Bloqueo impide **nuevas** URLs. La URL opaca vuelve a verificar sesión, rol, propiedad y versión de privacidad en cada GET/HEAD; tras bloqueo o logout se rechaza de inmediato y no revive al reabrir. No se entrega una firma S3 al navegador | `TestDownloadsRevokeIssuedCapabilitiesImmediately`, comprobación real con RustFS y proxy del cargador demo |
| Nombres públicos | Consentimiento literal sellado + ventana de política, demo 365 días | Restricción bloquea proyección; no revela cédula, notas, contacto o documentos. Copias hechas por terceros no se pueden retirar desde la API | `TestResultListsVersionedPublicationPrivacyAndRollback` |
| Notificaciones | Ventana de política demo 180 días; carga cifrada de trabajos expira operativamente antes | OTP/cargas aceptadas/superadas/vencidas se vacían según outbox; conservar historial de estados. Mailpit es QA y requiere mantenimiento separado; no es un archivo institucional | `TestPostgresOutboxIsTransactionalRetryableAndExpiresOTP`, `TestPostgresOutboxSMTPRealTestServer` |
| Respaldos | Captura diaria, siete días de versiones verificadas | Cifrado autenticado y clave separada; purga sólo tras revisión de hold, copia recuperable posterior y autorización. No se ejecuta purga histórica automática | `Backup-Restore-Demo.ps1`, `recovery-timing.json`, pruebas de cifrado |
| Logs/métricas | Logs de diagnóstico 7 días en perfil; métricas agregadas 30 días propuestos para el colector | Sin OTP, contraseñas, cookie, contenido de PDF, correo, cédula o nombre. Exportación de logs sólo sanitizada | `TestLogsUseRouteTemplateAndSafeRequestID`, `TestQueueGaugesRejectUnknownOrSensitiveLabels`, `Test-Privacy.ps1` |

Los plazos de política son **ventanas y revisión**, no autorización para ejecutar
DELETE al vencer. El código no tiene un reloj legal ni un motor de decisiones
jurídicas. Estos datos deben ser aprobados/versionados antes de uso real.

## Eliminación lógica, física y excepciones

1. **Restricción lógica:** administrador vigente registra motivo/versionado en
   privacidad. Se bloquean accesos, capacidades ya emitidas y nuevas capacidades por canal; el sello y la
   auditoría quedan intactos. No se inventa consentimiento de registros legados.
2. **Baja física ordinaria:** sólo un borrador editable permite eliminar documento
   o postulación. La transacción retira referencias y crea una tarea durable; el
   worker borra el objeto idempotentemente y confirma la tarea. Si storage cae se
   conserva el trabajo. No se considera borrado físico hasta esa confirmación.
3. **Expediente sellado:** sus restricciones de inmutabilidad se mantienen. Una
   disposición extraordinaria exige plan/migración revisada, inventario de
   dependencias, autoridad, motivo y evidencia separada, nunca desactivar triggers
   desde un endpoint. No existe endpoint de purga masiva de sellos.
4. **Hold/excepción:** cualquier reclamación abierta, investigación, orden de
   conservación, restauración no validada o duda sobre el plazo suspende la
   eliminación. Registrar identificador del caso, categoría, motivo, alcance,
   responsable, fecha y próxima revisión en el expediente de operación protegido.
   La política conservadora por defecto ya evita eliminar sellos/historial.
5. **Backups:** un hold alcanza las copias pertinentes. Si se restaura una copia
   antigua, aplicar restricciones vigentes antes de abrir tráfico y no enviar
   outbox a SMTP real hasta revisión. Invalidar sesiones restauradas evita
   resucitar revocaciones posteriores al punto de captura.

## Checklist de una decisión de disposición

- [ ] Categoría, política/versión/hash y motivo identificados.
- [ ] Autoridad institucional y responsable constan por escrito (para datos reales).
- [ ] No hay hold ni reclamación pendiente; revisar fuentes definitivas y referencias.
- [ ] Existe respaldo autenticado recuperable y se conoce qué copias quedarían.
- [ ] Ensayo en datos sintéticos, alcance y número de registros revisados.
- [ ] Separar restricción, baja de metadatos y confirmación de objeto físico.
- [ ] Mantener evidencia de autorización y ejecución sin duplicar datos personales.
- [ ] Registrar fallos/reintentos y verificar que no se borró evidencia protegida.

## Ejecutar la verificación por categorías

```powershell
./scripts/cross-cutting/Test-Privacy.ps1
```

La prueba no destruye expedientes ni respalda datos reales. Usa esquemas nuevos y
cuentas sintéticas; genera un informe agregado privado. Un resultado verde no
certifica la política legal, la eliminación de copias descargadas por terceros ni
la retirada de bytes ya descargados.


### Decisión confirmada: revocación inmediata

El usuario rechazó aceptar cinco minutos de vigencia tras revocar. La migración
00020 añade capacidades opacas ligadas a la sesión y versión de privacidad. Un
listener separado sirve el PDF privado sólo tras comprobarlas y auditar el acceso.
Comprueba de nuevo después de recuperar bytes de S3; nunca redirige ni entrega la
firma. No cachear estas respuestas en navegador/CDN ni registrar los tokens en
logs del proxy. Ocho descargas concurrentes y PDFs de máximo 4 MiB acotan memoria.
Se rechazan rangos parciales para no crear caminos sin control.

La revocación impide solicitudes nuevas; no borra archivos o bytes ya entregados
ni garantiza retirar una transferencia que ya empezó a enviarlos. Al migrar un
entorno que emitía firmas S3 directas, **rotar las credenciales de firma antiguas**
o mantenerlo sin tráfico hasta su vencimiento antes de declarar la nueva garantía.
El código nuevo no puede invalidar una firma antigua que ya conoce el navegador.
