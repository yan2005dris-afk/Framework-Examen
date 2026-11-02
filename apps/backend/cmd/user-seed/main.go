package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/password"
)

/**
 * seedAccount describe una cuenta de prueba. Las cédulas están reservadas
 * para el entorno de desarrollo: son cédulas válidas, por lo que jamás deben
 * existir en un ambiente real.
 */
type seedAccount struct {
	cedula    string
	nombres   string
	apellidos string
	correo    string
	role      string
}

/**
 * accounts son las cuentas de demostración del portal. Ninguna lleva
 * contraseña en el repositorio: el hash se calcula en tiempo de ejecución.
 *
 * La cuenta de administrador no está aquí: la crea la migración 00019 con
 * INIT_ADMIN_PASSWORD, que falla cerrada si la variable no está definida.
 */
var accounts = []seedAccount{
	{cedula: "1710034008", nombres: "Damian", apellidos: "Postulante", correo: "postulante@biess.local", role: "postulante"},
	{cedula: "1710034016", nombres: "Elena", apellidos: "Analista", correo: "analista@biess.local", role: "analista"},
}

/** adminCedula identifica la cuenta que aprueba la convocatoria de desarrollo. */
const adminCedula = "0910034008"

/** competitionCode identifica la convocatoria de demostración. */
const competitionCode = "convocatoria-2026-01"

/**
 * main inyecta las cuentas de prueba del entorno de desarrollo. Se niega a
 * ejecutarse en producción y nunca sobrescribe una cuenta existente.
 */
