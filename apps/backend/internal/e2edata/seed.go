// Package e2edata siembra un conjunto determinista de datos para las pruebas
// de extremo a extremo.
//
// El mismo seed sirve a las dos suites: la de interfaz (Playwright, contra el
// stack de desarrollo) y la de API (Go, contra un servidor de prueba). Por eso
// las credenciales, la convocatoria y los expedientes son fijos y el lector
// del DOM y el cliente HTTP ven exactamente el mismo escenario.
//
// La siembra es idempotente: repetarla no duplica cuentas ni expedientes, y
// deja el mismo estado observable. Todos los registros quedan marcados como
// sintéticos para que nadie los confunda con datos reales.
package e2edata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/password"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

/**
 * Mark es la marca visible que acompaña a todo dato sintético.
 */
const Mark = "DATOS SINTÉTICOS - NO VÁLIDO"

/**
 * Password es la contraseña compartida por las cuentas de prueba.
 */
const Password = "TestPassword.2026*"

/**
 * CompetitionCode es el código de la convocatoria sembrada.
 */
const CompetitionCode = "e2e-2026-01"

/**
 * CompetitionID es el identificador estable de la convocatoria sembrada. Es
 * fijo para que los specs puedan citarlo sin leerlo de la base.
 */
var CompetitionID = uuid.MustParse("01a1954b-bf4e-75c0-a235-24242ce05127")

/**
 * Los roles que el servidor reconoce.
 */
const (
	RolPostulante    = "postulante"
	RolAnalista      = "analista"
	RolAdministrador = "administrador"
)

/**
 * Los sectores que la simulación de elegibilidad reconoce.
 */
const (
	SectorAfiliados   = "afiliados"
	SectorJubilados   = "jubilados"
	SectorEmpleadores = "empleadores"
)

/**
 * Account describe una cuenta sembrada con su rol asignado.
 */
type Account struct {
	// Cedula es la clave de acceso de la cuenta.
	Cedula string
	// Correo es el correo declarado de la cuenta.
	Correo string
	// Rol es el rol que el servidor reconoce para la cuenta.
	Rol string
	// ID es el identificador interno de la cuenta.
	ID uuid.UUID
}

/**
 * Submission describe un expediente sembrado y su estado observable.
 */
type Submission struct {
	// Cedula es la cuenta titular del expediente.
	Cedula string
	// ID es el identificador del expediente.
	ID uuid.UUID
	// Estado es el estado del expediente: draft, submitted o closed.
	Estado string
	// Hash es el sello criptográfico; vacío si el expediente no fue enviado.
	Hash string
}

/**
 * Dataset es el resultado de la siembra: el contrato que comparten las dos
 * suites de pruebas.
 */
type Dataset struct {
	// Competition es la convocatoria sembrada.
	Competition uuid.UUID
	// Accounts son las cuentas por cédula.
	Accounts map[string]Account
	// Submissions son los expedientes por escenario.
	Submissions map[string]Submission
	// PolicyID es la política de privacidad vigente de la convocatoria.
	PolicyID uuid.UUID
	// PolicyHash es el hash de la política de privacidad.
	PolicyHash string
}

/**
 * AccountByRole devuelve la cuenta sembrada del rol indicado.
 *
 * @param {Dataset} dataset - Conjunto sembrado.
 * @param {string} rol - Rol buscado.
 * @returns {Account} Cuenta del rol o la cuenta vacía si no existe.
 */
func (d Dataset) AccountByRole(rol string) Account {
	for _, cuenta := range d.Accounts {
		if cuenta.Rol == rol {
			return cuenta
		}
	}
	return Account{}
}

/**
 * SubmissionByScenario devuelve el expediente sembrado del escenario indicado.
 *
 * @param {Dataset} dataset - Conjunto sembrado.
 * @param {string} escenario - Nombre del escenario.
 * @returns {Submission} Expediente del escenario o el vacío si no existe.
 */
func (d Dataset) SubmissionByScenario(escenario string) Submission {
	return d.Submissions[escenario]
}

/**
 * cuentaSembrada describe una cuenta que la siembra debe dejar lista.
 */
type cuentaSembrada struct {
	rol     string
	cedula  string
	correo  string
	nombres string
}

/**
 * cuentasSembradas son las cuentas que el seed deja disponibles. Las cédulas
 * coinciden con las que usan los specs de interfaz, de modo que ambas suites
 * hablan del mismo postulante.
 */
