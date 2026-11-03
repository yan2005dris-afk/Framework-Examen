// Package migrations expone las migraciones embebidas de la base de datos.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed *.sql
var files embed.FS

/**
 * open conecta con la base de datos y prepara el conjunto de migraciones
 * embebido.
 *
 * @param {string} databaseURL - Cadena de conexión a PostgreSQL.
 * @returns {sql.DB} Conexión lista para ejecutar migraciones.
 * @throws {Error} Si no se puede abrir la conexión.
 */
func open(databaseURL string) (*sql.DB, error) {
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir la conexión para migraciones: %w", err)
	}
	return database, nil
}

/**
 * provider construye el ejecutor de migraciones con bloqueo consultivo.
 *
 * El bloqueo serializa a varios procesos que arrancan a la vez —varias
 * réplicas de la API en un despliegue— para que no apliquen la misma
 * migración en paralelo. La API legacy de goose no coordina, de modo que sin
 * este bloqueo dos procesos pueden competir por la misma versión.
 *
 * @param {sql.DB} database - Conexión abierta.
 * @returns {*goose.Provider} Ejecutor con las migraciones embebidas.
 * @throws {Error} Si no se puede construir el ejecutor.
 */
func provider(database *sql.DB) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("no se pudo preparar el bloqueo de migraciones: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, database, files, goose.WithSessionLocker(locker))
	if err != nil {
		return nil, fmt.Errorf("no se pudo configurar el ejecutor de migraciones: %w", err)
	}
	return provider, nil
}

/**
 * Run aplica las migraciones pendientes en la base de datos indicada.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {string} databaseURL - Cadena de conexión a PostgreSQL.
 * @returns {void}
 * @throws {Error} Si alguna migración falla.
 */
func Run(ctx context.Context, databaseURL string) error {
	database, err := open(databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	if err := database.PingContext(ctx); err != nil {
		return fmt.Errorf("no hay conexión con la base de datos para migrar: %w", err)
	}

	runner, err := provider(database)
	if err != nil {
		return err
	}

	if _, err := runner.Up(ctx); err != nil {
		return fmt.Errorf("no se pudieron aplicar las migraciones: %w", err)
	}

	return nil
}

/**
 * RunTo permite preparar un fixture histórico de QA antes de las restricciones
 * nuevas; nunca revierte migraciones.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {string} databaseURL - Cadena de conexión a PostgreSQL.
 * @param {int64} version - Versión objetivo.
 * @returns {void}
 * @throws {Error} Si no se puede aplicar la migración.
 */
func RunTo(ctx context.Context, databaseURL string, version int64) error {
	database, err := open(databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	runner, err := provider(database)
	if err != nil {
		return err
	}

	if _, err := runner.UpTo(ctx, version); err != nil {
		return fmt.Errorf("no se pudo aplicar la versión %d de migraciones: %w", version, err)
	}
	return nil
}

/**
 * Down revierte la última migración aplicada.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {string} databaseURL - Cadena de conexión a PostgreSQL.
 * @returns {void}
 * @throws {Error} Si la reversión falla.
 */
func Down(ctx context.Context, databaseURL string) error {
	database, err := open(databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	runner, err := provider(database)
	if err != nil {
		return err
	}

	if _, err := runner.Down(ctx); err != nil {
		return fmt.Errorf("no se pudo revertir la última migración aplicada: %w", err)
	}

	return nil
}

/**
 * Status muestra el estado de cada migración conocida.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {string} databaseURL - Cadena de conexión a PostgreSQL.
 * @returns {void}
 * @throws {Error} Si no se puede leer el estado.
 */
func Status(ctx context.Context, databaseURL string) error {
	database, err := open(databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	runner, err := provider(database)
	if err != nil {
		return err
	}

	statuses, err := runner.Status(ctx)
	if err != nil {
		return fmt.Errorf("no se pudo leer el estado de las migraciones: %w", err)
	}

	for _, status := range statuses {
		fmt.Printf("%s  %s\n", strconv.FormatInt(status.Source.Version, 10), status.State)
	}
	return nil
}
