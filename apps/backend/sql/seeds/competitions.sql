-- Convocatoria BIESS 2026 para desarrollo local (UTF-8 garantizado)
BEGIN;

INSERT INTO competitions (
    id,
    code,
    title,
    opens_at,
    closes_at
) VALUES (
    COALESCE((SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'), '01a19577-c264-7b00-a673-ba2c8cfaecac'::uuid),
    'convocatoria-2026-01',
    'Concurso Público de Méritos y Oposición BIESS 2026',
    '2026-01-05 05:00:00+00',
    '2027-01-01 04:59:59.999999+00'
)
ON CONFLICT (code) DO UPDATE SET
    title = EXCLUDED.title,
    opens_at = EXCLUDED.opens_at,
    closes_at = EXCLUDED.closes_at;

INSERT INTO competition_controls (
    competition_id,
    version,
    approved,
    published,
    form_enabled,
    sectors,
    phases,
    approved_at,
    approved_by
) VALUES (
    (SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'),
    2,
    true,
    true,
    true,
    ARRAY['afiliados', 'jubilados', 'empleadores'],
    '[
      {"codigo": "postulacion", "titulo": "Postulación y Recepción de Documentos", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
      {"codigo": "meritos", "titulo": "Revisión y Calificación de Méritos", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
      {"codigo": "oposicion_psicotecnica", "titulo": "Pruebas Psicotécnicas", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
      {"codigo": "oposicion_teorica", "titulo": "Pruebas Teóricas y de Conocimiento", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
      {"codigo": "reconsideracion", "titulo": "Impugnaciones y Reconsideraciones", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true},
      {"codigo": "resultados", "titulo": "Publicación de Resultados Finales", "apertura": "2026-01-05T05:00:00Z", "cierre": "2027-01-01T04:59:59.999999Z", "publicada": true, "habilitada": true}
    ]'::jsonb,
    now(),
    (SELECT id FROM applicants WHERE cedula = '0910034008')
)
ON CONFLICT (competition_id) DO UPDATE SET
    phases = EXCLUDED.phases,
    approved = EXCLUDED.approved,
    published = EXCLUDED.published,
    form_enabled = EXCLUDED.form_enabled;

INSERT INTO privacy_policies (
    id,
    competition_id,
    version,
    definition,
    hash,
    actor_id,
    created_at
) VALUES (
    '01a1957f-eda2-7b66-b7f6-f0892fa1ef3c'::uuid,
    (SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'),
    2,
    '{
      "texto": "Los datos de este expediente se usan exclusivamente para el concurso de méritos y oposición. La publicación de nombres requiere consentimiento explícito. Los documentos, resultados y notificaciones se conservan durante los períodos definidos en esta política.",
      "diasAcceso": 365,
      "diasCorreo": 180,
      "diasReportes": 365,
      "permiteCorreo": true,
      "diasDocumentos": 365,
      "permiteExportacion": true,
      "referenciaAprobacion": "politica-biess-2026-v1",
      "disposicionHistoricos": "revision_humana_sin_borrado_automatico",
      "diasPublicacionNombres": 365,
      "informacionTratamiento": "Concurso BIESS 2026; auditoría al menos 730 días; sesiones expiradas 30 días.",
      "permiteNombresPublicos": true,
      "diasRevisionConservacion": 365
    }'::jsonb,
    'dc8e190c515bbb5bbd8a1593068c17a928bbba17cdc5d74ce17380c2ea70a076',
    (SELECT id FROM applicants WHERE cedula = '0910034008'),
    now()
)
ON CONFLICT (competition_id, version) DO NOTHING;

INSERT INTO privacy_controls (
    competition_id,
    version,
    active_id,
    blocked
) VALUES (
    (SELECT id FROM competitions WHERE code = 'convocatoria-2026-01'),
    2,
    '01a1957f-eda2-7b66-b7f6-f0892fa1ef3c'::uuid,
    false
)
ON CONFLICT (competition_id) DO UPDATE SET
    version = EXCLUDED.version,
    active_id = EXCLUDED.active_id,
    blocked = EXCLUDED.blocked;

COMMIT;

