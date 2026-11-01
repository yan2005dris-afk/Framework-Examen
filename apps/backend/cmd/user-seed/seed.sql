INSERT INTO applicants (
    id,
    cedula,
    nombres,
    apellidos,
    correo,
    metodo_validacion,
    fecha_registro,
    password_hash,
    is_verified,
    role
) VALUES
(
    COALESCE((SELECT id FROM applicants WHERE cedula = '1710034008'), uuidv7()),
    '1710034008',
    'Damian',
    'Postulante',
    'postulante@biess.local',
    'otp_correo',
    now(),
    '$argon2id$v=19$m=65536,t=3,p=2$LxGNHYoKLpKmf2zsF/JI+w$ARD/RIkWT2/PW3m4H3M21oHoZNT4pTBvpZ8lhQsbVPk',
    true,
    'postulante'
),
(
    COALESCE((SELECT id FROM applicants WHERE cedula = '1710034016'), uuidv7()),
    '1710034016',
    'Elena',
    'Analista',
    'analista@biess.local',
    'otp_correo',
    now(),
    '$argon2id$v=19$m=65536,t=3,p=2$LxGNHYoKLpKmf2zsF/JI+w$ARD/RIkWT2/PW3m4H3M21oHoZNT4pTBvpZ8lhQsbVPk',
    true,
    'analista'
),
(
    COALESCE((SELECT id FROM applicants WHERE cedula = '0910034008'), uuidv7()),
    '0910034008',
    'Carlos',
    'Administrador',
    'admin@biess.local',
    'otp_correo',
    now(),
    '$argon2id$v=19$m=65536,t=3,p=2$LxGNHYoKLpKmf2zsF/JI+w$ARD/RIkWT2/PW3m4H3M21oHoZNT4pTBvpZ8lhQsbVPk',
    true,
    'administrador'
)
ON CONFLICT (cedula) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    correo = EXCLUDED.correo,
    role = EXCLUDED.role,
    is_verified = true,
    updated_at = now();

