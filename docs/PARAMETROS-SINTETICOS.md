# Parámetros sintéticos de demostración

Este documento define los valores reproducibles para validar el backend con datos
sintéticos. **No son reglas institucionales, legales ni datos reales**; sirven para
pruebas locales, integración, demos y fixtures de QA.

## Ruta rápida

1. Aplicar las migraciones del backend.
2. Cargar los datos sintéticos respetando los valores de este documento.
3. Crear y aprobar las versiones de reglas, banco, examen y privacidad.
4. Ejecutar el flujo registro → postulación → méritos → examen → resultados.
5. Verificar que los resultados publicados indiquen que son sintéticos.

## Convocatoria y fases

| Parámetro | Valor sintético |
|---|---|
| Identificador | `demo-2026-01` |
| Nombre | `Convocatoria sintética BIESS 2026` |
| Zona horaria | `America/Guayaquil` |
| Apertura | `2026-01-05T00:00:00-05:00` |
| Cierre | `2026-12-31T23:59:59.999999-05:00` |
| Sectores | `afiliados`, `jubilados`, `empleadores` |
| Fases | `meritos`, `psicotecnica`, `evaluacion_tecnica`, `resultados` |
| Estado inicial | `publicada` |

Las fechas son sintéticas y solo habilitan el flujo de demostración. Las ventanas
usan el intervalo `[apertura, cierre)`.

## Reglas sintéticas de méritos

Versión: `meritos-sinteticos-v1`. Referencia: `DEMO-SINTETICA-2026`.

| Criterio | Métrica | Máximo | Peso |
|---|---|---:|---:|
| Formación académica | título validado | 3000 centipuntos | 30% |
| Experiencia profesional | cada año completo | 3000 centipuntos | 30% |
| Capacitación | horas certificadas | 2000 centipuntos | 20% |
| Publicaciones | publicación validada | 1000 centipuntos | 10% |
| Requisitos e inhabilidades | checklist aprobado | 1000 centipuntos | 10% |

Reglas de cálculo:

- Máximo total: `10000` centipuntos (`100.00` puntos).
- Umbral de avance: `7000` centipuntos (`70.00` puntos).
- Redondeo: `mas_cercano`, mitades hacia arriba.
- Experiencia: `unir_intervalos`.
- Cómputo: `dias_transcurridos`.
- Requiere evidencia verificada para formación, experiencia, capacitación y
  publicaciones.
- Un resultado `avanza` habilita la siguiente fase; un resultado `excluido` no
  habilita el examen.
- Desempate sintético: mayor experiencia, luego mayor formación, luego UUID
  ascendente únicamente para estabilizar la presentación.

## Evaluación psicotécnica

| Parámetro | Valor sintético |
|---|---|
| Instrumento | `PSICO-DEMO-1` |
| Versión | `1.0` |
| Escala | `0` a `100` |
| Peso en consolidación | `30%` |
| Umbral | `60.00` |
| Evidencia | `informe-psicotecnico-sintetico.pdf` |
| Publicación | Solo después de revisión administrativa |

La nota es sintética y no representa un diagnóstico psicológico real.

## Banco y examen teórico

Banco: `banco-demo-2026-v1`.

| Parámetro | Valor sintético |
|---|---:|
| Preguntas cargadas | 20 |
| Preguntas por intento | 10 |
| Opciones por pregunta | 4 |
| Duración | 3600 segundos (60 minutos) |
| Nota máxima | 100 |
| Umbral | 70 |
| Puntaje por respuesta correcta | 10 |
| Respuesta incorrecta/no respondida | 0 |
| Intentos | `un_intento` |
| Unicidad | `orden_preguntas_unico_rechazar_agotamiento` |
| Referencia de aprobación | `DEMO-BANCO-2026` |

El banco de demostración debe contener preguntas neutrales y ficticias, sin copiar
preguntas oficiales. Las respuestas correctas solo se almacenan en la matriz
administrativa y nunca se devuelven al postulante.

## Reconsideraciones

| Parámetro | Valor sintético |
|---|---|
| Ventana del postulante | 5 días calendario desde la publicación |
| Atención de Talento Humano | 3 días calendario |
| Resolución de la Comisión | 3 días calendario |
| Máximo de solicitudes por resultado | 1 |
| Evidencia máxima por solicitud | 2 MiB |
| Estados | `recibida` → `atendida` → `resuelta` |
| Calendario | Todos los días, sin feriados excluidos |
| Referencia | `DEMO-RECONSIDERACION-2026` |

Los plazos son parámetros de prueba y no constituyen términos legales.

## Política sintética de privacidad