var cuentasSembradas = []cuentaSembrada{
	{RolAdministrador, "2407612650", "e2e-administrador@biess.local", "SINTETICO Administrador"},
	{RolAnalista, "2402249987", "e2e-analista@biess.local", "SINTETICO Analista"},
	{RolPostulante, "2406425922", "e2e-aprobado@biess.local", "SINTETICO Aprobado"},
	{RolPostulante, "2402740035", "e2e-minimo@biess.local", "SINTETICO Minimo"},
	{RolPostulante, "2403401355", "e2e-cerrado@biess.local", "SINTETICO Cerrado"},
	{RolPostulante, "2408380695", "e2e-reconsideracion@biess.local", "SINTETICO Reconsideracion"},
}

/**
 * escenarioSembrado describe un expediente que la siembra debe dejar listo.
 */
type escenarioSembrado struct {
	nombre  string
	cedula  string
	estado  string
	sector  string
	enviado bool
}

/**
 * escenariosSembrados son los expedientes que el seed deja disponibles.
 */
var escenariosSembrados = []escenarioSembrado{
	{"aprobado", "2406425922", "submitted", SectorAfiliados, true},
	{"minimo", "2402740035", "draft", SectorAfiliados, false},
	{"cerrado", "2403401355", "closed", SectorAfiliados, false},
	{"reconsideracion", "2408380695", "submitted", SectorAfiliados, true},
}

/**
 * Seed aplica el conjunto de datos de extremo a extremo sobre el pool dado.
 *
 * La operación es idempotente: si las cuentas y la convocatoria ya existen,
 * se conservan y sólo se actualizan los campos que las pruebas necesitan
 * (contraseña y rol). Los expedientes se crean una vez y después se ajustan
 * al estado que el escenario declara.
 *
 * @param {context.Context} ctx - Contexto de la siembra.
 * @param {*pgxpool.Pool} pool - Pool conectado al esquema de pruebas.
 * @returns {Dataset} El conjunto sembrado, con los identificadores reales.
 * @throws {error} Si la siembra no se puede aplicar de forma consistente.
 */
func Seed(ctx context.Context, pool *pgxpool.Pool) (Dataset, error) {
	dataset := Dataset{
		Competition: CompetitionID,
		Accounts:    map[string]Account{},
		Submissions: map[string]Submission{},
	}

	hash, err := password.Hash(Password)
	if err != nil {
		return dataset, fmt.Errorf("e2edata: no se pudo derivar la contraseña: %w", err)
	}

	// ── Cuentas por rol ──────────────────────────────────────────────────────
	for _, cuenta := range cuentasSembradas {
		id, err := upsertApplicant(ctx, pool, cuenta.cedula, cuenta.correo, cuenta.nombres, cuenta.rol, hash)
		if err != nil {
			return dataset, err
		}
		dataset.Accounts[cuenta.cedula] = Account{
			Cedula: cuenta.cedula,
			Correo: cuenta.correo,
			Rol:    cuenta.rol,
			ID:     id,
		}
	}

	// ── Convocatoria y sus controles ─────────────────────────────────────────
	if err := upsertCompetition(ctx, pool); err != nil {
		return dataset, err
	}

	// ── Política de privacidad vigente ───────────────────────────────────────
	actorID := dataset.AccountByRole(RolAdministrador).ID
	policyID, policyHash, err := upsertPrivacyPolicy(ctx, pool, actorID)
	if err != nil {
		return dataset, err
	}
	dataset.PolicyID = policyID
	dataset.PolicyHash = policyHash

	// ── Expedientes por escenario ────────────────────────────────────────────
	for _, escenario := range escenariosSembrados {
		id, sello, err := upsertSubmission(ctx, pool, dataset, escenario.cedula, escenario.estado, escenario.sector, escenario.enviado)
		if err != nil {
			return dataset, err
		}
		dataset.Submissions[escenario.nombre] = Submission{
			Cedula: escenario.cedula,
			ID:     id,
			Estado: escenario.estado,
			Hash:   sello,
		}
	}

	return dataset, nil
}

