-- Esquema que consume sqlc. Se regenera desde las migraciones con `make db-schema`.
CREATE TABLE applicants (
    id                UUID PRIMARY KEY,
    cedula            VARCHAR(10)  NOT NULL UNIQUE,
    nombres           VARCHAR(120) NOT NULL,
    apellidos         VARCHAR(120) NOT NULL,
    correo            VARCHAR(255) NOT NULL UNIQUE,
    metodo_validacion VARCHAR(32)  NOT NULL DEFAULT 'fecha_expedicion',
    fecha_registro    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE submissions (
    id           UUID PRIMARY KEY,
    applicant_id UUID        NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    code         VARCHAR(32) NOT NULL UNIQUE,
    status       VARCHAR(16) NOT NULL DEFAULT 'draft',
    content      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    hash         VARCHAR(64),
    submitted_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT submissions_status_check CHECK (status IN ('draft', 'in_review', 'submitted'))
);

CREATE INDEX submissions_applicant_idx ON submissions (applicant_id, updated_at DESC);


-- Identificadores ordenados en el tiempo (RFC 9562, UUIDv7).
--
-- La función uuidv7() es nativa de PostgreSQL 18: el motor debe estar
-- actualizado antes de aplicar esta migración.
--
-- El índice incluye el identificador porque, al ser los UUIDv7 ordenados
-- por fecha de creación, el desempate de la paginación por cursor deja de
-- ser arbitrario y puede resolverse dentro del índice.

ALTER TABLE applicants ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE submissions ALTER COLUMN id SET DEFAULT uuidv7();

DROP INDEX IF EXISTS submissions_applicant_idx;
CREATE INDEX submissions_applicant_idx ON submissions (applicant_id, updated_at DESC, id DESC);


-- Contexto de identidad: contraseñas con Argon2id, sesiones en PostgreSQL
-- y solicitudes de restablecimiento de contraseña.

ALTER TABLE applicants ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE applicants ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE IF NOT EXISTS sessions (
    id           UUID PRIMARY KEY DEFAULT uuidv7(),
    applicant_id UUID NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    token_hash   CHAR(64) NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    ip           VARCHAR(45),
    user_agent   VARCHAR(512),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sessions_applicant_idx ON sessions (applicant_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS password_resets (
    id           UUID PRIMARY KEY DEFAULT uuidv7(),
    applicant_id UUID NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    token_hash   CHAR(64) NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    used         BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS password_resets_applicant_idx ON password_resets (applicant_id);


CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    section INTEGER NOT NULL CHECK (section BETWEEN 4 AND 8),
    record_id VARCHAR(255) NOT NULL DEFAULT '',
    name VARCHAR(255) NOT NULL,
    mime VARCHAR(64) NOT NULL CHECK (mime = 'application/pdf'),
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0 AND size_bytes <= CASE WHEN section = 4 THEN 4194304 ELSE 2097152 END),
    sha256 VARCHAR(64) NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    storage_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (submission_id, section, record_id),
    CHECK ((section = 4 AND record_id = '') OR (section <> 4 AND record_id <> ''))
);

-- Las bajas, incluidas las cascadas, se confirman junto a los metadatos.
CREATE TABLE document_deletions (
    storage_key TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose StatementBegin
CREATE FUNCTION enqueue_document_deletion() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO document_deletions (storage_key) VALUES (OLD.storage_key) ON CONFLICT DO NOTHING;
    RETURN OLD;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER document_deletion_queue AFTER DELETE ON documents
FOR EACH ROW EXECUTE FUNCTION enqueue_document_deletion();


ALTER TABLE applicants
    ADD COLUMN role VARCHAR(32) NOT NULL DEFAULT 'postulante'
    CONSTRAINT applicants_role_check CHECK (role IN ('postulante', 'analista'));


CREATE TABLE submission_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    applicant_id UUID NOT NULL REFERENCES applicants(id),
    event_type VARCHAR(16) NOT NULL CHECK (event_type IN ('guardada', 'enviada', 'consultada')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX submission_events_submission_idx
    ON submission_events (submission_id, created_at, id);


ALTER TABLE applicants DROP CONSTRAINT applicants_role_check;
ALTER TABLE applicants ADD CONSTRAINT applicants_role_check
    CHECK (role IN ('postulante', 'analista', 'administrador'));

CREATE TABLE administrative_access_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    actor_id UUID NOT NULL,
    actor_role VARCHAR(32) NOT NULL CHECK (actor_role IN ('analista', 'administrador')),
    action VARCHAR(64) NOT NULL,
    resource_id UUID NOT NULL,
    resource_type VARCHAR(32) NOT NULL,
    result VARCHAR(16) NOT NULL CHECK (result IN ('permitido', 'rechazado')),
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX administrative_access_resource_idx
    ON administrative_access_events (resource_type, resource_id, created_at DESC, id DESC);

CREATE TABLE applicant_role_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    applicant_id UUID NOT NULL,
    actor_id UUID,
    operator TEXT NOT NULL,
    previous_role VARCHAR(32) NOT NULL,
    new_role VARCHAR(32) NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX applicant_role_events_applicant_idx
    ON applicant_role_events (applicant_id, created_at DESC, id DESC);

-- +goose StatementBegin
CREATE FUNCTION record_applicant_role_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.role IS DISTINCT FROM NEW.role THEN
        INSERT INTO applicant_role_events
            (applicant_id, actor_id, operator, previous_role, new_role, reason)
        VALUES (NEW.id, NULLIF(current_setting('app.actor_id', true), '')::uuid,
            COALESCE(NULLIF(current_setting('app.operator', true), ''), current_user),
            OLD.role, NEW.role,
            COALESCE(NULLIF(current_setting('app.role_reason', true), ''), 'Actualización SQL por operador de base de datos'));
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER applicants_role_audit AFTER UPDATE OF role ON applicants
    FOR EACH ROW EXECUTE FUNCTION record_applicant_role_change();

-- +goose StatementBegin
CREATE FUNCTION protect_authorization_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'El historial de autorización no admite modificaciones';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER administrative_access_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON administrative_access_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER applicant_role_events_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON applicant_role_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();


-- Generado con go run ./cmd/catalog-seed; fuente: submission.FormDefinition().
-- PostgreSQL 18; 194 países RF, 24 provincias y 221 cantones del catálogo del proyecto.
CREATE TABLE IF NOT EXISTS catalogs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    kind VARCHAR(32) NOT NULL,
    parent_key VARCHAR(320) NOT NULL DEFAULT '',
    parent_id UUID REFERENCES catalogs(id) ON DELETE RESTRICT,
    value VARCHAR(160) NOT NULL CHECK (length(btrim(value)) > 0),
    position INTEGER NOT NULL CHECK (position >= 0),
    CONSTRAINT catalogs_value_unique UNIQUE (kind, parent_key, value),
    CONSTRAINT catalogs_kind_check CHECK (kind IN (
        'countries', 'provinces', 'cities', 'generos', 'estadosCiviles',
        'respuestas', 'parentescos', 'instituciones', 'sectores', 'niveles',
        'tiposInstitucion', 'areas', 'tiposExperiencia', 'tiposPublicacion', 'tiposEvento'
    )),
    CONSTRAINT catalogs_parent_check CHECK (
        (kind NOT IN ('provinces', 'cities') AND parent_key = '' AND parent_id IS NULL)
        OR (kind = 'provinces' AND parent_key = 'EC' AND parent_id IS NOT NULL)
        OR (kind = 'cities' AND parent_key LIKE 'EC/%' AND parent_id IS NOT NULL)
    )
);
CREATE INDEX IF NOT EXISTS catalogs_lookup_idx ON catalogs(kind, parent_key, position, value);

INSERT INTO catalogs (kind, parent_key, parent_id, value, position) VALUES
    ('countries', '', NULL, 'Afganistán', 0),
    ('countries', '', NULL, 'Albania', 1),
    ('countries', '', NULL, 'Alemania', 2),
    ('countries', '', NULL, 'Andorra', 3),
    ('countries', '', NULL, 'Angola', 4),
    ('countries', '', NULL, 'Antigua y Barbuda', 5),
    ('countries', '', NULL, 'Arabia Saudita', 6),
    ('countries', '', NULL, 'Argelia', 7),
    ('countries', '', NULL, 'Argentina', 8),
    ('countries', '', NULL, 'Armenia', 9),
    ('countries', '', NULL, 'Australia', 10),
    ('countries', '', NULL, 'Austria', 11),
    ('countries', '', NULL, 'Azerbaiyán', 12),
    ('countries', '', NULL, 'Bahamas', 13),
    ('countries', '', NULL, 'Bangladés', 14),
    ('countries', '', NULL, 'Barbados', 15),
    ('countries', '', NULL, 'Baréin', 16),
    ('countries', '', NULL, 'Bélgica', 17),
    ('countries', '', NULL, 'Belice', 18),
    ('countries', '', NULL, 'Benín', 19),
    ('countries', '', NULL, 'Bielorrusia', 20),
    ('countries', '', NULL, 'Birmania', 21),
    ('countries', '', NULL, 'Bolivia', 22),
    ('countries', '', NULL, 'Bosnia y Herzegovina', 23),
    ('countries', '', NULL, 'Botsuana', 24),
    ('countries', '', NULL, 'Brasil', 25),
    ('countries', '', NULL, 'Brunéi', 26),
    ('countries', '', NULL, 'Bulgaria', 27),
    ('countries', '', NULL, 'Burkina Faso', 28),
    ('countries', '', NULL, 'Burundi', 29),
    ('countries', '', NULL, 'Bután', 30),
    ('countries', '', NULL, 'Cabo Verde', 31),
    ('countries', '', NULL, 'Camboya', 32),
    ('countries', '', NULL, 'Camerún', 33),
    ('countries', '', NULL, 'Canadá', 34),
    ('countries', '', NULL, 'Catar', 35),
    ('countries', '', NULL, 'Chad', 36),
    ('countries', '', NULL, 'Chile', 37),
    ('countries', '', NULL, 'China', 38),
    ('countries', '', NULL, 'Chipre', 39),
    ('countries', '', NULL, 'Ciudad del Vaticano', 40),
    ('countries', '', NULL, 'Colombia', 41),
    ('countries', '', NULL, 'Comoras', 42),
    ('countries', '', NULL, 'Corea del Norte', 43),
    ('countries', '', NULL, 'Corea del Sur', 44),
    ('countries', '', NULL, 'Costa de Marfil', 45),
    ('countries', '', NULL, 'Costa Rica', 46),
    ('countries', '', NULL, 'Croacia', 47),
    ('countries', '', NULL, 'Cuba', 48),
    ('countries', '', NULL, 'Dinamarca', 49),
    ('countries', '', NULL, 'Dominica', 50),
    ('countries', '', NULL, 'Ecuador', 51),
    ('countries', '', NULL, 'Egipto', 52),
    ('countries', '', NULL, 'El Salvador', 53),
    ('countries', '', NULL, 'Emiratos Árabes Unidos', 54),
    ('countries', '', NULL, 'Eritrea', 55),
    ('countries', '', NULL, 'Eslovaquia', 56),
    ('countries', '', NULL, 'Eslovenia', 57),
    ('countries', '', NULL, 'España', 58),
    ('countries', '', NULL, 'Estados Unidos', 59),
    ('countries', '', NULL, 'Estonia', 60),
    ('countries', '', NULL, 'Etiopía', 61),
    ('countries', '', NULL, 'Filipinas', 62),
    ('countries', '', NULL, 'Finlandia', 63),
    ('countries', '', NULL, 'Fiyi', 64),
    ('countries', '', NULL, 'Francia', 65),
    ('countries', '', NULL, 'Gabón', 66),
    ('countries', '', NULL, 'Gambia', 67),
    ('countries', '', NULL, 'Georgia', 68),
    ('countries', '', NULL, 'Ghana', 69),
    ('countries', '', NULL, 'Granada', 70),
    ('countries', '', NULL, 'Grecia', 71),
    ('countries', '', NULL, 'Guatemala', 72),
    ('countries', '', NULL, 'Guyana', 73),
    ('countries', '', NULL, 'Guinea', 74),
    ('countries', '', NULL, 'Guinea-Bisáu', 75),
    ('countries', '', NULL, 'Guinea Ecuatorial', 76),
    ('countries', '', NULL, 'Haití', 77),
    ('countries', '', NULL, 'Honduras', 78),
    ('countries', '', NULL, 'Hungría', 79),
    ('countries', '', NULL, 'India', 80),
    ('countries', '', NULL, 'Indonesia', 81),
    ('countries', '', NULL, 'Irak', 82),
    ('countries', '', NULL, 'Irán', 83),
    ('countries', '', NULL, 'Irlanda', 84),
    ('countries', '', NULL, 'Islandia', 85),
    ('countries', '', NULL, 'Islas Marshall', 86),
    ('countries', '', NULL, 'Islas Salomón', 87),
    ('countries', '', NULL, 'Israel', 88),
    ('countries', '', NULL, 'Italia', 89),
    ('countries', '', NULL, 'Jamaica', 90),
    ('countries', '', NULL, 'Japón', 91),
    ('countries', '', NULL, 'Jordania', 92),
    ('countries', '', NULL, 'Kazajistán', 93),
    ('countries', '', NULL, 'Kenia', 94),
    ('countries', '', NULL, 'Kirguistán', 95),
    ('countries', '', NULL, 'Kiribati', 96),
    ('countries', '', NULL, 'Kuwait', 97),
    ('countries', '', NULL, 'Laos', 98),
    ('countries', '', NULL, 'Lesoto', 99),
    ('countries', '', NULL, 'Letonia', 100),
    ('countries', '', NULL, 'Líbano', 101),
    ('countries', '', NULL, 'Liberia', 102),
    ('countries', '', NULL, 'Libia', 103),
    ('countries', '', NULL, 'Liechtenstein', 104),
    ('countries', '', NULL, 'Lituania', 105),
    ('countries', '', NULL, 'Luxemburgo', 106),
    ('countries', '', NULL, 'Macedonia del Norte', 107),
    ('countries', '', NULL, 'Madagascar', 108),
    ('countries', '', NULL, 'Malasia', 109),
    ('countries', '', NULL, 'Malaui', 110),
    ('countries', '', NULL, 'Maldivas', 111),
    ('countries', '', NULL, 'Malí', 112),
    ('countries', '', NULL, 'Malta', 113),
    ('countries', '', NULL, 'Marruecos', 114),
    ('countries', '', NULL, 'Mauricio', 115),
    ('countries', '', NULL, 'Mauritania', 116),
    ('countries', '', NULL, 'México', 117),
    ('countries', '', NULL, 'Micronesia', 118),
    ('countries', '', NULL, 'Moldavia', 119),
    ('countries', '', NULL, 'Mónaco', 120),
    ('countries', '', NULL, 'Mongolia', 121),
    ('countries', '', NULL, 'Montenegro', 122),
    ('countries', '', NULL, 'Mozambique', 123),
    ('countries', '', NULL, 'Namibia', 124),
    ('countries', '', NULL, 'Nauru', 125),
    ('countries', '', NULL, 'Nepal', 126),
    ('countries', '', NULL, 'Nicaragua', 127),
    ('countries', '', NULL, 'Níger', 128),
    ('countries', '', NULL, 'Nigeria', 129),
    ('countries', '', NULL, 'Noruega', 130),
    ('countries', '', NULL, 'Nueva Zelanda', 131),
    ('countries', '', NULL, 'Omán', 132),
    ('countries', '', NULL, 'Países Bajos', 133),
    ('countries', '', NULL, 'Pakistán', 134),
    ('countries', '', NULL, 'Palaos', 135),
    ('countries', '', NULL, 'Panamá', 136),
    ('countries', '', NULL, 'Papúa Nueva Guinea', 137),
    ('countries', '', NULL, 'Paraguay', 138),
    ('countries', '', NULL, 'Perú', 139),
    ('countries', '', NULL, 'Polonia', 140),
    ('countries', '', NULL, 'Portugal', 141),
    ('countries', '', NULL, 'Reino Unido', 142),
    ('countries', '', NULL, 'República Centroafricana', 143),
    ('countries', '', NULL, 'República Checa', 144),
    ('countries', '', NULL, 'República del Congo', 145),
    ('countries', '', NULL, 'República Democrática del Congo', 146),
    ('countries', '', NULL, 'República Dominicana', 147),
    ('countries', '', NULL, 'Ruanda', 148),
    ('countries', '', NULL, 'Rumanía', 149),
    ('countries', '', NULL, 'Rusia', 150),
    ('countries', '', NULL, 'Samoa', 151),
    ('countries', '', NULL, 'San Cristóbal y Nieves', 152),
    ('countries', '', NULL, 'San Marino', 153),
    ('countries', '', NULL, 'San Vicente y las Granadinas', 154),
    ('countries', '', NULL, 'Santa Lucía', 155),
    ('countries', '', NULL, 'Santo Tomé y Príncipe', 156),
    ('countries', '', NULL, 'Senegal', 157),
    ('countries', '', NULL, 'Serbia', 158),
    ('countries', '', NULL, 'Seychelles', 159),
    ('countries', '', NULL, 'Sierra Leona', 160),
    ('countries', '', NULL, 'Singapur', 161),
    ('countries', '', NULL, 'Siria', 162),
    ('countries', '', NULL, 'Somalia', 163),
    ('countries', '', NULL, 'Sri Lanka', 164),
    ('countries', '', NULL, 'Suazilandia', 165),
    ('countries', '', NULL, 'Sudáfrica', 166),
    ('countries', '', NULL, 'Sudán', 167),
    ('countries', '', NULL, 'Sudán del Sur', 168),
    ('countries', '', NULL, 'Suecia', 169),
    ('countries', '', NULL, 'Suiza', 170),
    ('countries', '', NULL, 'Surinam', 171),
    ('countries', '', NULL, 'Tailandia', 172),
    ('countries', '', NULL, 'Tanzania', 173),
    ('countries', '', NULL, 'Tayikistán', 174),
    ('countries', '', NULL, 'Timor Oriental', 175),
    ('countries', '', NULL, 'Togo', 176),
    ('countries', '', NULL, 'Tonga', 177),
    ('countries', '', NULL, 'Trinidad y Tobago', 178),
    ('countries', '', NULL, 'Túnez', 179),
    ('countries', '', NULL, 'Turkmenistán', 180),
    ('countries', '', NULL, 'Turquía', 181),
    ('countries', '', NULL, 'Tuvalu', 182),
    ('countries', '', NULL, 'Ucrania', 183),
    ('countries', '', NULL, 'Uganda', 184),
    ('countries', '', NULL, 'Uruguay', 185),
    ('countries', '', NULL, 'Uzbekistán', 186),
    ('countries', '', NULL, 'Vanuatu', 187),
    ('countries', '', NULL, 'Venezuela', 188),
    ('countries', '', NULL, 'Vietnam', 189),
    ('countries', '', NULL, 'Yemen', 190),
    ('countries', '', NULL, 'Yibuti', 191),
    ('countries', '', NULL, 'Zambia', 192),
    ('countries', '', NULL, 'Zimbabue', 193),
    ('areas', '', NULL, 'Finanzas', 0),
    ('areas', '', NULL, 'Mercado de Capitales', 1),
    ('areas', '', NULL, 'Administración de Empresas', 2),
    ('areas', '', NULL, 'Economía', 3),
    ('areas', '', NULL, 'Derecho o materias afines', 4),
    ('estadosCiviles', '', NULL, 'Soltero/a', 0),
    ('estadosCiviles', '', NULL, 'Casado/a', 1),
    ('estadosCiviles', '', NULL, 'Unión de hecho', 2),
    ('estadosCiviles', '', NULL, 'Divorciado/a', 3),
    ('estadosCiviles', '', NULL, 'Viudo/a', 4),
    ('generos', '', NULL, 'Masculino', 0),
    ('generos', '', NULL, 'Femenino', 1),
    ('instituciones', '', NULL, 'IESS', 0),
    ('instituciones', '', NULL, 'BIESS', 1),
    ('niveles', '', NULL, 'Tercer Nivel de Grado', 0),
    ('niveles', '', NULL, 'Cuarto Nivel - Especialidad Académica', 1),
    ('niveles', '', NULL, 'Cuarto Nivel - Maestría Académica', 2),
    ('niveles', '', NULL, 'Cuarto Nivel PhD - Doctorado', 3),
    ('parentescos', '', NULL, 'conviviente', 0),
    ('parentescos', '', NULL, 'cónyuge', 1),
    ('parentescos', '', NULL, 'cuñado/a', 2),
    ('parentescos', '', NULL, 'hermano/a', 3),
    ('parentescos', '', NULL, 'hijo político', 4),
    ('parentescos', '', NULL, 'hijo/a', 5),
    ('parentescos', '', NULL, 'madre', 6),
    ('parentescos', '', NULL, 'nieto/a', 7),
    ('parentescos', '', NULL, 'nuera', 8),
    ('parentescos', '', NULL, 'padre', 9),
    ('parentescos', '', NULL, 'primo hermano/a', 10),
    ('parentescos', '', NULL, 'sobrino/a', 11),
    ('parentescos', '', NULL, 'suegro/a', 12),
    ('parentescos', '', NULL, 'tío/a', 13),
    ('parentescos', '', NULL, 'yerno', 14),
    ('respuestas', '', NULL, 'Sí', 0),
    ('respuestas', '', NULL, 'No', 1),
    ('sectores', '', NULL, 'afiliados', 0),
    ('sectores', '', NULL, 'jubilados', 1),
    ('sectores', '', NULL, 'empleadores', 2),
    ('tiposEvento', '', NULL, 'Curso', 0),
    ('tiposEvento', '', NULL, 'Seminario', 1),
    ('tiposEvento', '', NULL, 'Taller', 2),
    ('tiposExperiencia', '', NULL, 'General', 0),
    ('tiposExperiencia', '', NULL, 'Dirección', 1),
    ('tiposInstitucion', '', NULL, 'Nacional', 0),
    ('tiposInstitucion', '', NULL, 'Extranjera', 1),
    ('tiposPublicacion', '', NULL, 'Publicación', 0),
    ('tiposPublicacion', '', NULL, 'Trabajo de investigación', 1)
ON CONFLICT (kind, parent_key, value) DO NOTHING;

INSERT INTO catalogs (kind, parent_key, parent_id, value, position) VALUES
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Azuay', 0),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Bolívar', 1),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Carchi', 2),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Cañar', 3),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Chimborazo', 4),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Cotopaxi', 5),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'El Oro', 6),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Esmeraldas', 7),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Galápagos', 8),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Guayas', 9),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Imbabura', 10),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Loja', 11),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Los Ríos', 12),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Manabí', 13),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Morona Santiago', 14),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Napo', 15),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Orellana', 16),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Pastaza', 17),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Pichincha', 18),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Santa Elena', 19),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Santo Domingo de los Tsáchilas', 20),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Sucumbíos', 21),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Tungurahua', 22),
    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), 'Zamora Chinchipe', 23)