func main() {
	resetPassword := flag.Bool("reset-password", false, "rota la contraseña de las cuentas existentes en lugar de omitirlas")
	flag.Parse()

	if err := rejectProduction(); err != nil {
		log.Fatalf("No se pudo inyectar la semilla de usuarios: %v", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL es obligatoria para inyectar la semilla de usuarios.")
	}

	secret, generated, err := resolvePassword()
	if err != nil {
		log.Fatalf("No se pudo determinar la contraseña de la semilla: %v", err)
	}
	hash, err := password.Hash(secret)
	if err != nil {
		log.Fatalf("No se pudo calcular el hash de la semilla: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("No se pudo conectar a PostgreSQL (%s): %v", databaseURL, err)
	}
	defer conn.Close(context.Background())

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("No se pudo iniciar transacción: %v", err)
	}
	defer tx.Rollback(context.Background())

	created, skipped, err := apply(ctx, tx, hash, secret, *resetPassword)
	if err != nil {
		log.Fatalf("Error al ejecutar la semilla de usuarios: %v", err)
	}

	if err := seedCompetition(ctx, tx); err != nil {
		log.Fatalf("Error al inyectar la convocatoria de desarrollo: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("Error al confirmar la semilla de usuarios: %v", err)
	}

	report(created, skipped, secret, generated, *resetPassword)
}

/**
 * rejectProduction impide que la semilla de desarrollo se ejecute contra un
 * ambiente real. Una cuenta de administrador con credencial conocida nunca
 * debe depender de que alguien recuerde no correr un comando.
 *
 * @returns {error} Error explicativo si el entorno es de producción.
 */
func rejectProduction() error {
	environment := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if environment == "production" || environment == "produccion" {
		return errors.New("cmd/user-seed es una herramienta de desarrollo y no puede ejecutarse con APP_ENV de producción; en un ambiente real el alta se hace por el registro OTP y cmd/applicant-role")
	}
	return nil
}

/**
 * resolvePassword obtiene la contraseña de la semilla desde SEED_PASSWORD o
 * genera una aleatoria. Nunca se toma de un literal del repositorio.
 *
 * @returns {string, bool, error} Contraseña, indicación de si fue generada y error.
 */
func resolvePassword() (string, bool, error) {
	if secret := os.Getenv("SEED_PASSWORD"); strings.TrimSpace(secret) != "" {
		if err := password.ValidateStrength(secret); err != nil {
			return "", false, fmt.Errorf("SEED_PASSWORD no cumple las reglas de contraseña: %w", err)
		}
		return secret, false, nil
	}
	secret, err := generatePassword()
	if err != nil {
		return "", false, err
	}
	return secret, true, nil
}

/**
 * apply inserta las cuentas que no existen y, sólo con reset, rota la
 * contraseña de las que ya están. Nunca pisa credenciales ajenas.
 *
 * @param {pgx.Tx} tx - Transacción abierta.
 * @param {string} hash - Hash Argon2id de la contraseña de la semilla.
 * @param {string} secret - Contraseña en claro, sólo para informar al operador.
 * @param {bool} reset - Rota la contraseña de las cuentas existentes.
 * @returns {[]seedAccount, []seedAccount, error} Cuentas creadas, omitidas y error.
 */
func apply(ctx context.Context, tx pgx.Tx, hash string, secret string, reset bool) ([]seedAccount, []seedAccount, error) {
	created := []seedAccount{}
	skipped := []seedAccount{}
	for _, account := range accounts {
		// El identificador se genera aquí para distinguir sin heurísticas si la
		// fila se acaba de insertar o ya existía.
		proposed, err := uuid.NewV7()
		if err != nil {
			return nil, nil, fmt.Errorf("no se pudo generar el identificador de %s: %w", account.cedula, err)
		}

		var stored uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO applicants (id, cedula, nombres, apellidos, correo, metodo_validacion, fecha_registro, password_hash, is_verified, role)
			VALUES ($1, $2, $3, $4, $5, 'otp_correo', now(), $6, true, $7)
			ON CONFLICT (cedula) DO UPDATE SET
				nombres = EXCLUDED.nombres,
				apellidos = EXCLUDED.apellidos,
				metodo_validacion = EXCLUDED.metodo_validacion,
				is_verified = EXCLUDED.is_verified,
				role = EXCLUDED.role,
				updated_at = now()
			RETURNING id`,
			proposed, account.cedula, account.nombres, account.apellidos, account.correo, hash, account.role).Scan(&stored)
		if err != nil {
			return nil, nil, describeConflict(err, account)
		}

		// Una cuenta recién insertada ya trae la contraseña de la semilla.
		// Una cuenta preexistente conserva la suya: sólo se rota si el
		// operador lo pide de forma explícita.
		if stored == proposed {
			created = append(created, account)
			continue
		}
		if !reset {
			skipped = append(skipped, account)
			continue
		}

		tag, err := tx.Exec(ctx, `
			UPDATE applicants
			SET password_hash = $1, updated_at = now()
			WHERE cedula = $2`, hash, account.cedula)
		if err != nil {
			return nil, nil, fmt.Errorf("no se pudo rotar la contraseña de %s: %w", account.cedula, err)
		}
		if tag.RowsAffected() == 0 {
			skipped = append(skipped, account)
			continue
		}
		created = append(created, account)
	}
	return created, skipped, nil
}

/**
 * seedCompetition crea la convocatoria de desarrollo con sus controles, sus
 * fases y la política de privacidad asociada. La cuenta que aprueba es la del
 * administrador, que crea la migración 00019: si falta, la política no puede
 * insertarse y la operación falla en vez de quedar a medias.
 *
 * @param {pgx.Tx} tx - Transacción abierta.
 * @returns {error} Error si la convocatoria no puede sembrarse.
 */
func seedCompetition(ctx context.Context, tx pgx.Tx) error {
	var adminID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM applicants WHERE cedula = $1`, adminCedula).Scan(&adminID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("no existe la cuenta de administrador; levante la API una vez para que la migración 00019 la cree con INIT_ADMIN_PASSWORD")
	}
	if err != nil {
		return fmt.Errorf("no se pudo localizar la cuenta de administrador: %w", err)
	}

	const competition = `
INSERT INTO competitions (id, code, title, opens_at, closes_at)
VALUES (uuidv7(), $1, $2, $3, $4)
ON CONFLICT (code) DO UPDATE SET
	title = EXCLUDED.title,
	opens_at = EXCLUDED.opens_at,
	closes_at = EXCLUDED.closes_at`
	if _, err := tx.Exec(ctx, competition,
		competitionCode,
		"Concurso Público de Méritos y Oposición BIESS 2026",
		"2026-01-05 05:00:00+00",
		"2027-01-01 04:59:59.999999+00",
	); err != nil {
		return fmt.Errorf("no se pudo crear la convocatoria: %w", err)
	}

	const controls = `
INSERT INTO competition_controls (
	competition_id, version, approved, published, form_enabled,
	sectors, phases, approved_at, approved_by
) VALUES (
	(SELECT id FROM competitions WHERE code = $1),
	2, true, true, true,
	ARRAY['afiliados', 'jubilados', 'empleadores'],
	$2::jsonb,
	now(), $3
)
ON CONFLICT (competition_id) DO UPDATE SET
	phases = EXCLUDED.phases,
	approved = EXCLUDED.approved,
	published = EXCLUDED.published,
	form_enabled = EXCLUDED.form_enabled`
	if _, err := tx.Exec(ctx, controls, competitionCode, developmentPhases, adminID); err != nil {
		return fmt.Errorf("no se pudieron crear los controles de la convocatoria: %w", err)
	}

	const policy = `
INSERT INTO privacy_policies (id, competition_id, version, definition, hash, actor_id, created_at)
VALUES (uuidv7(), (SELECT id FROM competitions WHERE code = $1), 2, $2::jsonb, $3, $4, now())
ON CONFLICT (competition_id, version) DO NOTHING`
	if _, err := tx.Exec(ctx, policy, competitionCode, developmentPolicy, developmentPolicyHash, adminID); err != nil {
		return fmt.Errorf("no se pudo crear la política de privacidad: %w", err)
	}

	return nil
}

/**
 * describeConflict traduce una violación de unicidad a un mensaje que no
 * expone datos de la cuenta afectada.
 *
 * @param {error} err - Error devuelto por PostgreSQL.
 * @param {seedAccount} account - Cuenta que se intentaba insertar.
 * @returns {error} Error explicativo.
 */
func describeConflict(err error, account seedAccount) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		if strings.Contains(pgError.ConstraintName, "correo") {
			return fmt.Errorf("ya existe una cuenta registrada con el correo de %s; elija otro correo para la semilla", account.cedula)
		}
		return fmt.Errorf("ya existe una cuenta registrada con la cédula %s", account.cedula)
	}
	return fmt.Errorf("no se pudo insertar la cuenta %s: %w", account.cedula, err)
}

