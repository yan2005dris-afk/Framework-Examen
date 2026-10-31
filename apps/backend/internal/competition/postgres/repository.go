package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"time"
)

type Repository struct {
	queries *sqlcgen.Queries
	tx      *db.TxManager
	audit   audit.WorkflowRecorder
}

func NewRepository(q *sqlcgen.Queries, tx *db.TxManager, recorder audit.WorkflowRecorder) *Repository {
	return &Repository{q, tx, recorder}
}
func (r *Repository) record(ctx context.Context, q *sqlcgen.Queries, c *competition.Competition, actor uuid.UUID, action string) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err = q.RecordCompetitionVersion(ctx, sqlcgen.RecordCompetitionVersionParams{CompetitionID: c.ID, Version: c.Version, Configuration: raw, ActorID: actor}); err != nil {
		return err
	}
	if r.audit == nil {
		return audit.ErrUnavailable
	}
	return r.audit.RecordWorkflow(ctx, audit.Event{ActorID: actor, Role: "administrador", Action: action, ResourceID: c.ID, ResourceType: "convocatoria", Result: "permitido", CreatedAt: time.Now().UTC()})
}
func (r *Repository) Create(ctx context.Context, c *competition.Competition, actor uuid.UUID) error {
	return r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries.WithTx(tx)
		if err := verifyAdministrator(ctx, q, actor); err != nil {
			return err
		}
		if err := q.CreateCompetitionLifecycle(ctx, sqlcgen.CreateCompetitionLifecycleParams{ID: c.ID, Code: c.Code, Title: c.Title, OpensAt: pgtype.Timestamptz{Time: c.OpensAt, Valid: true}, ClosesAt: pgtype.Timestamptz{Time: c.ClosesAt, Valid: true}}); err != nil {
			var constraint *pgconn.PgError
			if errors.As(err, &constraint) && constraint.Code == "23505" && constraint.ConstraintName == "competitions_code_key" {
				return competition.ErrDuplicate
			}
			return err
		}
		phases, err := json.Marshal(c.Phases)
		if err != nil {
			return err
		}
		if err = q.CreateCompetitionControl(ctx, sqlcgen.CreateCompetitionControlParams{CompetitionID: c.ID, Sectors: c.Sectors, Phases: phases}); err != nil {
			return err
		}
		c.Version = 1
		return r.record(ctx, q, c, actor, "convocatoria.creada")
	})
}
func decode(row sqlcgen.GetCompetitionLifecycleRow) (*competition.Competition, error) {
	c := &competition.Competition{ID: row.ID, Code: row.Code, Title: row.Title, OpensAt: row.OpensAt.Time, ClosesAt: row.ClosesAt.Time, Configuration: competition.Configuration{Version: row.Version, Approved: row.Approved, Published: row.Published, FormEnabled: row.FormEnabled, Sectors: row.Sectors}}
	if err := json.Unmarshal(row.Phases, &c.Phases); err != nil {
		return nil, err
	}
	if row.ApprovedAt.Valid {
		t := row.ApprovedAt.Time
		c.ApprovedAt = &t
	}
	if row.ApprovedBy.Valid {
		id := uuid.UUID(row.ApprovedBy.Bytes)
		c.ApprovedBy = &id
	}
	return c, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*competition.Competition, error) {
	row, err := r.queries.GetCompetitionLifecycle(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, competition.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return decode(row)
}
func (r *Repository) Configure(ctx context.Context, id uuid.UUID, config competition.Configuration, actor uuid.UUID) (result *competition.Competition, err error) {
	err = r.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		ctx = db.WithTransaction(ctx, tx)
		q := r.queries.WithTx(tx)
		if err := verifyAdministrator(ctx, q, actor); err != nil {
			return err
		}
		if err := q.LockCompetitionControl(ctx, id); err != nil {
			return err
		}
		row, err := q.GetCompetitionLifecycle(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return competition.ErrNotFound
		}
		if err != nil {
			return err
		}
		current, err := decode(row)
		if err != nil {
			return err
		}
		if current.Version != config.Version {
			return competition.ErrVersion
		}
		current.Configuration = config
		if err = current.Validate(); err != nil {
			return err
		}
		raw, err := json.Marshal(config.Phases)
		if err != nil {
			return err
		}
		if err = q.UpdateCompetitionControl(ctx, sqlcgen.UpdateCompetitionControlParams{CompetitionID: id, Approved: config.Approved, Published: config.Published, FormEnabled: config.FormEnabled, Sectors: config.Sectors, Phases: raw, ApprovedBy: pgtype.UUID{Bytes: actor, Valid: true}}); err != nil {
			return err
		}
		row, err = q.GetCompetitionLifecycle(ctx, id)
		if err != nil {
			return err
		}
		result, err = decode(row)
		if err != nil {
			return err
		}
		return r.record(ctx, q, result, actor, "convocatoria.configurada")
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
func (r *Repository) ListPublished(ctx context.Context, limit int, before uuid.UUID) ([]competition.Competition, error) {
	rows, err := r.queries.ListPublicCompetitions(ctx, sqlcgen.ListPublicCompetitionsParams{Limit: int32(limit), BeforeID: pgtype.UUID{Bytes: before, Valid: before != uuid.Nil}})
	if err != nil {
		return nil, err
	}
	items := make([]competition.Competition, 0, len(rows))
	for _, row := range rows {
		c, err := decode(sqlcgen.GetCompetitionLifecycleRow(row))
		if err != nil {
			return nil, err
		}
		items = append(items, *c)
	}
	return items, nil
}

/** verifyAdministrator bloquea la cuenta para serializar la revocación del rol y esta operación. */
func verifyAdministrator(ctx context.Context, q *sqlcgen.Queries, actor uuid.UUID) error {
	rows, err := q.LockApplicantsForRoleChange(ctx, []uuid.UUID{actor})
	if err != nil {
		return err
	}
	if len(rows) != 1 || rows[0].Role != "administrador" {
		return competition.ErrForbidden
	}
	return nil
}