ON CONFLICT (kind, parent_key, value) DO NOTHING;

INSERT INTO catalogs (kind, parent_key, parent_id, value, position) VALUES
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Cuenca', 0),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Chordeleg', 1),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'El Pan', 2),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Girón', 3),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Guachapala', 4),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Gualaceo', 5),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Nabón', 6),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Oña', 7),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Paute', 8),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Ponce Enríquez', 9),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Pucará', 10),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'San Fernando', 11),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Santa Isabel', 12),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Sevilla de Oro', 13),
    ('cities', 'EC/Azuay', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Azuay'), 'Sigsig', 14),
    ('cities', 'EC/Bolívar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Bolívar'), 'Guaranda', 0),
    ('cities', 'EC/Bolívar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Bolívar'), 'Caluma', 1),
    ('cities', 'EC/Bolívar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Bolívar'), 'Chillanes', 2),
    ('cities', 'EC/Bolívar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Bolívar'), 'Chimbo', 3),
    ('cities', 'EC/Bolívar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Bolívar'), 'Echeandía', 4),
    ('cities', 'EC/Bolívar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Bolívar'), 'Las Naves', 5),
    ('cities', 'EC/Bolívar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Bolívar'), 'San Miguel', 6),
    ('cities', 'EC/Carchi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Carchi'), 'Tulcán', 0),
    ('cities', 'EC/Carchi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Carchi'), 'Bolívar', 1),
    ('cities', 'EC/Carchi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Carchi'), 'Espejo', 2),
    ('cities', 'EC/Carchi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Carchi'), 'Mira', 3),
    ('cities', 'EC/Carchi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Carchi'), 'Montúfar', 4),
    ('cities', 'EC/Carchi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Carchi'), 'San Pedro de Huaca', 5),
    ('cities', 'EC/Cañar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cañar'), 'Azogues', 0),
    ('cities', 'EC/Cañar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cañar'), 'Biblián', 1),
    ('cities', 'EC/Cañar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cañar'), 'Cañar', 2),
    ('cities', 'EC/Cañar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cañar'), 'Déleg', 3),
    ('cities', 'EC/Cañar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cañar'), 'El Tambo', 4),
    ('cities', 'EC/Cañar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cañar'), 'La Troncal', 5),
    ('cities', 'EC/Cañar', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cañar'), 'Suscal', 6),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Riobamba', 0),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Alausi', 1),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Chambo', 2),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Chunchi', 3),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Colta', 4),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Cumandá', 5),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Guamote', 6),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Guano', 7),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Pallatanga', 8),
    ('cities', 'EC/Chimborazo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Chimborazo'), 'Penipe', 9),
    ('cities', 'EC/Cotopaxi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cotopaxi'), 'Latacunga', 0),
    ('cities', 'EC/Cotopaxi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cotopaxi'), 'La Maná', 1),
    ('cities', 'EC/Cotopaxi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cotopaxi'), 'Pangua', 2),
    ('cities', 'EC/Cotopaxi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cotopaxi'), 'Pujilí', 3),
    ('cities', 'EC/Cotopaxi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cotopaxi'), 'Salcedo', 4),
    ('cities', 'EC/Cotopaxi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cotopaxi'), 'Saquisilí', 5),
    ('cities', 'EC/Cotopaxi', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Cotopaxi'), 'Sigchos', 6),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Machala', 0),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Arenillas', 1),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Atahualpa', 2),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Balsas', 3),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Chilla', 4),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'El Guabo', 5),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Huaquillas', 6),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Las Lajas', 7),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Marcabelí', 8),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Pasaje', 9),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Piñas', 10),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Portovelo', 11),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Santa Rosa', 12),
    ('cities', 'EC/El Oro', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'El Oro'), 'Zaruma', 13),
    ('cities', 'EC/Esmeraldas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Esmeraldas'), 'Esmeraldas', 0),
    ('cities', 'EC/Esmeraldas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Esmeraldas'), 'Atacames', 1),
    ('cities', 'EC/Esmeraldas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Esmeraldas'), 'Eloy Alfaro', 2),
    ('cities', 'EC/Esmeraldas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Esmeraldas'), 'Muisne', 3),
    ('cities', 'EC/Esmeraldas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Esmeraldas'), 'Quinindé', 4),
    ('cities', 'EC/Esmeraldas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Esmeraldas'), 'Rioverde', 5),
    ('cities', 'EC/Esmeraldas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Esmeraldas'), 'San Lorenzo', 6),
    ('cities', 'EC/Galápagos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Galápagos'), 'San Cristóbal', 0),
    ('cities', 'EC/Galápagos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Galápagos'), 'Isabela', 1),
    ('cities', 'EC/Galápagos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Galápagos'), 'Santa Cruz', 2),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Guayaquil', 0),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Alfredo Baquerizo Moreno (Juján)', 1),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Balao', 2),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Balzar', 3),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Colimes', 4),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Daule', 5),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Durán', 6),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'El Empalme', 7),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'El Triunfo', 8),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'General Antonio Elizalde (Bucay)', 9),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Isidro Ayora', 10),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Lomas de Sargentillo', 11),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Marcelino Maridueña', 12),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Milagro', 13),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Naranjal', 14),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Naranjito', 15),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Nobol', 16),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Palestina', 17),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Pedro Carbo', 18),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Playas', 19),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Salitre', 20),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Samborondón', 21),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Santa Lucía', 22),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Simón Bolívar', 23),
    ('cities', 'EC/Guayas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Guayas'), 'Yaguachi', 24),
    ('cities', 'EC/Imbabura', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Imbabura'), 'Ibarra', 0),
    ('cities', 'EC/Imbabura', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Imbabura'), 'Antonio Ante', 1),
    ('cities', 'EC/Imbabura', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Imbabura'), 'Cotacachi', 2),
    ('cities', 'EC/Imbabura', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Imbabura'), 'Otavalo', 3),
    ('cities', 'EC/Imbabura', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Imbabura'), 'Pimampiro', 4),
    ('cities', 'EC/Imbabura', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Imbabura'), 'San Miguel de Urcuquí', 5),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Loja', 0),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Calvas', 1),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Catamayo', 2),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Celica', 3),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Chaguarpamba', 4),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Espíndola', 5),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Gonzanamá', 6),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Macará', 7),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Olmedo', 8),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Paltas', 9),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Pindal', 10),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Puyango', 11),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Quilanga', 12),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Saraguro', 13),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Sozoranga', 14),
    ('cities', 'EC/Loja', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Loja'), 'Zapotillo', 15),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Babahoyo', 0),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Baba', 1),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Buena Fe', 2),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Mocache', 3),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Montalvo', 4),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Palenque', 5),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Puebloviejo', 6),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Quevedo', 7),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Quinsaloma', 8),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Urdaneta', 9),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Valencia', 10),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Ventanas', 11),
    ('cities', 'EC/Los Ríos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Los Ríos'), 'Vinces', 12),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Portoviejo', 0),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Bolívar', 1),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Chone', 2),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'El Carmen', 3),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Flavio Alfaro', 4),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Jama', 5),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Jaramijó', 6),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Jipijapa', 7),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Junín', 8),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Manta', 9),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Montecristi', 10),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Olmedo', 11),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Paján', 12),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Pedernales', 13),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Pichincha', 14),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Puerto López', 15),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Rocafuerte', 16),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'San Vicente', 17),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Santa Ana', 18),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Sucre (Bahía)', 19),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Tosagua', 20),
    ('cities', 'EC/Manabí', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Manabí'), 'Veinticuatro de Mayo', 21),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Morona (Macas)', 0),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Gualaquiza', 1),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Huamboya', 2),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Limón Indanza', 3),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Logroño', 4),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Pablo Sexto', 5),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Palora', 6),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'San Juan Bosco', 7),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Santiago', 8),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Sucúa', 9),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Taisha', 10),
    ('cities', 'EC/Morona Santiago', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Morona Santiago'), 'Tiwintza', 11),
    ('cities', 'EC/Napo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Napo'), 'Tena', 0),
    ('cities', 'EC/Napo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Napo'), 'Archidona', 1),
    ('cities', 'EC/Napo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Napo'), 'Carlos Julio Arosemena Tola', 2),
    ('cities', 'EC/Napo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Napo'), 'El Chaco', 3),
    ('cities', 'EC/Napo', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Napo'), 'Quijos (Baeza)', 4),
    ('cities', 'EC/Orellana', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Orellana'), 'Francisco de Orellana (El Coca)', 0),
    ('cities', 'EC/Orellana', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Orellana'), 'Aguarico', 1),
    ('cities', 'EC/Orellana', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Orellana'), 'La Joya de los Sachas', 2),
    ('cities', 'EC/Orellana', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Orellana'), 'Loreto', 3),
    ('cities', 'EC/Pastaza', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pastaza'), 'Pastaza (Puyo)', 0),
    ('cities', 'EC/Pastaza', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pastaza'), 'Arajuno', 1),
    ('cities', 'EC/Pastaza', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pastaza'), 'Mera', 2),
    ('cities', 'EC/Pastaza', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pastaza'), 'Santa Clara', 3),
    ('cities', 'EC/Pichincha', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pichincha'), 'Quito', 0),
    ('cities', 'EC/Pichincha', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pichincha'), 'Cayambe', 1),
    ('cities', 'EC/Pichincha', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pichincha'), 'Mejía (Machachi)', 2),
    ('cities', 'EC/Pichincha', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pichincha'), 'Pedro Moncayo (Tabacundo)', 3),
    ('cities', 'EC/Pichincha', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pichincha'), 'Pedro Vicente Maldonado', 4),
    ('cities', 'EC/Pichincha', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pichincha'), 'Puerto Quito', 5),
    ('cities', 'EC/Pichincha', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pichincha'), 'Rumiñahui (Sangolquí)', 6),
    ('cities', 'EC/Pichincha', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Pichincha'), 'San Miguel de Los Bancos', 7),
    ('cities', 'EC/Santa Elena', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Santa Elena'), 'Santa Elena', 0),
    ('cities', 'EC/Santa Elena', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Santa Elena'), 'La Libertad', 1),
    ('cities', 'EC/Santa Elena', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Santa Elena'), 'Salinas', 2),
    ('cities', 'EC/Santo Domingo de los Tsáchilas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Santo Domingo de los Tsáchilas'), 'Santo Domingo', 0),
    ('cities', 'EC/Santo Domingo de los Tsáchilas', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Santo Domingo de los Tsáchilas'), 'La Concordia', 1),
    ('cities', 'EC/Sucumbíos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Sucumbíos'), 'Lago Agrio (Nueva Loja)', 0),
    ('cities', 'EC/Sucumbíos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Sucumbíos'), 'Cascales', 1),
    ('cities', 'EC/Sucumbíos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Sucumbíos'), 'Cuyabeno', 2),
    ('cities', 'EC/Sucumbíos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Sucumbíos'), 'Gonzalo Pizarro', 3),
    ('cities', 'EC/Sucumbíos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Sucumbíos'), 'Putumayo', 4),
    ('cities', 'EC/Sucumbíos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Sucumbíos'), 'Shushufindi', 5),
    ('cities', 'EC/Sucumbíos', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Sucumbíos'), 'Sucumbíos', 6),
    ('cities', 'EC/Tungurahua', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Tungurahua'), 'Ambato', 0),
    ('cities', 'EC/Tungurahua', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Tungurahua'), 'Baños de Agua Santa', 1),
    ('cities', 'EC/Tungurahua', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Tungurahua'), 'Cevallos', 2),
    ('cities', 'EC/Tungurahua', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Tungurahua'), 'Mocha', 3),
    ('cities', 'EC/Tungurahua', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Tungurahua'), 'Patate', 4),
    ('cities', 'EC/Tungurahua', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Tungurahua'), 'Pelileo', 5),
    ('cities', 'EC/Tungurahua', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Tungurahua'), 'Píllaro', 6),
    ('cities', 'EC/Tungurahua', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Tungurahua'), 'Quero', 7),
    ('cities', 'EC/Tungurahua', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Tungurahua'), 'Tisaleo', 8),
    ('cities', 'EC/Zamora Chinchipe', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Zamora Chinchipe'), 'Zamora', 0),
    ('cities', 'EC/Zamora Chinchipe', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Zamora Chinchipe'), 'Centinela del Cóndor', 1),
    ('cities', 'EC/Zamora Chinchipe', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Zamora Chinchipe'), 'Chinchipe', 2),
    ('cities', 'EC/Zamora Chinchipe', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Zamora Chinchipe'), 'El Pangui', 3),
    ('cities', 'EC/Zamora Chinchipe', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Zamora Chinchipe'), 'Nangaritza', 4),
    ('cities', 'EC/Zamora Chinchipe', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Zamora Chinchipe'), 'Palanda', 5),
    ('cities', 'EC/Zamora Chinchipe', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Zamora Chinchipe'), 'Paquisha', 6),
    ('cities', 'EC/Zamora Chinchipe', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Zamora Chinchipe'), 'Yacuambi', 7),
    ('cities', 'EC/Zamora Chinchipe', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = 'Zamora Chinchipe'), 'Yantzaza', 8)
ON CONFLICT (kind, parent_key, value) DO NOTHING;


-- El historial sobrevive a la baja de borradores/cuentas. No contiene sus datos personales.
ALTER TABLE submission_events DROP CONSTRAINT submission_events_submission_id_fkey;
ALTER TABLE submission_events DROP CONSTRAINT submission_events_applicant_id_fkey;
ALTER TABLE submission_events DROP CONSTRAINT submission_events_event_type_check;
ALTER TABLE submission_events ALTER COLUMN event_type TYPE VARCHAR(64);
-- No adivinar el rol de eventos antiguos a partir del perfil actual.
ALTER TABLE submission_events ADD COLUMN actor_role VARCHAR(32) NOT NULL DEFAULT 'no_registrado';
ALTER TABLE submission_events ADD CONSTRAINT submission_events_event_type_check
    CHECK (event_type IN ('guardada','enviada','consultada','eliminada','documento.cargado','documento.eliminado'));
CREATE TRIGGER submission_events_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON submission_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose StatementBegin
CREATE FUNCTION record_submission_deletion() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO submission_events(submission_id,applicant_id,event_type,actor_role)
        VALUES (OLD.id,OLD.applicant_id,'eliminada',
            COALESCE((SELECT role FROM applicants WHERE id=OLD.applicant_id),'no_registrado'));
    RETURN OLD;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER submissions_deletion_audit BEFORE DELETE ON submissions
    FOR EACH ROW EXECUTE FUNCTION record_submission_deletion();

-- +goose StatementBegin
CREATE FUNCTION record_document_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE sid uuid; owner_id uuid; document_id uuid;
BEGIN
    sid := CASE WHEN TG_OP = 'DELETE' THEN OLD.submission_id ELSE NEW.submission_id END;
    document_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.id ELSE NEW.id END;
    SELECT applicant_id INTO owner_id FROM submissions WHERE id = sid;
    -- La cascada de una baja de borrador ya está cubierta por 'eliminada'.
    IF owner_id IS NOT NULL THEN
        INSERT INTO submission_events(submission_id,applicant_id,event_type,payload,actor_role)
            VALUES (sid,owner_id,CASE WHEN TG_OP = 'DELETE' THEN 'documento.eliminado' ELSE 'documento.cargado' END,
                jsonb_build_object('documentId',document_id),
                COALESCE((SELECT role FROM applicants WHERE id=owner_id),'no_registrado'));
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER documents_change_audit AFTER INSERT OR DELETE ON documents
    FOR EACH ROW EXECUTE FUNCTION record_document_change();

ALTER TABLE administrative_access_events ADD COLUMN submission_id UUID;
-- La migración puede completar enlaces antiguos antes de volver a proteger el historial.
ALTER TABLE administrative_access_events DISABLE TRIGGER administrative_access_immutable;
UPDATE administrative_access_events SET submission_id = resource_id WHERE resource_type = 'postulacion';
UPDATE administrative_access_events a SET submission_id = d.submission_id
    FROM documents d WHERE a.resource_type = 'documento' AND a.resource_id = d.id;
ALTER TABLE administrative_access_events ENABLE TRIGGER administrative_access_immutable;
CREATE INDEX administrative_access_submission_idx
    ON administrative_access_events (submission_id, created_at DESC, id DESC);

-- Puerto para las operaciones de T17/T18/T20-T27: sólo metadatos, sin payload libre.
CREATE TABLE workflow_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    submission_id UUID,
    actor_id UUID,
    actor_role VARCHAR(32) NOT NULL CHECK (actor_role IN ('sistema','postulante','analista','administrador')),
    action VARCHAR(64) NOT NULL CHECK (action IN (
        'fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
        'meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
        'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado')),
    resource_id UUID NOT NULL,
    resource_type VARCHAR(32) NOT NULL CHECK (resource_type IN ('postulacion','convocatoria','evaluacion','reconsideracion')),
    result VARCHAR(16) NOT NULL CHECK (result IN ('permitido','rechazado')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((actor_role = 'sistema') = (actor_id IS NULL))
);
CREATE INDEX workflow_events_submission_idx ON workflow_events (submission_id, created_at DESC, id DESC);
CREATE TRIGGER workflow_events_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON workflow_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();


-- Registro de períodos para T9. T18 gestionará aprobación, fases y publicación.
-- Sin semilla: no inventar una convocatoria para los expedientes anteriores.
CREATE TABLE competitions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    code VARCHAR(64) NOT NULL UNIQUE CHECK (btrim(code) <> ''),
    title VARCHAR(200) NOT NULL CHECK (btrim(title) <> ''),
    opens_at TIMESTAMPTZ NOT NULL,
    closes_at TIMESTAMPTZ NOT NULL CHECK (closes_at > opens_at),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE submissions ADD COLUMN competition_id UUID REFERENCES competitions(id);
CREATE INDEX submissions_review_idx ON submissions (competition_id, created_at DESC, id DESC)
    WHERE status = 'submitted' AND hash IS NOT NULL AND submitted_at IS NOT NULL;
CREATE INDEX submissions_review_sector_idx ON submissions
    (competition_id, (content->'seccion4'->>'sector'), created_at DESC, id DESC)
    WHERE status = 'submitted' AND hash IS NOT NULL AND submitted_at IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION protect_submission_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE period competitions;
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'submitted' THEN
            RAISE EXCEPTION 'El expediente enviado es inmutable';
        END IF;
        RETURN OLD;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        IF OLD.status = 'submitted' AND NEW IS DISTINCT FROM OLD THEN
            RAISE EXCEPTION 'El expediente enviado es inmutable';
        END IF;
        IF NEW.competition_id IS DISTINCT FROM OLD.competition_id THEN
            RAISE EXCEPTION 'La convocatoria de origen es inmutable';
        END IF;
    END IF;
    IF NEW.competition_id IS NOT NULL AND (TG_OP = 'INSERT' OR NEW.status = 'submitted') THEN
        SELECT * INTO period FROM competitions WHERE id = NEW.competition_id FOR SHARE;
        IF NOT FOUND OR statement_timestamp() < period.opens_at OR statement_timestamp() >= period.closes_at THEN
            RAISE EXCEPTION 'La convocatoria no admite postulaciones' USING ERRCODE = '23514', CONSTRAINT = 'competition_open';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER submissions_snapshot BEFORE INSERT OR UPDATE OR DELETE ON submissions
    FOR EACH ROW EXECUTE FUNCTION protect_submission_snapshot();

-- +goose StatementBegin
CREATE FUNCTION protect_sealed_documents() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent_id uuid; previous_id uuid;
BEGIN
    parent_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.submission_id ELSE NEW.submission_id END;
    IF TG_OP = 'UPDATE' THEN previous_id := OLD.submission_id; END IF;
    PERFORM id FROM submissions WHERE id IN (parent_id,previous_id) ORDER BY id FOR SHARE;
    IF EXISTS (SELECT 1 FROM submissions WHERE id IN (parent_id,previous_id) AND status = 'submitted') THEN
        RAISE EXCEPTION 'Los certificados de un expediente enviado son inmutables';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER documents_sealed BEFORE INSERT OR UPDATE OR DELETE ON documents
    FOR EACH ROW EXECUTE FUNCTION protect_sealed_documents();
CREATE TRIGGER documents_no_truncate BEFORE TRUNCATE ON documents
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose StatementBegin
CREATE FUNCTION protect_competition_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    -- Código, nombre y período no cambian después de recibir postulaciones.
    IF EXISTS (SELECT 1 FROM submissions WHERE competition_id = OLD.id) THEN
        RAISE EXCEPTION 'La convocatoria con expedientes conserva su período e identidad';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER competitions_history BEFORE UPDATE OR DELETE ON competitions
    FOR EACH ROW EXECUTE FUNCTION protect_competition_history();
CREATE TRIGGER submissions_no_truncate BEFORE TRUNCATE ON submissions
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER competitions_no_truncate BEFORE TRUNCATE ON competitions
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();


-- Nunca persistir OTP, destinatario ni formulario de un correo en texto claro.
CREATE TABLE mail_outbox (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    dedup_key VARCHAR(64) NOT NULL UNIQUE,
    kind VARCHAR(32) NOT NULL,
    resource_id UUID,
    ciphertext BYTEA NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending'
        CHECK(status IN ('pending','processing','smtp_accepted','failed','expired','superseded')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 5),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    lease_id UUID,
    last_error_code VARCHAR(32) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX mail_outbox_pending_idx ON mail_outbox(next_attempt_at,created_at)
    WHERE status IN ('pending','processing');
CREATE INDEX mail_outbox_resource_idx ON mail_outbox(resource_id,created_at DESC,id DESC);


ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_action_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_action_check CHECK(action IN (
 'convocatoria.creada','convocatoria.configurada','fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
 'meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
 'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado'));
ALTER TABLE submission_events DROP CONSTRAINT submission_events_event_type_check;
ALTER TABLE submission_events ADD CONSTRAINT submission_events_event_type_check CHECK(event_type IN (
 'guardada','enviada','consultada','eliminada','documento.cargado','documento.eliminado','postulacion.cerrada_elegibilidad'));
CREATE TABLE competition_controls (
 competition_id UUID PRIMARY KEY REFERENCES competitions(id),
 version BIGINT NOT NULL DEFAULT 1 CHECK(version>0),
 approved BOOLEAN NOT NULL DEFAULT false,
 published BOOLEAN NOT NULL DEFAULT false,
 form_enabled BOOLEAN NOT NULL DEFAULT false,
 sectors TEXT[] NOT NULL DEFAULT '{}',
 phases JSONB NOT NULL DEFAULT '[]',
 approved_at TIMESTAMPTZ,
 approved_by UUID,
 CHECK(NOT published OR approved),
 CHECK(NOT form_enabled OR (approved AND published))
);
-- No habilitar automáticamente períodos existentes ni reasignar expedientes legados.
INSERT INTO competition_controls(competition_id) SELECT id FROM competitions;
CREATE TABLE competition_versions (
 id UUID PRIMARY KEY DEFAULT uuidv7(), competition_id UUID NOT NULL,
 version BIGINT NOT NULL, configuration JSONB NOT NULL, actor_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(competition_id,version)
);
CREATE TRIGGER competition_versions_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON competition_versions
 FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

ALTER TABLE submissions ADD COLUMN eligibility JSONB;
ALTER TABLE submissions DROP CONSTRAINT submissions_status_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_status_check CHECK(status IN ('draft','in_review','submitted','closed'));
ALTER TABLE submissions ADD CONSTRAINT submission_closure_evidence CHECK(status<>'closed' OR (eligibility IS NOT NULL AND eligibility->>'elegible'='false'));
CREATE TABLE eligibility_checks (
 id UUID PRIMARY KEY, submission_id UUID NOT NULL, applicant_id UUID NOT NULL,
 competition_id UUID NOT NULL, sector TEXT NOT NULL, result BOOLEAN NOT NULL,
 source TEXT NOT NULL, checked_at TIMESTAMPTZ NOT NULL, assessment JSONB NOT NULL
);
CREATE INDEX eligibility_checks_submission_idx ON eligibility_checks(submission_id,checked_at DESC,id DESC);
CREATE TRIGGER eligibility_checks_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON eligibility_checks
 FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose StatementBegin
CREATE FUNCTION assert_competition_writable(period_id UUID, selected_sector TEXT) RETURNS void LANGUAGE plpgsql AS $$
DECLARE p competitions; c competition_controls; instant TIMESTAMPTZ; phase JSONB;
BEGIN
 SELECT * INTO p FROM competitions WHERE id=period_id FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'Seleccione una convocatoria' USING ERRCODE='23514',CONSTRAINT='competition_open'; END IF;
 SELECT * INTO c FROM competition_controls WHERE competition_id=period_id FOR SHARE;
 instant:=clock_timestamp();
 IF NOT FOUND OR NOT(c.approved AND c.published AND c.form_enabled) OR instant<p.opens_at OR instant>=p.closes_at THEN
  RAISE EXCEPTION 'La convocatoria no admite escrituras' USING ERRCODE='23514',CONSTRAINT='competition_open';
 END IF;
 IF coalesce(selected_sector,'')<>'' AND NOT(selected_sector=ANY(c.sectors)) THEN
  RAISE EXCEPTION 'El sector no está habilitado' USING ERRCODE='23514',CONSTRAINT='competition_open';
 END IF;
 SELECT value INTO phase FROM jsonb_array_elements(c.phases) WHERE value->>'codigo'='postulacion';
 IF phase IS NULL OR NOT coalesce((phase->>'publicada')::boolean,false) OR NOT coalesce((phase->>'habilitada')::boolean,false)
  OR instant<(phase->>'apertura')::timestamptz OR instant>=(phase->>'cierre')::timestamptz THEN
  RAISE EXCEPTION 'El EF-12 no está habilitado' USING ERRCODE='23514',CONSTRAINT='competition_open';
 END IF;
END; $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION enforce_submission_window() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE item submissions;
BEGIN
 IF TG_OP='DELETE' THEN item:=OLD; ELSE item:=NEW; END IF;
 IF TG_OP<>'INSERT' AND OLD.status IN ('submitted','closed') THEN
  RAISE EXCEPTION 'El expediente no admite cambios' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 PERFORM assert_competition_writable(item.competition_id,item.content->'seccion4'->>'sector');
 IF EXISTS(SELECT 1 FROM submissions WHERE applicant_id=item.applicant_id AND competition_id=item.competition_id AND status='closed' AND id<>item.id) THEN
  RAISE EXCEPTION 'Existe una postulación cerrada para esta cuenta y convocatoria' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END; $$;
-- +goose StatementEnd
CREATE TRIGGER submissions_window BEFORE INSERT OR UPDATE OR DELETE ON submissions
 FOR EACH ROW EXECUTE FUNCTION enforce_submission_window();

-- +goose StatementBegin
CREATE FUNCTION enforce_document_window() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent submissions; parent_id UUID;
BEGIN
 parent_id:=CASE WHEN TG_OP='DELETE' THEN OLD.submission_id ELSE NEW.submission_id END;
 SELECT * INTO parent FROM submissions WHERE id=parent_id FOR SHARE;
 -- La cascada del padre ya pasó por su propio control de escritura.
 IF NOT FOUND AND TG_OP='DELETE' THEN RETURN OLD; END IF;
 IF NOT FOUND OR parent.status NOT IN ('draft','in_review') THEN
  RAISE EXCEPTION 'El expediente no admite adjuntos' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 IF TG_OP='UPDATE' AND NEW.submission_id<>OLD.submission_id THEN RAISE EXCEPTION 'No se cambia la titularidad de un adjunto'; END IF;
 PERFORM assert_competition_writable(parent.competition_id,parent.content->'seccion4'->>'sector');
 IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END; $$;
-- +goose StatementEnd
CREATE TRIGGER documents_window BEFORE INSERT OR UPDATE OR DELETE ON documents
 FOR EACH ROW EXECUTE FUNCTION enforce_document_window();


CREATE TABLE merit_rule_streams (
 competition_id UUID NOT NULL REFERENCES competitions(id), sector TEXT NOT NULL CHECK(sector IN ('afiliados','jubilados','empleadores')),
 version BIGINT NOT NULL DEFAULT 0, active_rule_id UUID, PRIMARY KEY(competition_id,sector)
);
CREATE TABLE merit_rule_sets (
 id UUID PRIMARY KEY DEFAULT uuidv7(), competition_id UUID NOT NULL REFERENCES competitions(id), sector TEXT NOT NULL,
 version BIGINT NOT NULL CHECK(version>0), definition JSONB NOT NULL, definition_hash VARCHAR(64) NOT NULL,
 approved BOOLEAN NOT NULL DEFAULT false, approval_reference VARCHAR(2000), approved_by UUID, approved_at TIMESTAMPTZ,
 actor_id UUID NOT NULL, created_at TIMESTAMPTZ NOT NULL, UNIQUE(competition_id,sector,version), UNIQUE(id,competition_id,sector),
 CHECK(NOT approved OR (approval_reference IS NOT NULL AND approved_by IS NOT NULL AND approved_at IS NOT NULL))
);
ALTER TABLE merit_rule_streams ADD CONSTRAINT merit_active_rule FOREIGN KEY(active_rule_id,competition_id,sector) REFERENCES merit_rule_sets(id,competition_id,sector);
CREATE TABLE merit_case_streams (
 submission_id UUID PRIMARY KEY REFERENCES submissions(id), review_id UUID, calculation_id UUID, publication_id UUID
);
CREATE TABLE merit_reviews (
 id UUID PRIMARY KEY, submission_id UUID NOT NULL REFERENCES submissions(id), version BIGINT NOT NULL CHECK(version>0),
 rule_set_id UUID NOT NULL REFERENCES merit_rule_sets(id), payload JSONB NOT NULL, UNIQUE(submission_id,version), UNIQUE(id,submission_id)
);
CREATE TABLE merit_calculations (
 id UUID PRIMARY KEY, submission_id UUID NOT NULL REFERENCES submissions(id), version BIGINT NOT NULL CHECK(version>0),
 review_id UUID NOT NULL, rule_set_id UUID NOT NULL REFERENCES merit_rule_sets(id),
 payload JSONB NOT NULL, UNIQUE(submission_id,version), UNIQUE(id,submission_id),
 FOREIGN KEY(review_id,submission_id) REFERENCES merit_reviews(id,submission_id)
);
CREATE TABLE merit_publications (
 id UUID PRIMARY KEY, submission_id UUID NOT NULL REFERENCES submissions(id), version BIGINT NOT NULL CHECK(version>0),
 calculation_id UUID NOT NULL, supersedes_id UUID,
 decision TEXT NOT NULL CHECK(decision IN ('avanza','excluido')), payload JSONB NOT NULL, UNIQUE(submission_id,version), UNIQUE(id,submission_id),
 FOREIGN KEY(calculation_id,submission_id) REFERENCES merit_calculations(id,submission_id),
 FOREIGN KEY(supersedes_id,submission_id) REFERENCES merit_publications(id,submission_id)
);
ALTER TABLE merit_case_streams ADD FOREIGN KEY(review_id,submission_id) REFERENCES merit_reviews(id,submission_id);
ALTER TABLE merit_case_streams ADD FOREIGN KEY(calculation_id,submission_id) REFERENCES merit_calculations(id,submission_id);
ALTER TABLE merit_case_streams ADD FOREIGN KEY(publication_id,submission_id) REFERENCES merit_publications(id,submission_id);

CREATE TRIGGER merit_reviews_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON merit_reviews FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER merit_calculations_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON merit_calculations FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER merit_publications_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON merit_publications FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER merit_rules_no_delete BEFORE DELETE OR TRUNCATE ON merit_rule_sets FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
-- +goose StatementBegin
CREATE FUNCTION protect_merit_rules() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.approved OR NOT NEW.approved OR (NEW.id,NEW.competition_id,NEW.sector,NEW.version,NEW.definition,NEW.definition_hash,NEW.actor_id,NEW.created_at)
 IS DISTINCT FROM (OLD.id,OLD.competition_id,OLD.sector,OLD.version,OLD.definition,OLD.definition_hash,OLD.actor_id,OLD.created_at) THEN
  RAISE EXCEPTION 'Las reglas se versionan; una aprobación no modifica su definición';
 END IF;
 RETURN NEW;
END; $$;
-- +goose StatementEnd
CREATE TRIGGER merit_rules_immutable BEFORE UPDATE ON merit_rule_sets FOR EACH ROW EXECUTE FUNCTION protect_merit_rules();
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_action_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_action_check CHECK(action IN (
 'convocatoria.creada','convocatoria.configurada','fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
 'meritos.reglas_creadas','meritos.reglas_aprobadas','meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
 'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado'));


CREATE TABLE evaluation_policy_streams (
 competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL CHECK(sector IN ('afiliados','jubilados','empleadores')),
 kind TEXT NOT NULL CHECK(kind IN ('oposicion','reconsideracion')),version BIGINT NOT NULL DEFAULT 0,active_id UUID,
 PRIMARY KEY(competition_id,sector,kind)
);
CREATE TABLE evaluation_policies (
 id UUID PRIMARY KEY,competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL,kind TEXT NOT NULL,
 version BIGINT NOT NULL CHECK(version>0),definition JSONB NOT NULL,definition_hash VARCHAR(64) NOT NULL,approved BOOLEAN NOT NULL DEFAULT false,
 approval_reference TEXT,approved_by UUID,approved_at TIMESTAMPTZ,actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(competition_id,sector,kind,version),UNIQUE(id,competition_id,sector,kind),
 CHECK(NOT approved OR (approval_reference IS NOT NULL AND approved_by IS NOT NULL AND approved_at IS NOT NULL))
);
ALTER TABLE evaluation_policy_streams ADD FOREIGN KEY(active_id,competition_id,sector,kind) REFERENCES evaluation_policies(id,competition_id,sector,kind);
CREATE TABLE evaluation_heads (
 submission_id UUID PRIMARY KEY REFERENCES submissions(id),version BIGINT NOT NULL DEFAULT 0,
 state JSONB NOT NULL DEFAULT '{"resultados":{}}'
);
CREATE TABLE evaluation_records (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),action TEXT NOT NULL,
 phase TEXT NOT NULL CHECK(phase IN ('meritos','oposicion_teorica','oposicion_psicotecnica','resultados')),
 payload JSONB NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 CHECK(action IN ('psicotecnica.registrada','psicotecnica.publicada','teorica.finalizada','expediente.consolidado','reconsideracion.presentada','reconsideracion.atendida','reconsideracion.resuelta'))
);
CREATE INDEX evaluation_history_idx ON evaluation_records(submission_id,created_at DESC,id DESC);
CREATE UNIQUE INDEX evaluation_result_version ON evaluation_records(submission_id,phase,((payload->>'version')::bigint)) WHERE action IN ('psicotecnica.publicada','teorica.finalizada','expediente.consolidado');
CREATE UNIQUE INDEX evaluation_theory_attempt ON evaluation_records(submission_id,(payload->>'fuenteId')) WHERE action='teorica.finalizada';
CREATE TABLE evaluation_commands (
 id UUID PRIMARY KEY,actor_id UUID NOT NULL,submission_id UUID NOT NULL REFERENCES submissions(id),operation TEXT NOT NULL,resource_id UUID NOT NULL,
 request_hash VARCHAR(64) NOT NULL,response JSONB NOT NULL
);
CREATE TABLE reconsiderations (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),phase TEXT NOT NULL,result_id UUID NOT NULL,
 policy_id UUID NOT NULL REFERENCES evaluation_policies(id),status TEXT NOT NULL CHECK(status IN ('recibida','atendida','resuelta')),
 version BIGINT NOT NULL CHECK(version>0),payload JSONB NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(submission_id,phase,result_id)
);
CREATE INDEX reconsiderations_page ON reconsiderations(submission_id,created_at DESC,id DESC);
CREATE TABLE evaluation_evidence (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),applicant_id UUID NOT NULL,actor_id UUID NOT NULL,
 applicant_upload BOOLEAN NOT NULL,name TEXT NOT NULL,size_bytes BIGINT NOT NULL CHECK(size_bytes>0 AND size_bytes<=2097152),
 sha256 VARCHAR(64) NOT NULL,storage_key TEXT NOT NULL UNIQUE,created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX evaluation_evidence_quota ON evaluation_evidence(applicant_id) WHERE applicant_upload;
CREATE TABLE evaluation_upload_intents (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),storage_key TEXT NOT NULL UNIQUE,expires_at TIMESTAMPTZ NOT NULL
);
CREATE TRIGGER evaluation_records_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON evaluation_records FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER evaluation_commands_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON evaluation_commands FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER evaluation_evidence_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON evaluation_evidence FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER evaluation_policies_no_delete BEFORE DELETE OR TRUNCATE ON evaluation_policies FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
-- +goose StatementBegin
CREATE FUNCTION protect_evaluation_policy() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF OLD.approved OR NOT NEW.approved OR (NEW.id,NEW.competition_id,NEW.sector,NEW.kind,NEW.version,NEW.definition,NEW.definition_hash,NEW.actor_id,NEW.created_at)
 IS DISTINCT FROM (OLD.id,OLD.competition_id,OLD.sector,OLD.kind,OLD.version,OLD.definition,OLD.definition_hash,OLD.actor_id,OLD.created_at) THEN
 RAISE EXCEPTION 'Las reglas aprobadas son inmutables; cree otra versión'; END IF; RETURN NEW;
END; $$;
-- +goose StatementEnd
CREATE TRIGGER evaluation_policy_immutable BEFORE UPDATE ON evaluation_policies FOR EACH ROW EXECUTE FUNCTION protect_evaluation_policy();
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_action_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_action_check CHECK(action IN (
 'convocatoria.creada','convocatoria.configurada','fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
 'meritos.reglas_creadas','meritos.reglas_aprobadas','meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
 'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado','evaluacion.reglas_creadas','evaluacion.reglas_aprobadas',
 'psicotecnica.registrada','psicotecnica.publicada','expediente.consolidado','reconsideracion.atendida'));

ALTER TABLE mail_outbox ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE mail_outbox ADD COLUMN total_attempts INTEGER NOT NULL DEFAULT 0;
UPDATE mail_outbox SET total_attempts=attempts;
CREATE TABLE stage_notices (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),source_id UUID NOT NULL,
 action TEXT NOT NULL,template_version TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(submission_id,source_id,action)
);
-- Vincula avisos T21/T22 ya encolados sin regenerar sus destinatarios ni cuerpos.
INSERT INTO stage_notices(id,submission_id,source_id,action,template_version,created_at)
SELECT e.id,e.submission_id,e.id,e.action,'resultado-etapa-v1',m.created_at FROM evaluation_records e
JOIN mail_outbox m ON m.resource_id=e.id WHERE m.kind IN ('stage.result','reconsideration.decision')
ON CONFLICT DO NOTHING;
CREATE INDEX stage_notices_page ON stage_notices(submission_id,created_at DESC,id DESC);
CREATE TABLE mail_delivery_events (
 id UUID PRIMARY KEY DEFAULT uuidv7(),mail_id UUID NOT NULL REFERENCES mail_outbox(id),
 status TEXT NOT NULL,error_code TEXT NOT NULL,attempt INTEGER NOT NULL,cycle INTEGER NOT NULL,
 actor_id UUID,reason TEXT,created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX mail_delivery_page ON mail_delivery_events(mail_id,created_at DESC,id DESC);
CREATE TRIGGER stage_notices_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON stage_notices FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER mail_delivery_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON mail_delivery_events FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_resource_type_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_resource_type_check CHECK(resource_type IN ('postulacion','convocatoria','evaluacion','reconsideracion','notificacion'));
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_action_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_action_check CHECK(action IN (
 'convocatoria.creada','convocatoria.configurada','fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
 'meritos.reglas_creadas','meritos.reglas_aprobadas','meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
 'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado','evaluacion.reglas_creadas','evaluacion.reglas_aprobadas',
 'psicotecnica.registrada','psicotecnica.publicada','expediente.consolidado','reconsideracion.atendida',
 'notificacion.reintentada','lista.generada','lista.publicada'));

CREATE TABLE result_list_streams (
 competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL CHECK(sector IN ('afiliados','jubilados','empleadores')),
 version BIGINT NOT NULL DEFAULT 0,published_version BIGINT NOT NULL DEFAULT 0,published_id UUID,
 PRIMARY KEY(competition_id,sector)
);
CREATE TABLE result_lists (
 id UUID PRIMARY KEY,competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL,
 version BIGINT NOT NULL,sources_hash VARCHAR(64) NOT NULL,payload JSONB NOT NULL,actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(competition_id,sector,version),UNIQUE(id,competition_id,sector)
);
CREATE TABLE result_list_publications (
 id UUID PRIMARY KEY,list_id UUID NOT NULL UNIQUE REFERENCES result_lists(id),competition_id UUID NOT NULL,sector TEXT NOT NULL,
 version BIGINT NOT NULL,previous_id UUID REFERENCES result_list_publications(id),request_hash VARCHAR(64) NOT NULL,
 approval_reference TEXT NOT NULL,privacy_reference TEXT NOT NULL,is_public BOOLEAN NOT NULL,public_payload JSONB NOT NULL,
 actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(competition_id,sector,version),UNIQUE(id,competition_id,sector),
 FOREIGN KEY(list_id,competition_id,sector) REFERENCES result_lists(id,competition_id,sector)
);
ALTER TABLE result_list_streams ADD FOREIGN KEY(published_id,competition_id,sector) REFERENCES result_list_publications(id,competition_id,sector);
CREATE TABLE result_list_commands (
 id UUID PRIMARY KEY,actor_id UUID NOT NULL,operation TEXT NOT NULL,scope TEXT NOT NULL,request_hash VARCHAR(64) NOT NULL,resource_id UUID NOT NULL
);
CREATE INDEX result_lists_page ON result_lists(competition_id,sector,created_at DESC,id DESC);
CREATE INDEX result_publications_page ON result_list_publications(competition_id,sector,created_at DESC,id DESC);
CREATE TRIGGER result_lists_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON result_lists FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER result_publications_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON result_list_publications FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER result_commands_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON result_list_commands FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