/**
 * upsertApplicant crea o actualiza una cuenta y le asigna el rol indicado.
 *
 * @param {context.Context} ctx - Contexto de la siembra.
 * @param {*pgxpool.Pool} pool - Pool de la base de pruebas.
 * @param {string} cedula - Cédula de la cuenta.
 * @param {string} correo - Correo de la cuenta.
 * @param {string} nombres - Nombres declarados.
 * @param {string} rol - Rol que el servidor debe reconocer.
 * @param {string} hash - Hash Argon2id de la contraseña compartida.
 * @returns {uuid.UUID} Identificador interno de la cuenta.
 * @throws {error} Si la cuenta no se puede guardar.
 */
func upsertApplicant(ctx context.Context, pool *pgxpool.Pool, cedula string, correo string, nombres string, rol string, hash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO applicants(cedula,nombres,apellidos,correo,metodo_validacion,password_hash,is_verified,role)
		VALUES($1,$3,$4,$2,'correo',$5,true,$6)
		ON CONFLICT (cedula) DO UPDATE SET
			correo=EXCLUDED.correo,
			password_hash=EXCLUDED.password_hash,
			role=EXCLUDED.role,
			nombres=EXCLUDED.nombres,
			apellidos=EXCLUDED.apellidos,
			is_verified=true
		RETURNING id`, cedula, correo, nombres, Mark, hash, rol).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("e2edata: no se pudo sembrar la cuenta %s: %w", cedula, err)
	}
	return id, nil
}

/**
 * upsertCompetition crea la convocatoria con su control aprobado y publicado.
 *
 * @param {context.Context} ctx - Contexto de la siembra.
 * @param {*pgxpool.Pool} pool - Pool de la base de pruebas.
 * @returns {error} Si la convocatoria no se puede guardar.
 */
func upsertCompetition(ctx context.Context, pool *pgxpool.Pool) error {
	inicio := time.Now().UTC().Add(-24 * time.Hour)
	cierre := time.Now().UTC().Add(365 * 24 * time.Hour)
	// Una convocatoria con expedientes conserva su período y su identidad:
	// el seed nunca reescribe las ventanas de una convocatoria ya sembrada.
	if _, err := pool.Exec(ctx, `
		INSERT INTO competitions(id,code,title,opens_at,closes_at)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (id) DO NOTHING`,
		CompetitionID, CompetitionCode, "Convocatoria de pruebas "+Mark, inicio, cierre); err != nil {
		return fmt.Errorf("e2edata: no se pudo sembrar la convocatoria: %w", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO competition_controls(competition_id,version,approved,published,form_enabled,sectors,phases,approved_at)
		VALUES($1,1,true,true,true,ARRAY[$2,$3,$4],jsonb_build_array(
			jsonb_build_object('codigo','postulacion','titulo','Postulación','apertura',$5::timestamptz,'cierre',$6::timestamptz,'publicada',true,'habilitada',true),
			jsonb_build_object('codigo','meritos','titulo','Méritos','apertura',$5::timestamptz,'cierre',$6::timestamptz,'publicada',true,'habilitada',true),
			jsonb_build_object('codigo','reconsideracion','titulo','Reconsideración','apertura',$5::timestamptz,'cierre',$6::timestamptz,'publicada',true,'habilitada',true)
		),now())
		ON CONFLICT (competition_id) DO UPDATE SET
			approved=true, published=true, form_enabled=true,
			sectors=EXCLUDED.sectors, phases=EXCLUDED.phases`,
		CompetitionID, SectorAfiliados, SectorJubilados, SectorEmpleadores, inicio, cierre); err != nil {
		return fmt.Errorf("e2edata: no se pudo sembrar el control de la convocatoria: %w", err)
	}
	return nil
}

/**
 * upsertPrivacyPolicy publica la política de privacidad de la convocatoria.
 *
 * @param {context.Context} ctx - Contexto de la siembra.
 * @param {*pgxpool.Pool} pool - Pool de la base de pruebas.
 * @param {uuid.UUID} actorID - Cuenta que publica la política.
 * @returns {uuid.UUID} Identificador de la política.
 * @returns {string} Hash de la política.
 * @throws {error} Si la política no se puede guardar.
 */
