package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/password"
)

/**
 * 00021_seed_admin.go
 *
 * Alta inicial del administrador de plataforma. Se aplica una sola vez, junto
 * con las migraciones de esquema, porque goose registra la versión aplicada:
 * reiniciar la API no vuelve a sembrar la cuenta ni rota su contraseña.
 *
 * De dónde sale la contraseña:
 *   - $INIT_ADMIN_PASSWORD, obligatoria. Si falta, la migración falla y el
 *     arranque se aborta con un error explícito, en vez de sembrar una
 *     credencial conocida que viva en el repositorio.
 *
 * El hash se calcula en tiempo de ejecución con el mismo Argon2id que usa el
 * registro de postulantes (internal/platform/password), con sal aleatoria
 * nueva por ambiente y formato PHC estándar.
 */

const (
	defaultAdminCedula   = "0910034008"
	defaultAdminCorreo   = "admin@biess.local"
	defaultAdminNombres  = "Carlos"
	defaultAdminApellido = "Administrador"
)

func init() {
	goose.AddMigrationContext(upSeedAdmin, downSeedAdmin)
}

/**
 * upSeedAdmin crea la cuenta del administrador con la contraseña del entorno.
 *
 * @param {context.Context} ctx - Contexto de la migración.
 * @param {sql.Tx} tx - Transacción de la migración.
 * @returns {error} Error si falta la contraseña o si la escritura falla.
 */
func upSeedAdmin(ctx context.Context, tx *sql.Tx) error {
	secret := os.Getenv("INIT_ADMIN_PASSWORD")
	if secret == "" {
		return errors.New("seed admin: INIT_ADMIN_PASSWORD es obligatoria (no hay valor por defecto; el arranque se aborta antes que sembrar una credencial conocida)")
	}
	if err := password.ValidateStrength(secret); err != nil {
		return fmt.Errorf("seed admin: INIT_ADMIN_PASSWORD no cumple las reglas de contraseña: %w", err)
	}

	hash, err := password.Hash(secret)
	if err != nil {
		return fmt.Errorf("seed admin: no se pudo calcular el hash: %w", err)
	}

	cedula := envOr("INIT_ADMIN_CEDULA", defaultAdminCedula)
	correo := envOr("INIT_ADMIN_CORREO", defaultAdminCorreo)
	nombres := envOr("INIT_ADMIN_NOMBRES", defaultAdminNombres)
	apellidos := envOr("INIT_ADMIN_APELLIDOS", defaultAdminApellido)

	// El ON CONFLICT sólo alcanzaría a esta fila si la cédula ya estuviera
	// registrada; el cambio de rol queda auditado por el trigger de la
	// migración 00007 con el operador de la base de datos.
	const upsert = `
INSERT INTO applicants (
    id, cedula, nombres, apellidos, correo, metodo_validacion,
    fecha_registro, password_hash, is_verified, role
) VALUES (
    uuidv7(), $1, $2, $3, $4, 'otp_correo',
    now(), $5, true, 'administrador'
)
ON CONFLICT (cedula) DO UPDATE SET
    nombres = EXCLUDED.nombres,
    apellidos = EXCLUDED.apellidos,
    correo = EXCLUDED.correo,
    metodo_validacion = EXCLUDED.metodo_validacion,
    password_hash = EXCLUDED.password_hash,
    is_verified = true,
    role = 'administrador',
    updated_at = now()`

	if _, err := tx.ExecContext(ctx, upsert, cedula, nombres, apellidos, correo, hash); err != nil {
		return fmt.Errorf("seed admin: no se pudo escribir la cuenta: %w", err)
	}
	return nil
}

/**
 * downSeedAdmin retira únicamente la fila que administra esta migración; las
 * cuentas creadas por el registro de postulantes no se tocan.
 *
 * @param {context.Context} ctx - Contexto de la migración.
 * @param {sql.Tx} tx - Transacción de la migración.
 * @returns {error} Error si la eliminación falla.
 */
func downSeedAdmin(ctx context.Context, tx *sql.Tx) error {
	cedula := envOr("INIT_ADMIN_CEDULA", defaultAdminCedula)
	if _, err := tx.ExecContext(ctx, `DELETE FROM applicants WHERE cedula = $1`, cedula); err != nil {
		return fmt.Errorf("seed admin: no se pudo eliminar la cuenta: %w", err)
	}
	return nil
}

/**
 * envOr lee una variable del entorno y, si falta, devuelve el valor por
 * defecto indicado.
 *
 * @param {string} key - Nombre de la variable de entorno.
 * @param {string} fallback - Valor por defecto.
 * @returns {string} Valor de la variable o el valor por defecto.
 */
func envOr(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
