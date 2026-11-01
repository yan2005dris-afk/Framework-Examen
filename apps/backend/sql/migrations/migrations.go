// Package migrations expone las migraciones embebidas de la base de datos.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

/**
 * open conecta con la base de datos y prepara goose con el conjunto de
 * migraciones embebido.
 *
 * @param {string} databaseURL - Cadena de conexión a PostgreSQL.
 * @returns {sql.DB} Conexión lista para ejecutar migraciones.
 * @throws {Error} Si no se puede abrir la conexión o configurar goose.
 */
func open(databaseURL string) (*sql.DB, error) {
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir la conexión para migraciones: %w", err)
	}

	goose.SetBaseFS(files)
	if err := goose.SetDialect("postgres"); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("no se pudo configurar el dialecto de migraciones: %w", err)
	}

	return database, nil
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

	if err := goose.UpContext(ctx, database, "."); err != nil {
		return fmt.Errorf("no se pudieron aplicar las migraciones: %w", err)
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

	if err := goose.DownContext(ctx, database, "."); err != nil {
		return fmt.Errorf("no se pudo revertir la última migración: %w", err)
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

	if err := goose.StatusContext(ctx, database, "."); err != nil {
		return fmt.Errorf("no se pudo leer el estado de las migraciones: %w", err)
	}

	return nil
}

/** RunTo permite preparar un fixture histórico de QA antes de las restricciones nuevas; nunca revierte migraciones. */
func RunTo(ctx context.Context, databaseURL string, version int64) error {
	database, err := open(databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()
	return goose.UpToContext(ctx, database, ".", version)
}