func upsertPrivacyPolicy(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID) (uuid.UUID, string, error) {
	// La definición se arma con la struct del dominio: si el seed usara sus
	// propias claves, el hash no coincidiría con el que el servidor calcula
	// al verificar la aceptación.
	rules := privacy.Rules{
		Text:         "Los datos de este expediente son sintéticos y existen sólo para probar el sistema. " + Mark,
		Treatment:    Mark + " Origen e2e; eliminación simulada; auditoría conservada.",
		Reference:    "politica-e2e-2026-v1 - " + Mark,
		AccessDays:   365,
		DocumentDays: 365,
		ReportDays:   365,
		PublicDays:   365,
		EmailDays:    180,
		ReviewDays:   365,
		Historical:   "revision_humana_sin_borrado_automatico",
		AllowExports: true,
		AllowEmail:   true,
		AllowNames:   true,
	}
	raw, err := json.Marshal(rules)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo serializar la política: %w", err)
	}
	hash, err := evaluation.Digest(rules)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo resumir la política: %w", err)
	}

	// La política es inmutable por diseño: una versión publicada nunca se
	// reescribe. Si el contenido sembrado cambia, se publica una versión
	// nueva y se activa; la anterior queda como historial.
	var id uuid.UUID
	var version int64
	err = pool.QueryRow(ctx, `
		SELECT id, version FROM privacy_policies WHERE competition_id=$1 ORDER BY version DESC LIMIT 1`,
		CompetitionID).Scan(&id, &version)
	switch {
	case err == nil:
		var hashGuardado string
		if err := pool.QueryRow(ctx, `SELECT hash FROM privacy_policies WHERE id=$1`, id).Scan(&hashGuardado); err != nil {
			return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo leer el hash de la política: %w", err)
		}
		if hashGuardado != hash {
			version++
			policyID, err := uuid.NewV7()
			if err != nil {
				return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo generar el identificador de la política: %w", err)
			}
			if err := pool.QueryRow(ctx, `
				INSERT INTO privacy_policies(id,competition_id,version,definition,hash,actor_id,created_at)
				VALUES($1,$2,$3,$4,$5,$6,now())
				RETURNING id`, policyID, CompetitionID, version, raw, hash, actorID).Scan(&id); err != nil {
				return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo publicar la versión nueva de la política: %w", err)
			}
		}
	case errors.Is(err, pgx.ErrNoRows):
		policyID, err := uuid.NewV7()
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo generar el identificador de la política: %w", err)
		}
		version = 1
		if err := pool.QueryRow(ctx, `
			INSERT INTO privacy_policies(id,competition_id,version,definition,hash,actor_id,created_at)
			VALUES($1,$2,1,$3,$4,$5,now())
			RETURNING id`, policyID, CompetitionID, raw, hash, actorID).Scan(&id); err != nil {
			return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo sembrar la política de privacidad: %w", err)
		}
	default:
		return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo consultar la política de privacidad: %w", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO privacy_controls(competition_id,active_id,version)
		VALUES($1,$2,1)
		ON CONFLICT (competition_id) DO UPDATE SET active_id=EXCLUDED.active_id, version=EXCLUDED.version`,
		CompetitionID, id); err != nil {
		return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo activar la política de privacidad: %w", err)
	}
	return id, hash, nil
}

/**
 * upsertSubmission crea o ajusta el expediente del escenario declarado.
 *
 * @param {context.Context} ctx - Contexto de la siembra.
 * @param {*pgxpool.Pool} pool - Pool de la base de pruebas.
 * @param {Dataset} dataset - Conjunto sembrado hasta ahora.
 * @param {string} cedula - Titular del expediente.
 * @param {string} estado - Estado final del expediente.
 * @param {string} sector - Sector declarado del postulante.
 * @param {bool} enviado - Si el expediente queda sellado.
 * @returns {uuid.UUID} Identificador del expediente.
 * @returns {string} Hash del sello, vacío si no se envió.
 * @throws {error} Si el expediente no se puede guardar.
 */
func upsertSubmission(ctx context.Context, pool *pgxpool.Pool, dataset Dataset, cedula string, estado string, sector string, enviado bool) (uuid.UUID, string, error) {
	cuenta, ok := dataset.Accounts[cedula]
	if !ok {
		return uuid.Nil, "", fmt.Errorf("e2edata: el titular %s no fue sembrado", cedula)
	}

	form := buildForm(cedula, cuenta.Correo, sector)
	// El contenido se valida con el decodificador del dominio antes de
	// guardarlo: un seed que no pase la validación rompería la suite.
	if _, err := submission.DecodeForm(mustMarshal(form)); err != nil {
		return uuid.Nil, "", fmt.Errorf("e2edata: el formulario sembrado no es válido: %w", err)
	}
	raw, err := json.Marshal(form)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo serializar el formulario: %w", err)
	}
	// El sello se calcula con la misma función del dominio: si el seed usara
	// su propio resumen, la verificación del servidor rechazaría el hash.
	canonico, err := submission.CanonicalContent(raw)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("e2edata: el formulario sembrado no es canónico: %w", err)
	}

	var hashSello any
	var enviadoEn any
	if enviado {
		hashSello = submission.HashContent(canonico)
		enviadoEn = time.Now().UTC()
	}

	evidencia := eligibilityEvidence(estado == "closed")
	var id uuid.UUID
	// Se busca el expediente vinculado a la política vigente. El seed
	// idempotente no puede depender de una versión anterior de la política:
	// el expediente con la política correcta es el único que la suite usa.
	err = pool.QueryRow(ctx, `
		SELECT s.id FROM submissions s
		JOIN privacy_bindings b ON b.submission_id = s.id
		WHERE s.applicant_id=$1 AND s.competition_id=$2 AND b.policy_id=$3
		ORDER BY s.created_at DESC
		LIMIT 1`, cuenta.ID, CompetitionID, dataset.PolicyID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// No hay expediente con la política vigente, pero el dominio admite
		// uno solo por cuenta y convocatoria: se busca el que exista para
		// vincularlo a la política actual.
		err = pool.QueryRow(ctx, `
			SELECT id FROM submissions WHERE applicant_id=$1 AND competition_id=$2
			ORDER BY created_at DESC LIMIT 1`, cuenta.ID, CompetitionID).Scan(&id)
	}
	switch {
	case err == nil:
		// El expediente ya existía de una siembra anterior. Un expediente
		// enviado o cerrado es inmutable por diseño: se conserva tal cual.
		// Sólo el borrador se realinea, porque es el único estado que la
		// suite necesita poder volver a fijar.
		var estadoActual string
		if err := pool.QueryRow(ctx, `SELECT status FROM submissions WHERE id=$1`, id).Scan(&estadoActual); err != nil {
			return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo leer el estado del expediente de %s: %w", cedula, err)
		}
		if estadoActual == "draft" {
			if _, err := pool.Exec(ctx, `
				UPDATE submissions SET status=$2, content=$3, hash=$4, submitted_at=$5, eligibility=$6 WHERE id=$1`,
				id, estado, canonico, hashSello, enviadoEn, evidencia); err != nil {
				return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo alinear el expediente de %s: %w", cedula, err)
			}
		}
		// El vínculo de privacidad no se reescribe: si el expediente no lo
		// tiene, se registra con la política vigente.
		var policyActual uuid.UUID
		errVinculo := pool.QueryRow(ctx, `SELECT policy_id FROM privacy_bindings WHERE submission_id=$1`, id).Scan(&policyActual)
		switch {
		case errVinculo == nil && policyActual == dataset.PolicyID:
		case errVinculo == nil:
			// El expediente está vinculado a una política anterior: no se
			// modifica la evidencia, así que la suite trabaja con el
			// expediente que ya está en la política vigente.
			var vigente uuid.UUID
			errVigente := pool.QueryRow(ctx, `
				SELECT s.id FROM submissions s
				JOIN privacy_bindings b ON b.submission_id = s.id
				WHERE s.applicant_id=$1 AND s.competition_id=$2 AND b.policy_id=$3
				ORDER BY s.created_at DESC LIMIT 1`, cuenta.ID, CompetitionID, dataset.PolicyID).Scan(&vigente)
			if errVigente != nil {
				return uuid.Nil, "", fmt.Errorf("e2edata: el expediente %s está vinculado a otra política de privacidad", id)
			}
			id = vigente
		case errors.Is(errVinculo, pgx.ErrNoRows):
			if _, err := pool.Exec(ctx, `
				INSERT INTO privacy_bindings(submission_id,policy_id,policy_hash,accepted_at,retention_review_at)
				VALUES($1,$2,$3,$4,$5)`,
				id, dataset.PolicyID, dataset.PolicyHash, time.Now().UTC(), time.Now().UTC().Add(365*24*time.Hour)); err != nil {
				return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo registrar la aceptación de la política: %w", err)
			}
		default:
			return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo consultar la aceptación de la política: %w", errVinculo)
		}
	case errors.Is(err, pgx.ErrNoRows):
		id, err = crearExpediente(ctx, pool, cuenta, cedula, estado, canonico, hashSello, enviadoEn, evidencia, dataset.PolicyHash)
		if err != nil {
			return uuid.Nil, "", err
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO privacy_bindings(submission_id,policy_id,policy_hash,accepted_at,retention_review_at)
			VALUES($1,$2,$3,$4,$5)`,
			id, dataset.PolicyID, dataset.PolicyHash, time.Now().UTC(), time.Now().UTC().Add(365*24*time.Hour)); err != nil {
			return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo registrar la aceptación de la política: %w", err)
		}
	default:
		return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo consultar el expediente de %s: %w", cedula, err)
	}

	var leido any
	if err := pool.QueryRow(ctx, `SELECT hash FROM submissions WHERE id=$1`, id).Scan(&leido); err != nil {
		return uuid.Nil, "", fmt.Errorf("e2edata: no se pudo leer el sello de %s: %w", cedula, err)
	}
	sello := ""
	if leido != nil {
		sello, _ = leido.(string)
	}
	return id, sello, nil
}

