-- Banco de preguntas y configuraciones oficiales para los 3 sectores (EF-14)
BEGIN;

-- ------------------------------------------------------------
-- Sector: afiliados
-- ------------------------------------------------------------
INSERT INTO exam_bank_streams (competition_id, sector, version)
VALUES ((SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'), 'afiliados', 1)
ON CONFLICT (competition_id, sector) DO UPDATE SET version = EXCLUDED.version;

INSERT INTO exam_banks (id, competition_id, sector, version, definition, hash, actor_id, created_at)
VALUES (
    '01a19fb4-7d9f-7d55-ae59-4da33f1fa0c3'::uuid,
    (SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'),
    'afiliados',
    1,
    '{"versionAnterior":0,"preguntas":[{"id":"af-01","enunciado":"¿Cuál es la máxima autoridad de gobierno del Banco del Instituto Ecuatoriano de Seguridad Social (BIESS)?","opciones":[{"id":"af-01-a","texto":"La Superintendencia de Bancos"},{"id":"af-01-b","texto":"El Directorio del BIESS"},{"id":"af-01-c","texto":"El Ministerio de Economía y Finanzas"},{"id":"af-01-d","texto":"La Asamblea Nacional"}],"correcta":"af-01-b"},{"id":"af-02","enunciado":"¿Quiénes conforman la representación de los afiliados en el Directorio del BIESS según la normativa vigente?","opciones":[{"id":"af-02-a","texto":"Un Director Principal y un Director Alterno"},{"id":"af-02-b","texto":"Dos Directores Principales sin alternos"},{"id":"af-02-c","texto":"Un representante designado directamente por el Ejecutivo"},{"id":"af-02-d","texto":"Tres vocales de elección provincial"}],"correcta":"af-02-a"},{"id":"af-03","enunciado":"Los fondos previsionales administrados por el BIESS deben invertirse primordialmente bajo qué criterios técnicos:","opciones":[{"id":"af-03-a","texto":"Especulación financiera y endeudamiento subordinado"},{"id":"af-03-b","texto":"Seguridad, solvencia, liquidez y rentabilidad"},{"id":"af-03-c","texto":"Exclusivamente en títulos de deuda pública soberana sin rendimiento"},{"id":"af-03-d","texto":"Financiamiento de gasto corriente de la seguridad social"}],"correcta":"af-03-b"},{"id":"af-04","enunciado":"¿Qué modalidad de crédito otorga el BIESS a los afiliados para el fomento directo de la vivienda propia?","opciones":[{"id":"af-04-a","texto":"Créditos quirografarios de emergencia"},{"id":"af-04-b","texto":"Créditos hipotecarios"},{"id":"af-04-c","texto":"Créditos prendarios de montepío"},{"id":"af-04-d","texto":"Microcréditos comerciales minoristas"}],"correcta":"af-04-b"},{"id":"af-05","enunciado":"De conformidad con la Ley del BIESS, las resoluciones del Directorio requieren para su validez y aprobación:","opciones":[{"id":"af-05-a","texto":"Consenso unánime de todos los miembros titulares"},{"id":"af-05-b","texto":"Mayoría absoluta de votos de los miembros presentes"},{"id":"af-05-c","texto":"Aprobación previa y vinculante de la Contraloría General del Estado"},{"id":"af-05-d","texto":"Dictamen favorable del Ministerio de Trabajo"}],"correcta":"af-05-b"}],"referencia":"Resolución No. C.D. 701 - Banco de Preguntas Sector Afiliados BIESS 2026"}'::jsonb,
    'ca64e3f618bb7edd28de41d62fa0bd9799751ac0427047c9e82eed5ca9c7ee5a',
    (SELECT id FROM applicants WHERE cedula = '0910034008'),
    now()
)
ON CONFLICT (competition_id, sector, version) DO NOTHING;

INSERT INTO exam_configs (id, bank_id, competition_id, sector, definition, hash, actor_id, created_at)
VALUES (
    '01a19fb4-7d9f-7d56-8afd-d8f54c29af4c'::uuid,
    '01a19fb4-7d9f-7d55-ae59-4da33f1fa0c3'::uuid,
    (SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'),
    'afiliados',
    '{"cantidadPreguntas":3,"duracionSegundos":1800,"maximo":100,"umbral":70,"redondeo":"inferior","politicaIntentos":"un_intento","politicaUnicidad":"orden_preguntas_unico_rechazar_agotamiento","referenciaAprobacion":"Aprobación Comisión de Selección - Reglas Sector Afiliados 2026","abreEn":"2026-01-05T05:00:00Z","cierraEn":"2027-01-01T04:59:59Z"}'::jsonb,
    'a29918b1ec2cdaf0f250a78b17d4f3e8c24d548dbb68bb05c97b39c08b9cb1c2',
    (SELECT id FROM applicants WHERE cedula = '0910034008'),
    now()
)
ON CONFLICT (bank_id) DO NOTHING;

-- ------------------------------------------------------------
-- Sector: jubilados
-- ------------------------------------------------------------
INSERT INTO exam_bank_streams (competition_id, sector, version)
VALUES ((SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'), 'jubilados', 1)
ON CONFLICT (competition_id, sector) DO UPDATE SET version = EXCLUDED.version;

INSERT INTO exam_banks (id, competition_id, sector, version, definition, hash, actor_id, created_at)
VALUES (
    '01a19fb4-7d9f-7d57-b830-efe047dab640'::uuid,
    (SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'),
    'jubilados',
    1,
    '{"versionAnterior":0,"preguntas":[{"id":"ju-01","enunciado":"¿Cuál es el objeto primordial de las inversiones que realiza el BIESS con los recursos del Fondo de Pensiones?","opciones":[{"id":"ju-01-a","texto":"Financiar el presupuesto general del Estado sin generar rendimiento"},{"id":"ju-01-b","texto":"Preservar el valor patrimonial y generar rendimientos que aseguren el pago oportuno de las pensiones"},{"id":"ju-01-c","texto":"Otorgar subsidios a corporaciones privadas en liquidación"},{"id":"ju-01-d","texto":"Cubrir el déficit operativo de ministerios del sector social"}],"correcta":"ju-01-b"},{"id":"ju-02","enunciado":"¿Qué servicio crediticio ofrece el BIESS a los pensionistas y jubilados garantizado con sus pensiones?","opciones":[{"id":"ju-02-a","texto":"Préstamos Quirografarios"},{"id":"ju-02-b","texto":"Líneas de crédito rotativo internacional"},{"id":"ju-02-c","texto":"Leasing de bienes corporativos"},{"id":"ju-02-d","texto":"Participaciones en fondos de capital de riesgo"}],"correcta":"ju-02-a"},{"id":"ju-03","enunciado":"¿Qué organismo técnico ejerce el control y supervisión directa de las operaciones financieras del BIESS?","opciones":[{"id":"ju-03-a","texto":"El Consejo de Participación Ciudadana y Control Social"},{"id":"ju-03-b","texto":"La Superintendencia de Bancos"},{"id":"ju-03-c","texto":"El Banco Central del Ecuador"},{"id":"ju-03-d","texto":"La Defensoría del Pueblo"}],"correcta":"ju-03-b"},{"id":"ju-04","enunciado":"Según la Constitución y la normativa de seguridad social, las pensiones jubilares:","opciones":[{"id":"ju-04-a","texto":"Pueden ser objeto de embargo por deudas mercantiles generales"},{"id":"ju-04-b","texto":"Son inembargables, salvo por obligaciones de alimentos o créditos con el IESS/BIESS"},{"id":"ju-04-c","texto":"Se gravan con el impuesto a la renta en su tarifa máxima"},{"id":"ju-04-d","texto":"Quedan congeladas de forma definitiva sin indexación anual"}],"correcta":"ju-04-b"},{"id":"ju-05","enunciado":"Los postulantes a representantes de los jubilados ante el Directorio del BIESS deben:","opciones":[{"id":"ju-05-a","texto":"Acreditar la condición legal de jubilado y no hallarse inmerso en causales de inhabilidad"},{"id":"ju-05-b","texto":"Contar con menos de 45 años de edad a la fecha de postulación"},{"id":"ju-05-c","texto":"Haber ejercido cargos de dirección en entidades bancarias extranjeras exclusivamente"},{"id":"ju-05-d","texto":"Estar en relación de dependencia con el sector público ministerial"}],"correcta":"ju-05-a"}],"referencia":"Resolución No. C.D. 701 - Banco de Preguntas Sector Jubilados BIESS 2026"}'::jsonb,
    '8063675ce8fc3aaa1b08a90a400e65a536dba980f03898f7d6c28b4398fa8aa5',
    (SELECT id FROM applicants WHERE cedula = '0910034008'),
    now()
)
ON CONFLICT (competition_id, sector, version) DO NOTHING;

INSERT INTO exam_configs (id, bank_id, competition_id, sector, definition, hash, actor_id, created_at)
VALUES (
    '01a19fb4-7d9f-7d58-a125-c94a4b2c08df'::uuid,
    '01a19fb4-7d9f-7d57-b830-efe047dab640'::uuid,
    (SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'),
    'jubilados',
    '{"cantidadPreguntas":3,"duracionSegundos":1800,"maximo":100,"umbral":70,"redondeo":"inferior","politicaIntentos":"un_intento","politicaUnicidad":"orden_preguntas_unico_rechazar_agotamiento","referenciaAprobacion":"Aprobación Comisión de Selección - Reglas Sector Jubilados 2026","abreEn":"2026-01-05T05:00:00Z","cierraEn":"2027-01-01T04:59:59Z"}'::jsonb,
    'a77b98e8632cdde9bf83f88d34012ba0e0d58230195155d9ff18168017dee841',
    (SELECT id FROM applicants WHERE cedula = '0910034008'),
    now()
)
ON CONFLICT (bank_id) DO NOTHING;

-- ------------------------------------------------------------
-- Sector: empleadores
-- ------------------------------------------------------------
INSERT INTO exam_bank_streams (competition_id, sector, version)
VALUES ((SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'), 'empleadores', 1)
ON CONFLICT (competition_id, sector) DO UPDATE SET version = EXCLUDED.version;

INSERT INTO exam_banks (id, competition_id, sector, version, definition, hash, actor_id, created_at)
VALUES (
    '01a19fb4-7da0-763b-8155-f215a6ca0398'::uuid,
    (SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'),
    'empleadores',
    1,
    '{"versionAnterior":0,"preguntas":[{"id":"em-01","enunciado":"¿Cuál es la obligación patronal primordial frente al régimen de seguridad social del Ecuador?","opciones":[{"id":"em-01-a","texto":"Retener y depositar las aportaciones personales y patronales dentro de los plazos fijados por ley"},{"id":"em-01-b","texto":"Administrar privadamente los fondos de cesantía de cada empleado"},{"id":"em-01-c","texto":"Determinar autónomamente las tasas actuariales de contribución"},{"id":"em-01-d","texto":"Eximir del pago de aportes a trabajadores a tiempo parcial"}],"correcta":"em-01-a"},{"id":"em-02","enunciado":"En la canalización de inversiones del BIESS hacia el sector productivo, ¿qué instrumentos son autorizados?","opciones":[{"id":"em-02-a","texto":"Anticipos de caja sin respaldo de garantías reales"},{"id":"em-02-b","texto":"Inversiones en títulos valores inscritos en el catastro público del mercado de valores"},{"id":"em-02-c","texto":"Aportes no reembolsables de reactivación privada"},{"id":"em-02-d","texto":"Préstamos directos sin calificación crediticia a accionistas"}],"correcta":"em-02-b"},{"id":"em-03","enunciado":"La mora patronal en el cumplimiento de aportes ante el IESS ocasiona:","opciones":[{"id":"em-03-a","texto":"La extinción irrevocable del derecho a pensiones del afiliado"},{"id":"em-03-b","texto":"Intereses por mora, coactiva y determinación de responsabilidad patronal"},{"id":"em-03-c","texto":"El débito directo a las cuentas personales de los trabajadores afectados"},{"id":"em-03-d","texto":"La prescripción automática de las obligaciones vencidas"}],"correcta":"em-03-b"},{"id":"em-04","enunciado":"El Comité de Inversiones del BIESS tiene como función clave:","opciones":[{"id":"em-04-a","texto":"Evaluar propuestas de inversión, límites de asignación de activos y perfil de riesgo"},{"id":"em-04-b","texto":"Dictaminar en juicios coactivos de cobranza patronal"},{"id":"em-04-c","texto":"Designar al Procurador General del IESS"},{"id":"em-04-d","texto":"Fijar el salario básico unificado a nivel nacional"}],"correcta":"em-04-a"},{"id":"em-05","enunciado":"Los postulantes a representar al sector empleador ante el Directorio del BIESS deben contar con respaldo de:","opciones":[{"id":"em-05-a","texto":"Federaciones o cámaras de la producción de representación nacional legalmente constituidas"},{"id":"em-05-b","texto":"Organismos multilaterales con sede en el exterior"},{"id":"em-05-c","texto":"Colegios profesionales no vinculados a la actividad económica"},{"id":"em-05-d","texto":"Gremios gremiales únicamente cantonales o parroquiales"}],"correcta":"em-05-a"}],"referencia":"Resolución No. C.D. 701 - Banco de Preguntas Sector Empleadores BIESS 2026"}'::jsonb,
    '471e2f092c00628dfb523c98386594030ecffb950579ecb5fe431cf85bb03ee7',
    (SELECT id FROM applicants WHERE cedula = '0910034008'),
    now()
)
ON CONFLICT (competition_id, sector, version) DO NOTHING;

INSERT INTO exam_configs (id, bank_id, competition_id, sector, definition, hash, actor_id, created_at)
VALUES (
    '01a19fb4-7da0-763c-b1ce-b35fab0a2e5c'::uuid,
    '01a19fb4-7da0-763b-8155-f215a6ca0398'::uuid,
    (SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'),
    'empleadores',
    '{"cantidadPreguntas":3,"duracionSegundos":1800,"maximo":100,"umbral":70,"redondeo":"inferior","politicaIntentos":"un_intento","politicaUnicidad":"orden_preguntas_unico_rechazar_agotamiento","referenciaAprobacion":"Aprobación Comisión de Selección - Reglas Sector Empleadores 2026","abreEn":"2026-01-05T05:00:00Z","cierraEn":"2027-01-01T04:59:59Z"}'::jsonb,
    'ccf44f27ebf0cdafdf07b7586d3ecd4c40e54cd8850321169f50e87609d626dc',
    (SELECT id FROM applicants WHERE cedula = '0910034008'),
    now()
)
ON CONFLICT (bank_id) DO NOTHING;

COMMIT;