/**
 * report informa el resultado de la semilla. La contraseña sólo se muestra si
 * la cuenta se creó o se rotó en esta ejecución.
 *
 * @param {[]seedAccount} created - Cuentas creadas o rotadas.
 * @param {[]seedAccount} skipped - Cuentas que ya existían.
 * @param {string} secret - Contraseña de la semilla.
 * @param {bool} generated - Indica si la contraseña fue generada.
 * @param {bool} reset - Indica si la operación fue una rotación.
 */
func report(created []seedAccount, skipped []seedAccount, secret string, generated bool, reset bool) {
	fmt.Println("Semilla de usuarios y convocatoria aplicada:")
	for _, account := range created {
		fmt.Printf("  - %-14s %s / %s (%s)\n", account.cedula, account.correo, account.role, account.nombres)
	}
	if len(skipped) > 0 {
		fmt.Println("Cuentas ya existentes: se repararon sus datos no secretos y se conservó la contraseña:")
		for _, account := range skipped {
			fmt.Printf("  - %-14s %s (%s)\n", account.cedula, account.correo, account.role)
		}
	}
	fmt.Printf("  - Convocatoria:  %s\n", competitionCode)
	if len(created) == 0 {
		fmt.Println("No se creó ni se rotó ninguna cuenta. Use -reset-password para rotar las claves existentes.")
		return
	}
	if generated {
		fmt.Printf("\nContraseña generada para esta ejecución: %s\n", secret)
		fmt.Println("Guárdela: no se vuelve a mostrar y no está en el repositorio.")
	} else {
		fmt.Println("\nContraseña tomada de SEED_PASSWORD.")
	}
	if reset {
		fmt.Println("Las contraseñas existentes fueron rotadas por el flag -reset-password.")
	}
}

/**
 * generatePassword crea una contraseña aleatoria criptográfica que cumple
 * las reglas de RN-02: mayúscula, minúscula, número y símbolo.
 *
 * @returns {string, error} Contraseña generada y error.
 */
func generatePassword() (string, error) {
	groups := []string{
		"abcdefghijkmnopqrstuvwxyz",
		"ABCDEFGHJKLMNPQRSTUVWXYZ",
		"23456789",
		"!@#$%&*?",
	}
	alphabet := strings.Join(groups, "")
	out := make([]byte, 0, 16)
	for _, group := range groups {
		choice, err := randomChoice(group)
		if err != nil {
			return "", err
		}
		out = append(out, choice)
	}
	for i := 0; i < 12; i++ {
		choice, err := randomChoice(alphabet)
		if err != nil {
			return "", err
		}
		out = append(out, choice)
	}
	for i := len(out) - 1; i > 0; i-- {
		j, err := randomIndex(i + 1)
		if err != nil {
			return "", err
		}
		out[i], out[j] = out[j], out[i]
	}
	return string(out), nil
}

/**
 * randomChoice elige un carácter del conjunto indicado.
 *
 * @param {string} chars - Conjunto de caracteres disponibles.
 * @returns {byte, error} Carácter elegido y error.
 */
func randomChoice(chars string) (byte, error) {
	index, err := randomIndex(len(chars))
	if err != nil {
		return 0, err
	}
	return chars[index], nil
}

/**
 * randomIndex devuelve un índice aleatorio criptográfico menor que max.
 *
 * @param {int} max - Límite exclusive.
 * @returns {int, error} Índice y error.
 */
func randomIndex(max int) (int, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}
