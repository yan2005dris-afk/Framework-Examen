-- +goose Up
-- Fase 2 de la normalización del formulario de postulación (EF-12): una tabla
-- por sección y por registro repetible. Las tablas nacen vacías; el adaptador
-- empieza a escribirlas en la fase 3 y `submissions.content` sigue siendo la
-- fuente hasta la fase 4.
--
-- Jerarquía: submissions → submission_sections → datos de la sección → registros.
-- Una fila de datos sólo existe si su sección está guardada: la sección viaja
-- como columna generada constante dentro de la clave foránea.
--
-- Reglas de los CHECK: nunca más estrictos que la validación de Go con el
-- borrador incompleto (`ValidateForm(..., false)`). Un texto pendiente es '' y
-- una fecha pendiente es NULL. Las fechas futuras, el módulo 10 y lo que exige
-- el envío completo siguen en Go.
--
-- No se guarda lo que se deriva de otra fuente (3FN): cédula y correo principal
-- (cuenta en `applicants`), duración de la experiencia (se calcula), metadatos
-- del PDF (`documents`), `verificadoRegistroCivil` (siempre falso, ADR-008) y la
-- versión y los textos del consentimiento (constantes del RF). El snapshot
-- sellado los conserva tal como se enviaron.

-- Claves que permiten referenciar catálogos y documentos con su contexto.
ALTER TABLE catalogs ADD CONSTRAINT catalogs_id_kind_unique UNIQUE (id, kind);
ALTER TABLE catalogs ADD CONSTRAINT catalogs_id_parent_unique UNIQUE (id, parent_id);
ALTER TABLE documents ADD CONSTRAINT documents_reference_unique UNIQUE (id, submission_id, section, record_id);

-- Secciones guardadas. `items_declared` distingue una lista enviada vacía de una
-- no declarada (`null`) en las secciones con registros repetibles.
CREATE TABLE submission_sections (
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    section SMALLINT NOT NULL CHECK (section BETWEEN 1 AND 9),
    items_declared BOOLEAN NOT NULL DEFAULT true,
    saved_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (submission_id, section),
    CONSTRAINT submission_sections_items_check CHECK (items_declared OR section IN (3, 5, 6, 7, 8))
);

-- 1.1 Información personal.
CREATE TABLE submission_personal (
    submission_id UUID PRIMARY KEY,
    section SMALLINT GENERATED ALWAYS AS (1) STORED,
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    gender TEXT NOT NULL DEFAULT '' CHECK (gender IN ('', 'Masculino', 'Femenino')),
    birth_date DATE,
    nationality TEXT NOT NULL DEFAULT '' CHECK (nationality IN ('', 'Ecuatoriana')),
    marital_status TEXT NOT NULL DEFAULT '' CHECK (marital_status IN ('', 'Soltero/a', 'Casado/a', 'Unión de hecho', 'Divorciado/a', 'Viudo/a')),
    spouse_name TEXT NOT NULL DEFAULT '',
    spouse_id_card TEXT NOT NULL DEFAULT '' CHECK (spouse_id_card = '' OR spouse_id_card ~ '^[0-9]{10}$'),
    FOREIGN KEY (submission_id, section) REFERENCES submission_sections(submission_id, section) ON DELETE CASCADE
);

