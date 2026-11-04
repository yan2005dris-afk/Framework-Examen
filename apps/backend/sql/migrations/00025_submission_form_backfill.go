package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	submissionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/postgres"
)

/**
 * 00025 — Fase 4 de la normalización del formulario: rellena las tablas de 00024
 * con el contenido de todas las postulaciones existentes.
 *
 * Reutiliza el adaptador de postulaciones, que habla pgx; goose entrega
 * database/sql. Por eso la migración no usa la transacción de goose: toma la
 * conexión pgx nativa y abre la suya. Sigue siendo atómica: ante cualquier error
 * se revierte todo y goose no registra la versión.
 *
 * Los borradores deben leerse desde las tablas igual que desde `content`, o la
 * migración falla y nombra sus UUID. `submissions` no se modifica: el hash y el
 * snapshot de los expedientes enviados quedan intactos.
 */
func init() {
	goose.AddMigrationNoTxContext(upFormBackfill, downFormBackfill)
}

func upFormBackfill(ctx context.Context, database *sql.DB) error {
	return withPgxTx(ctx, database, func(tx pgx.Tx) error {
		report, err := submissionpostgres.BackfillForms(ctx, tx, time.Now())
		if err != nil {
			return fmt.Errorf("no se pudo normalizar el formulario de las postulaciones: %w", err)
		}
		// Sólo cantidades e identificadores: nunca datos del formulario.
		slog.InfoContext(ctx, "formulario normalizado", "borradores", report.Drafts, "selladas", report.Sealed, "omitidas", len(report.Skipped))
		for _, id := range report.Skipped {
			slog.WarnContext(ctx, "postulación sellada sin descomponer; se conserva su snapshot", "postulacion", id)
		}
		return nil
	})
}

// downFormBackfill vacía las tablas normalizadas; `content` conserva todos los datos.
func downFormBackfill(ctx context.Context, database *sql.DB) error {
	return withPgxTx(ctx, database, func(tx pgx.Tx) error {
		for _, statement := range []string{
			`ALTER TABLE submission_sections DISABLE TRIGGER submission_sections_window`,
			`ALTER TABLE submission_consents DISABLE TRIGGER submission_consents_window`,
			// La cascada desde submission_sections vacía las tablas de cada sección.
			`ALTER TABLE submission_personal DISABLE TRIGGER submission_personal_window`,
			`ALTER TABLE submission_contact DISABLE TRIGGER submission_contact_window`,
			`ALTER TABLE submission_additional DISABLE TRIGGER submission_additional_window`,
			`ALTER TABLE submission_relatives DISABLE TRIGGER submission_relatives_window`,
			`ALTER TABLE submission_position DISABLE TRIGGER submission_position_window`,
			`ALTER TABLE submission_degrees DISABLE TRIGGER submission_degrees_window`,
			`ALTER TABLE submission_experiences DISABLE TRIGGER submission_experiences_window`,
			`ALTER TABLE submission_publications_info DISABLE TRIGGER submission_publications_info_window`,
			`ALTER TABLE submission_publications DISABLE TRIGGER submission_publications_window`,
			`ALTER TABLE submission_trainings DISABLE TRIGGER submission_trainings_window`,
			`ALTER TABLE submission_disqualifications DISABLE TRIGGER submission_disqualifications_window`,
			`ALTER TABLE submission_removal_details DISABLE TRIGGER submission_removal_details_window`,
			`DELETE FROM submission_sections`,
			`DELETE FROM submission_consents`,
		} {
			if _, err := tx.Exec(ctx, statement); err != nil {
				return err
			}
		}
		// Los triggers desactivados dentro de la transacción se reactivan al confirmarla
		// sólo si se vuelven a habilitar explícitamente.
		for _, table := range []string{"submission_sections", "submission_consents", "submission_personal", "submission_contact",
			"submission_additional", "submission_relatives", "submission_position", "submission_degrees", "submission_experiences",
			"submission_publications_info", "submission_publications", "submission_trainings", "submission_disqualifications",
			"submission_removal_details"} {
			if _, err := tx.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s ENABLE TRIGGER %s_window`, table, table)); err != nil {
				return err
			}
		}
		return nil
	})
}

/** withPgxTx ejecuta fn en una transacción pgx sobre una conexión de database/sql. */
func withPgxTx(ctx context.Context, database *sql.DB, fn func(pgx.Tx) error) error {
	conn, err := database.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(driverConn any) error {
		native, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return errors.New("la migración requiere el controlador pgx")
		}
		tx, err := native.Conn().Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		if err := fn(tx); err != nil {
			return err
		}
		return tx.Commit(ctx)
	})
}