| Parámetro | Valor sintético |
|---|---|
| Identificador | `politica-demo-2026-v1` |
| Versión | `1.0` |
| Vigencia | 2026-01-01 a 2026-12-31 |
| Retención de expedientes sellados | 365 días |
| Retención de documentos | 365 días |
| Retención de auditoría | 730 días |
| Retención de notificaciones | 180 días |
| Retención de sesiones expiradas | 30 días |
| URLs temporales | 5 minutos |
| Lista pública | Solo nombre declarado y con consentimiento explícito |
| Eliminación | Simulada; nunca ejecutar sobre datos reales |

Texto de demostración:

> Los datos de este expediente son sintéticos y se usan exclusivamente para
> probar el sistema. La publicación de nombres requiere consentimiento explícito.
> Los documentos, resultados y notificaciones se conservan durante los períodos
> definidos en esta política de demostración.

La política sintética debe vincularse al sello mediante su identificador y hash.
No se debe presentar como aviso legal ni usar para decidir retenciones reales.

## Datos de prueba mínimos

- Tres postulantes: uno que supera méritos, uno que no los supera y uno con
  reconsideración.
- Un postulante habilitado para examen con diez respuestas correctas.
- Un postulante con examen vencido y respuestas parciales.
- Una solicitud de reconsideración en cada estado.
- Una notificación enviada, una fallida y una pendiente.
- Una lista pública con consentimiento y otra bloqueada sin consentimiento.
- Un expediente legado marcado explícitamente como `sinConvocatoria`.

Todos los nombres, cédulas, correos, documentos y UUID deben ser ficticios y
generados para la ejecución. No usar información de postulantes reales.

## Límites de uso

- Estos valores no reemplazan aprobación institucional.
- No deben sembrarse como valores por defecto para producción.
- Los fixtures deben identificar su origen como `synthetic-demo`.
- Las exportaciones, PDFs y correos de prueba deben llevar una marca visible:
  `DATOS SINTÉTICOS - NO VÁLIDO`.
- Los parámetros pueden cambiarse creando otra versión; no se deben mutar
  versiones ya usadas por resultados o exámenes.

## Aclaración confirmada para consolidación

El usuario confirmó en este chat el reparto exclusivamente sintético:
**40% méritos + 30% evaluación teórica + 30% psicotécnica**. Se conserva el umbral
psicotécnico de 60 y la escala/umbral teóricos de la tabla. No es regla institucional.
Los alias de fases del documento se adaptan al contrato existente:
`psicotecnica` → `oposicion_psicotecnica`, `evaluacion_tecnica` → `oposicion_teorica`.
Se añaden las fases técnicas `postulacion` y `reconsideracion` necesarias para
recorrer el flujo, sin cambiar el período demo-2026-01.


## Precisión operativa de los fixtures (QA, no institucional)

Para que las unidades de captura sean reproducibles, el cargador concreta la
matriz así: un título validado completa formación; cuatro bloques completos de
365 días completan experiencia (750 centipuntos ponderados por bloque); 100 horas
completan capacitación; una publicación válida completa publicaciones. El
checklist sólo puntúa cuando requisitos e inhabilidades están confirmados.
`unidadesCompletas: true` evita pagar fracciones del bloque de experiencia.
No se interpreta este bloque sintético como cómputo legal de antigüedad.

La consolidación de demostración usa escala 0–100, umbral técnico 70 y redondeo
`mas_cercano`. El umbral es una concreción del fixture, no una aprobación
institucional; el reparto 40/30/30 sí fue confirmado expresamente por el usuario.
El umbral psicotécnico 60 se configura mediante `umbralPsicotecnico` en la versión
aprobada; una conclusión admitida no puede saltarlo.

### Excepción de prueba autorizada

El usuario autorizó **un examen adicional de QA de cinco segundos** para probar
vencimientos sin esperar una hora. Se identifica por
`DEMO-QA-VENCIMIENTO-5S`, usa otra versión inmutable del banco y otra configuración.
El banco principal conserva 10 preguntas, 3600 segundos y referencia
`DEMO-BANCO-2026`. No se cambia el reloj ni el plazo de intentos ya iniciados.
La configuración corta sólo la utiliza el escenario `vencido`; no representa una
regla del concurso ni una prueba de espera real de 60 minutos.

Las reconsideraciones simultáneas abiertas se aíslan en `jubilados`; la lista de
`afiliados` se genera con sus propias reconsideraciones resueltas. Al cerrar la
fase global quedan conservadas como fixtures de consulta; para continuar su
atención se requiere reabrir la fase mediante configuración administrativa.