-- 1.2 Información domiciliaria y de contacto. País siempre del catálogo; provincia
-- y cantón del catálogo para Ecuador o escritos a mano para otro país, nunca ambos.
-- Las claves compuestas obligan a que la provincia sea del país y el cantón de la provincia.
CREATE TABLE submission_contact (
    submission_id UUID PRIMARY KEY,
    section SMALLINT GENERATED ALWAYS AS (2) STORED,
    country_id UUID,
    country_kind VARCHAR(32) GENERATED ALWAYS AS ('countries') STORED,
    province_id UUID,
    province_name TEXT NOT NULL DEFAULT '' CHECK (char_length(province_name) <= 120),
    city_id UUID,
    city_name TEXT NOT NULL DEFAULT '' CHECK (char_length(city_name) <= 120),
    mobile TEXT NOT NULL DEFAULT '' CHECK (mobile = '' OR mobile ~ '^\+?[0-9 ()-]{7,20}$'),
    secondary_email TEXT NOT NULL DEFAULT '' CHECK (secondary_email = '' OR secondary_email ~ '^[^\t\n\f\r @]+@[^\t\n\f\r @]+\.[^\t\n\f\r @]+$'),
    main_street TEXT NOT NULL DEFAULT '',
    number TEXT NOT NULL DEFAULT '',
    secondary_street TEXT NOT NULL DEFAULT '',
    area TEXT NOT NULL DEFAULT '',
    reference TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '' CHECK (phone = '' OR phone ~ '^\+?[0-9 ()-]{7,20}$'),
    FOREIGN KEY (submission_id, section) REFERENCES submission_sections(submission_id, section) ON DELETE CASCADE,
    FOREIGN KEY (country_id, country_kind) REFERENCES catalogs(id, kind),
    FOREIGN KEY (province_id, country_id) REFERENCES catalogs(id, parent_id),
    FOREIGN KEY (city_id, province_id) REFERENCES catalogs(id, parent_id),
    CONSTRAINT submission_contact_province_source CHECK (province_id IS NULL OR province_name = ''),
    CONSTRAINT submission_contact_city_source CHECK (city_id IS NULL OR city_name = ''),
    CONSTRAINT submission_contact_city_catalog CHECK (city_id IS NULL OR province_id IS NOT NULL),
    CONSTRAINT submission_contact_province_country CHECK ((province_id IS NULL AND province_name = '') OR country_id IS NOT NULL),
    CONSTRAINT submission_contact_city_province CHECK ((city_id IS NULL AND city_name = '') OR province_id IS NOT NULL OR province_name <> '')
);

-- 1.3 Información adicional y familiares en el IESS/BIESS.
CREATE TABLE submission_additional (
    submission_id UUID PRIMARY KEY,
    section SMALLINT GENERATED ALWAYS AS (3) STORED,
    has_relatives TEXT NOT NULL DEFAULT '' CHECK (has_relatives IN ('', 'Sí', 'No')),
    FOREIGN KEY (submission_id, section) REFERENCES submission_sections(submission_id, section) ON DELETE CASCADE
);