/**
 * codigoExpediente arma un código visible único para el expediente. El
 * sufijo deriva de la versión de la política: el mismo expediente sembrado
 * con la misma política conserva su código, y una política nueva produce
 * un expediente nuevo sin chocar con el anterior.
 *
 * @param {string} cedula - Cédula del titular.
 * @param {string} policyHash - Hash de la política de privacidad vigente.
 * @returns {string} Código visible del expediente.
 */
func codigoExpediente(cedula string, policyHash string) string {
	return fmt.Sprintf("E2E-%s-%s", cedula[:6], policyHash[:12])
}

/**
 * crearExpediente inserta un expediente nuevo con su contenido y su estado.
 *
 * @param {context.Context} ctx - Contexto de la siembra.
 * @param {*pgxpool.Pool} pool - Pool de la base de pruebas.
 * @param {Account} cuenta - Titular del expediente.
 * @param {string} cedula - Cédula del titular.
 * @param {string} estado - Estado del expediente.
 * @param {json.RawMessage} canonico - Contenido canónico del formulario.
 * @param {any} hashSello - Sello criptográfico o nil.
 * @param {any} enviadoEn - Instante de envío o nil.
 * @param {string} evidencia - Evidencia de elegibilidad.
 * @param {string} policyHash - Hash de la política de privacidad vigente.
 * @returns {uuid.UUID} Identificador del expediente creado.
 * @throws {error} Si el expediente no se puede guardar.
 */
