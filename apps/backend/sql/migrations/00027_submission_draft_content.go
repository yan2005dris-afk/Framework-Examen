package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	submissionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/postgres"
)

/**
 * 00027 — Fase 5 de la normalización del formulario: los borradores dejan de
 * guardar `content`; su única fuente son las tablas de 00024.
 *
 * Antes de vaciarlo se repite el relleno verificado de 00025, que es
 * idempotente: un borrador guardado entre ambas migraciones queda comprobado
 * byte a byte antes de perder su copia. La reversión reconstruye `content`
 * desde las tablas, de modo que bajar de versión no pierde datos.
 */
func init() {
	goose.AddMigrationNoTxContext(upDraftContent, downDraftContent)
}

func upDraftContent(ctx context.Context, database *sql.DB) error {
	return withPgxTx(ctx, database, func(tx pgx.Tx) error {
		report, err := submissionpostgres.BackfillForms(ctx, tx, time.Now())
		if err != nil {
			return fmt.Errorf("no se pudo verificar el formulario antes de vaciar content: %w", err)
		}
		// Los borradores de convocatorias ya cerradas también se vacían.
		if _, err := tx.Exec(ctx, `ALTER TABLE submissions DISABLE TRIGGER submissions_window`); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE submissions SET content = NULL WHERE status IN ('draft','in_review') AND content IS NOT NULL`)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `ALTER TABLE submissions ENABLE TRIGGER submissions_window`); err != nil {
			return err
		}
		slog.InfoContext(ctx, "content reservado para el snapshot", "borradores", report.Drafts, "vaciados", tag.RowsAffected())
		return nil
	})
}

func downDraftContent(ctx context.Context, database *sql.DB) error {
	return withPgxTx(ctx, database, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id, status FROM submissions WHERE status IN ('draft','in_review') AND content IS NULL`)
		if err != nil {
			return err
		}
		type draft struct {
			id     uuid.UUID
			status string
		}
		drafts := []draft{}
		for rows.Next() {
			var d draft
			if err := rows.Scan(&d.id, &d.status); err != nil {
				rows.Close()
				return err
			}
			drafts = append(drafts, d)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `ALTER TABLE submissions DISABLE TRIGGER submissions_window`); err != nil {
			return err
		}
		repo := submissionpostgres.NewRepository(sqlcgen.New(tx))
		for _, d := range drafts {
			item := &submission.Submission{ID: d.id, Status: submission.Status(d.status)}
			if err := repo.Hydrate(ctx, item); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE submissions SET content = $2 WHERE id = $1`, d.id, json.RawMessage(item.Content)); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `ALTER TABLE submissions ENABLE TRIGGER submissions_window`)
		return err
	})
}