-- Columnas comunes de los registros repetibles: `position` es el orden del arreglo
-- (el hash depende de él) y `record_key` el identificador que genera el cliente,
-- que en un borrador puede faltar y por eso sólo es único cuando existe.
CREATE TABLE submission_relatives (
    submission_id UUID NOT NULL REFERENCES submission_additional(submission_id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    record_key TEXT NOT NULL DEFAULT '' CHECK (char_length(record_key) <= 255),
    name TEXT NOT NULL DEFAULT '',
    relationship TEXT NOT NULL DEFAULT '' CHECK (relationship IN ('', 'conviviente', 'cónyuge', 'cuñado/a', 'hermano/a', 'hijo político', 'hijo/a', 'madre', 'nieto/a', 'nuera', 'padre', 'primo hermano/a', 'sobrino/a', 'suegro/a', 'tío/a', 'yerno')),
    institution TEXT NOT NULL DEFAULT '' CHECK (institution IN ('', 'IESS', 'BIESS')),
    area TEXT NOT NULL DEFAULT '',
    job_title TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (submission_id, position)
);
CREATE UNIQUE INDEX submission_relatives_record_key ON submission_relatives (submission_id, record_key) WHERE record_key <> '';

-- 1.4 Cargo al que postula. El sector vive en `submissions.sector`.
-- Los respaldos referencian el PDF de la misma postulación, sección y registro;
-- la comprobación se difiere al COMMIT porque el PDF se borra antes de limpiar la referencia.
CREATE TABLE submission_position (
    submission_id UUID PRIMARY KEY,
    section SMALLINT GENERATED ALWAYS AS (4) STORED,
    eligible BOOLEAN,
    verification_detail TEXT NOT NULL DEFAULT '',
    has_organization TEXT NOT NULL DEFAULT '' CHECK (has_organization IN ('', 'Sí', 'No')),
    organization TEXT NOT NULL DEFAULT '',
    document_id UUID,
    document_record VARCHAR(255) GENERATED ALWAYS AS ('') STORED,
    FOREIGN KEY (submission_id, section) REFERENCES submission_sections(submission_id, section) ON DELETE CASCADE,
    FOREIGN KEY (document_id, submission_id, section, document_record)
        REFERENCES documents(id, submission_id, section, record_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX submission_position_document ON submission_position (document_id) WHERE document_id IS NOT NULL;

-- 1.5 Formación académica.
CREATE TABLE submission_degrees (
    submission_id UUID NOT NULL,
    section SMALLINT GENERATED ALWAYS AS (5) STORED,
    position INTEGER NOT NULL CHECK (position >= 0),
    record_key TEXT NOT NULL DEFAULT '' CHECK (char_length(record_key) <= 255),
    level TEXT NOT NULL DEFAULT '' CHECK (level IN ('', 'Tercer Nivel de Grado', 'Cuarto Nivel - Especialidad Académica', 'Cuarto Nivel - Maestría Académica', 'Cuarto Nivel PhD - Doctorado')),
    institution TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '' CHECK (type IN ('', 'Nacional', 'Extranjera')),
    title TEXT NOT NULL DEFAULT '',
    area TEXT NOT NULL DEFAULT '' CHECK (area IN ('', 'Finanzas', 'Mercado de Capitales', 'Administración de Empresas', 'Economía', 'Derecho o materias afines')),
    registry TEXT NOT NULL DEFAULT '',
    registry_date DATE,
    document_id UUID,
    PRIMARY KEY (submission_id, position),
    FOREIGN KEY (submission_id, section) REFERENCES submission_sections(submission_id, section) ON DELETE CASCADE,
    FOREIGN KEY (document_id, submission_id, section, record_key)
        REFERENCES documents(id, submission_id, section, record_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE UNIQUE INDEX submission_degrees_record_key ON submission_degrees (submission_id, record_key) WHERE record_key <> '';
CREATE INDEX submission_degrees_document ON submission_degrees (document_id) WHERE document_id IS NOT NULL;

-- 1.6 Experiencia profesional. Las actividades usan las cinco áreas del RF 1.6.6.
CREATE TABLE submission_experiences (
    submission_id UUID NOT NULL,
    section SMALLINT GENERATED ALWAYS AS (6) STORED,
    position INTEGER NOT NULL CHECK (position >= 0),
    record_key TEXT NOT NULL DEFAULT '' CHECK (char_length(record_key) <= 255),
    type TEXT NOT NULL DEFAULT '' CHECK (type IN ('', 'General', 'Dirección')),
    job_title TEXT NOT NULL DEFAULT '',
    institution TEXT NOT NULL DEFAULT '',
    start_date DATE,
    end_date DATE,
    is_current BOOLEAN NOT NULL DEFAULT false,
    activities TEXT NOT NULL DEFAULT '' CHECK (activities IN ('', 'Finanzas', 'Mercado de Capitales', 'Administración de Empresas', 'Economía', 'Derecho o materias afines')),
    functions TEXT NOT NULL DEFAULT '' CHECK (char_length(functions) <= 1000),
    document_id UUID,
    PRIMARY KEY (submission_id, position),
    FOREIGN KEY (submission_id, section) REFERENCES submission_sections(submission_id, section) ON DELETE CASCADE,
    FOREIGN KEY (document_id, submission_id, section, record_key)
        REFERENCES documents(id, submission_id, section, record_id) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT submission_experiences_current CHECK (NOT is_current OR end_date IS NULL),
    CONSTRAINT submission_experiences_range CHECK (start_date IS NULL OR end_date IS NULL OR end_date >= start_date)
);
CREATE UNIQUE INDEX submission_experiences_record_key ON submission_experiences (submission_id, record_key) WHERE record_key <> '';
CREATE INDEX submission_experiences_document ON submission_experiences (document_id) WHERE document_id IS NOT NULL;

-- 1.7 Publicaciones.
CREATE TABLE submission_publications_info (
    submission_id UUID PRIMARY KEY,
    section SMALLINT GENERATED ALWAYS AS (7) STORED,
    has_publications TEXT NOT NULL DEFAULT '' CHECK (has_publications IN ('', 'Sí', 'No')),
    FOREIGN KEY (submission_id, section) REFERENCES submission_sections(submission_id, section) ON DELETE CASCADE
);

CREATE TABLE submission_publications (
    submission_id UUID NOT NULL REFERENCES submission_publications_info(submission_id) ON DELETE CASCADE,
    section SMALLINT GENERATED ALWAYS AS (7) STORED,
    position INTEGER NOT NULL CHECK (position >= 0),
    record_key TEXT NOT NULL DEFAULT '' CHECK (char_length(record_key) <= 255),
    title TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '' CHECK (type IN ('', 'Publicación', 'Trabajo de investigación')),
    medium TEXT NOT NULL DEFAULT '',
    published_on DATE,
    related TEXT NOT NULL DEFAULT '' CHECK (related IN ('', 'Sí', 'No')),
    document_id UUID,
    PRIMARY KEY (submission_id, position),
    FOREIGN KEY (document_id, submission_id, section, record_key)
        REFERENCES documents(id, submission_id, section, record_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE UNIQUE INDEX submission_publications_record_key ON submission_publications (submission_id, record_key) WHERE record_key <> '';
CREATE INDEX submission_publications_document ON submission_publications (document_id) WHERE document_id IS NOT NULL;

-- 1.8 Capacitación. El borrador admite 0 horas; el envío exige más de cero (Go).
CREATE TABLE submission_trainings (
    submission_id UUID NOT NULL,
    section SMALLINT GENERATED ALWAYS AS (8) STORED,
    position INTEGER NOT NULL CHECK (position >= 0),
    record_key TEXT NOT NULL DEFAULT '' CHECK (char_length(record_key) <= 255),
    type TEXT NOT NULL DEFAULT '' CHECK (type IN ('', 'Curso', 'Seminario', 'Taller')),
    name TEXT NOT NULL DEFAULT '',
    institution TEXT NOT NULL DEFAULT '',
    start_date DATE,
    end_date DATE,
    hours BIGINT NOT NULL DEFAULT 0 CHECK (hours >= 0),
    document_id UUID,
    PRIMARY KEY (submission_id, position),
    FOREIGN KEY (submission_id, section) REFERENCES submission_sections(submission_id, section) ON DELETE CASCADE,
    FOREIGN KEY (document_id, submission_id, section, record_key)
        REFERENCES documents(id, submission_id, section, record_id) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT submission_trainings_range CHECK (start_date IS NULL OR end_date IS NULL OR end_date >= start_date)
);
CREATE UNIQUE INDEX submission_trainings_record_key ON submission_trainings (submission_id, record_key) WHERE record_key <> '';
CREATE INDEX submission_trainings_document ON submission_trainings (document_id) WHERE document_id IS NOT NULL;

-- 1.9 Inhabilidades: trece preguntas fijas del RF, una columna por pregunta.
CREATE TABLE submission_disqualifications (
    submission_id UUID PRIMARY KEY,
    section SMALLINT GENERATED ALWAYS AS (9) STORED,
    q1 TEXT NOT NULL DEFAULT '' CHECK (q1 IN ('', 'Sí', 'No')),
    q2 TEXT NOT NULL DEFAULT '' CHECK (q2 IN ('', 'Sí', 'No')),
    q3 TEXT NOT NULL DEFAULT '' CHECK (q3 IN ('', 'Sí', 'No')),
    q4 TEXT NOT NULL DEFAULT '' CHECK (q4 IN ('', 'Sí', 'No')),
    q5 TEXT NOT NULL DEFAULT '' CHECK (q5 IN ('', 'Sí', 'No')),
    q6 TEXT NOT NULL DEFAULT '' CHECK (q6 IN ('', 'Sí', 'No')),
    q7 TEXT NOT NULL DEFAULT '' CHECK (q7 IN ('', 'Sí', 'No')),
    q8 TEXT NOT NULL DEFAULT '' CHECK (q8 IN ('', 'Sí', 'No')),
    q9 TEXT NOT NULL DEFAULT '' CHECK (q9 IN ('', 'Sí', 'No')),
    q10 TEXT NOT NULL DEFAULT '' CHECK (q10 IN ('', 'Sí', 'No')),
    q11 TEXT NOT NULL DEFAULT '' CHECK (q11 IN ('', 'Sí', 'No')),
    q12 TEXT NOT NULL DEFAULT '' CHECK (q12 IN ('', 'Sí', 'No')),
    q13 TEXT NOT NULL DEFAULT '' CHECK (q13 IN ('', 'Sí', 'No')),
    UNIQUE (submission_id, q7),
    FOREIGN KEY (submission_id, section) REFERENCES submission_sections(submission_id, section) ON DELETE CASCADE
);

-- 1.9.7 Detalle de la remoción: sólo existe si la pregunta 7 es «Sí». Diferida
-- para que cambiar la respuesta y quitar el detalle quepan en la misma transacción.
CREATE TABLE submission_removal_details (
    submission_id UUID PRIMARY KEY,
    q7 TEXT GENERATED ALWAYS AS ('Sí') STORED,
    institution TEXT NOT NULL DEFAULT '',
    job_title TEXT NOT NULL DEFAULT '',
    end_date DATE,
    reason TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (submission_id, q7) REFERENCES submission_disqualifications(submission_id, q7)
        ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED
);

-- 1.10 Consentimiento y firma. La aceptación sólo se fecha al sellar.
CREATE TABLE submission_consents (
    submission_id UUID PRIMARY KEY REFERENCES submissions(id) ON DELETE CASCADE,
    privacy_policy_id TEXT NOT NULL DEFAULT '',
    privacy_policy_hash TEXT NOT NULL DEFAULT '',
    accepted_at TIMESTAMPTZ,
    data_processing BOOLEAN NOT NULL DEFAULT false,
    truthfulness BOOLEAN NOT NULL DEFAULT false,
    competition_rules BOOLEAN NOT NULL DEFAULT false,
    name_publication BOOLEAN NOT NULL DEFAULT false,
    physical_delivery BOOLEAN NOT NULL DEFAULT false,
    signature TEXT NOT NULL DEFAULT '',
    signed_at TEXT NOT NULL DEFAULT ''
);

-- +goose StatementBegin
-- Mismo control que los adjuntos: sólo borradores, dentro de la ventana de la
-- convocatoria y sin cambiar de titular. Un expediente enviado o cerrado queda inmutable.
CREATE FUNCTION enforce_submission_child() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent submissions; parent_id UUID;
BEGIN
 parent_id:=CASE WHEN TG_OP='DELETE' THEN OLD.submission_id ELSE NEW.submission_id END;
 SELECT * INTO parent FROM submissions WHERE id=parent_id FOR SHARE;
 -- La cascada del padre ya pasó por su propio control de escritura.
 IF NOT FOUND AND TG_OP='DELETE' THEN RETURN OLD; END IF;
 IF NOT FOUND OR parent.status NOT IN ('draft','in_review') THEN
  RAISE EXCEPTION 'El expediente no admite cambios' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 IF TG_OP='UPDATE' AND NEW.submission_id<>OLD.submission_id THEN RAISE EXCEPTION 'No se cambia la titularidad de un registro'; END IF;
 PERFORM assert_competition_writable(parent.competition_id,parent.sector);
 IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END; $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ DECLARE tab TEXT; BEGIN
 FOREACH tab IN ARRAY ARRAY['submission_sections','submission_personal','submission_contact','submission_additional','submission_relatives',
  'submission_position','submission_degrees','submission_experiences','submission_publications_info','submission_publications',
  'submission_trainings','submission_disqualifications','submission_removal_details','submission_consents'] LOOP
  EXECUTE format('CREATE TRIGGER %I BEFORE INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION enforce_submission_child()', tab||'_window', tab);
  EXECUTE format('CREATE TRIGGER %I BEFORE TRUNCATE ON %I FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history()', tab||'_no_truncate', tab);
 END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE submission_consents;
DROP TABLE submission_removal_details;
DROP TABLE submission_disqualifications;
DROP TABLE submission_trainings;
DROP TABLE submission_publications;
DROP TABLE submission_publications_info;
DROP TABLE submission_experiences;
DROP TABLE submission_degrees;
DROP TABLE submission_position;
DROP TABLE submission_relatives;
DROP TABLE submission_additional;
DROP TABLE submission_contact;
DROP TABLE submission_personal;
DROP TABLE submission_sections;
DROP FUNCTION enforce_submission_child();
ALTER TABLE documents DROP CONSTRAINT documents_reference_unique;
ALTER TABLE catalogs DROP CONSTRAINT catalogs_id_parent_unique;
ALTER TABLE catalogs DROP CONSTRAINT catalogs_id_kind_unique;
