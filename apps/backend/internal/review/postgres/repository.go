// Package postgres adapta las consultas de revisión sin mezclar perfiles vigentes y snapshots.
package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/review"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
)

/** Repository utiliza SQLC como único adaptador SQL del contexto. */
type Repository struct {
	queries      *sqlcgen.Queries
	transactions *db.TxManager
}

/** NewRepository recibe el ejecutor generado de las consultas. */
func NewRepository(queries *sqlcgen.Queries, transactions ...*db.TxManager) *Repository {
	r := &Repository{queries: queries}
	if len(transactions) > 0 {
		r.transactions = transactions[0]
	}
	return r
}

/** Export comprueba el presupuesto antes de cargar JSON y materializa una sola instantánea de lectura. */
func (r *Repository) Export(ctx context.Context, f review.Filter) (items []review.Snapshot, err error) {
	if r.transactions == nil {
		return nil, errors.New("transacciones de revisión no configuradas")
	}
	err = r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY"); err != nil {
			return err
		}
		queries := r.queries.WithTx(tx)
		budget, err := queries.ReviewExportBudget(ctx, sqlcgen.ReviewExportBudgetParams{CompetitionID: periodID(f.CompetitionID), Sector: f.Sector, Status: f.Status})
		if err != nil {
			return err
		}
		if budget.Total > review.MaxExportRows || budget.Bytes > review.MaxExportBytes {
			return review.ErrExportTooLarge
		}
		items, err = (&Repository{queries: queries}).List(ctx, f, domain.Position{}, review.MaxExportRows+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func competition(row sqlcgen.Competition) review.Competition {
	return review.Competition{ID: row.ID, Code: row.Code, Title: row.Title,
		OpensAt: row.OpensAt.Time, ClosesAt: row.ClosesAt.Time, CreatedAt: row.CreatedAt.Time}
}

/** Competition devuelve el período exacto sin sustituirlo por la convocatoria activa. */
func (r *Repository) Competition(ctx context.Context, id uuid.UUID) (*review.Competition, error) {
	row, err := r.queries.GetReviewCompetition(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	value := competition(row)
	return &value, nil
}

/** Competitions lista también los períodos cerrados con un cursor estable. */
func (r *Repository) Competitions(ctx context.Context, p domain.Position, limit int) ([]review.Competition, error) {
	rows, err := r.queries.ListReviewCompetitions(ctx, sqlcgen.ListReviewCompetitionsParams{HasCursor: !p.At.IsZero(),
		BeforeAt: pgtype.Timestamptz{Time: p.At, Valid: true}, BeforeID: p.ID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := make([]review.Competition, 0, len(rows))
	for _, row := range rows {
		items = append(items, competition(row))
	}
	return items, nil
}

func periodID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

/** List usa una única sentencia y una selección exacta para paginación o exportación completa. */
func (r *Repository) List(ctx context.Context, f review.Filter, p domain.Position, limit int) ([]review.Snapshot, error) {
	rows, err := r.queries.ListReviewSubmissions(ctx, sqlcgen.ListReviewSubmissionsParams{CompetitionID: periodID(f.CompetitionID),
		Sector: f.Sector, Status: f.Status, HasCursor: !p.At.IsZero(), BeforeAt: pgtype.Timestamptz{Time: p.At, Valid: true}, BeforeID: p.ID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := make([]review.Snapshot, 0, len(rows))
	for _, row := range rows {
		hash := ""
		if row.Hash != nil {
			hash = *row.Hash
		}
		items = append(items, review.Snapshot{ID: row.ID, Code: row.Code, CompetitionCode: row.CompetitionCode, Status: row.Status,
			Content: row.Content, Hash: hash, SubmittedAt: row.SubmittedAt.Time, CreatedAt: row.CreatedAt})
	}
	return items, nil
}

/** Counts no cachea los conteos ni incluye borradores o expedientes de otro período. */
func (r *Repository) Counts(ctx context.Context, f review.Filter) ([]review.Count, error) {
	rows, err := r.queries.CountReviewSubmissions(ctx, sqlcgen.CountReviewSubmissionsParams{CompetitionID: periodID(f.CompetitionID), Sector: f.Sector, Status: f.Status})
	if err != nil {
		return nil, err
	}
	items := make([]review.Count, 0, len(rows))
	for _, row := range rows {
		items = append(items, review.Count{Sector: row.Sector, Status: row.Status, Applicants: row.Applicants, Submissions: row.Total})
	}
	return items, nil
}