func crearExpediente(ctx context.Context, pool *pgxpool.Pool, cuenta Account, cedula string, estado string, canonico json.RawMessage, hashSello any, enviadoEn any, evidencia string, policyHash string) (uuid.UUID, error) {
	submissionID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("e2edata: no se pudo generar el identificador del expediente: %w", err)
	}
	var id uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO submissions(id,applicant_id,code,status,content,hash,submitted_at,competition_id,eligibility)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`,
		submissionID, cuenta.ID, codigoExpediente(cedula, policyHash), estado, canonico, hashSello, enviadoEn,
		CompetitionID, evidencia).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("e2edata: no se pudo sembrar el expediente de %s: %w", cedula, err)
	}
	return id, nil
}

/**
 * eligibilityEvidence construye la evidencia de elegibilidad del expediente.
 *
 * @param {bool} noElegible - Si el expediente está cerrado por no elegibilidad.
 * @returns {string} Evidencia en JSON.
 */
func eligibilityEvidence(noElegible bool) string {
	if !noElegible {
		return `{"elegible":true,"fuente":"e2e","razon":"sector_verificado","mensaje":"Postulante afiliado al IESS"}`
	}
	return `{"elegible":false,"fuente":"e2e","razon":"sector_no_elegible","mensaje":"El postulante no mantiene la calidad de afiliado"}`
}

/**
 * buildForm arma el formulario EF-12 con los datos declarados del titular.
 *
 * El formulario se construye con los structs del dominio: si el seed usara
 * mapas sueltos, el servidor rechazaría el contenido por campos desconocidos
 * y la suite leería un error en vez del expediente.
 *
 * @param {string} cedula - Cédula del titular.
 * @param {string} correo - Correo del titular.
 * @param {string} sector - Sector declarado.
 * @returns {map[string]any} Formulario con las nueve secciones.
 */
func buildForm(cedula string, correo string, sector string) map[string]any {
	form := submission.Form{
		Personal: &submission.PersonalSection{
			FirstName:     "SINTETICO",
			LastName:      Mark,
			IDCard:        cedula,
			Gender:        "Masculino",
			BirthDate:     "1985-05-20",
			Nationality:   "Ecuatoriana",
			MaritalStatus: "Soltero/a",
		},
		Contact: &submission.ContactSection{
			Country:         "Ecuador",
			Province:        "Pichincha",
			City:            "Quito",
			Mobile:          "0987654321",
			Email:           correo,
			MainStreet:      "Av. Amazonas",
			Number:          "N34-56",
			SecondaryStreet: "Av. Republica",
			Area:            "La Carolina",
			Reference:       "Frente al parque",
			Phone:           "0987654321",
		},
		Additional: &submission.AdditionalSection{
			HasRelatives: "No",
			Relatives:    []submission.Relative{},
		},
		Position: &submission.PositionSection{
			Sector:             sector,
			VerificationDetail: "Verificación simulada en pruebas",
			HasOrganization:    "No",
		},
		Education: &submission.EducationSection{
			Degrees: []submission.Degree{{
				ID:           "titulo-1",
				Level:        "Tercer Nivel de Grado",
				Institution:  "Institución de pruebas",
				Type:         "Nacional",
				Title:        Mark,
				Area:         "Administración de Empresas",
				Registry:     "REG-PRUEBA-1",
				RegistryDate: "2015-07-01",
			}},
		},
		Experience: &submission.ExperienceSection{
			Experiences: []submission.Experience{{
				ID:          "experiencia-1",
				Type:        "General",
				Position:    "Analista",
				Institution: "Empresa de pruebas",
				StartDate:   "2018-01-01",
				EndDate:     "2024-01-01",
				Activities:  "Administración de Empresas",
				Functions:   "Funciones de prueba",
			}},
		},
		Publications: &submission.PublicationsSection{
			HasPublications: "Sí",
			Publications: []submission.Publication{{
				ID:      "pub-1",
				Title:   Mark,
				Type:    "Publicación",
				Medium:  "e2e",
				Date:    "2020-01-01",
				Related: "Sí",
			}},
		},
		Training: &submission.TrainingSection{
			Events: []submission.Training{{
				ID:          "curso-1",
				Type:        "Curso",
				Name:        Mark,
				Institution: "e2e",
				StartDate:   "2020-01-01",
				EndDate:     "2020-02-01",
				Hours:       40,
			}},
		},
		Disqualifications: &submission.DisqualificationsSection{
			Q1:  "Sí",
			Q2:  "No",
			Q3:  "Sí",
			Q4:  "No",
			Q5:  "Sí",
			Q6:  "No",
			Q7:  "Sí",
			Q8:  "No",
			Q9:  "Sí",
			Q10: "No",
			Q11: "Sí",
			Q12: "No",
			Q13: "Sí",
		},
		Consent: &submission.Consent{
			DataProcessing:   true,
			Truthfulness:     true,
			CompetitionRules: true,
			NamePublication:  true,
			PhysicalDelivery: true,
		},
	}
	return formMap(form)
}

/**
 * mustMarshal serializa un valor que el dominio ya garantiza serializable.
 *
 * @param {any} valor - Valor a serializar.
 * @returns {json.RawMessage} Contenido serializado.
 */
func mustMarshal(valor any) json.RawMessage {
	raw, err := json.Marshal(valor)
	if err != nil {
		return json.RawMessage("{}")
	}
	return raw
}

/**
 * formMap convierte el formulario del dominio en un mapa genérico para
 * serializarlo. El mapa se devuelve como `any` porque el contenido se
 * canonicaliza igual que el que produce el servidor.
 *
 * @param {submission.Form} form - Formulario del dominio.
 * @returns {map[string]any} Contenido serializable.
 */
func formMap(form submission.Form) map[string]any {
	raw, err := json.Marshal(form)
	if err != nil {
		return map[string]any{}
	}
	var contenido map[string]any
	if err := json.Unmarshal(raw, &contenido); err != nil {
		return map[string]any{}
	}
	return contenido
}
