// Package postgres es el adaptador de persistencia del contexto de
// postulaciones sobre PostgreSQL. Habla SQL mediante las queries generadas
// por sqlc y traduce los errores de la base de datos a errores de dominio.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

/**
 * Repository es el adaptador de persistencia de postulaciones.
 */
type Repository struct {
	queries      *sqlcgen.Queries
	transactions *db.TxManager
}

/**
 * NewRepository crea el adaptador con las queries generadas por sqlc.
 *
 * @param {sqlcgen.Queries} queries - Queries generadas desde sql/queries.
 * @returns {Repository} Adaptador listo para usar.
 */
func NewRepository(queries *sqlcgen.Queries, txManager ...*db.TxManager) *Repository {
	r := &Repository{queries: queries}
	if len(txManager) > 0 {
		r.transactions = txManager[0]
	}
	return r
}

/**
 * Create inserta una postulación nueva.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {Submission} item - Postulación a persistir.
 * @returns {void}
 * @throws {Error} ErrDuplicateCode o ErrInvalidApplicant según la restricción violada.
 */
func (r *Repository) Create(ctx context.Context, item *submission.Submission) error {
	params := sqlcgen.CreateSubmissionParams{
		ID:          item.ID,
		ApplicantID: item.ApplicantID,
		Code:        item.Code,
		Status:      string(item.Status),
		Content:     item.Content,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}

	row, err := r.queries.CreateSubmission(ctx, params)
	if err != nil {
		return mapError(err)
	}

	applyRow(item, row)
	return nil
}

/**
 * FindByID devuelve la postulación con el identificador indicado.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {uuid.UUID} id - Identificador de la postulación.
 * @returns {Submission} Postulación encontrada o nil si no existe.
 */
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*submission.Submission, error) {
	row, err := r.queries.GetSubmissionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("no se pudo consultar la postulación: %w", err)
	}

	item := &submission.Submission{}
	applyRow(item, row)
	return item, nil
}

/**
 * ListByApplicant devuelve la página de postulaciones de un postulante y el
 * total de registros disponibles.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {uuid.UUID} applicantID - Postulante cuyas postulaciones se listan.
 * @param {number} limit - Cantidad máxima de filas a leer.
 * @param {number} offset - Cantidad de filas a omitir.
 * @returns {Submission[]} Postulaciones de la página actual.
 * @returns {number} Total de postulaciones del postulante.
 */
func (r *Repository) ListByApplicant(ctx context.Context, applicantID uuid.UUID, limit int, offset int) ([]*submission.Submission, int64, error) {
	rows, err := r.queries.ListSubmissionsByApplicant(ctx, sqlcgen.ListSubmissionsByApplicantParams{
		ApplicantID: applicantID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("no se pudieron listar las postulaciones: %w", err)
	}

	total, err := r.queries.CountSubmissionsByApplicant(ctx, applicantID)
	if err != nil {
		return nil, 0, fmt.Errorf("no se pudo contar las postulaciones: %w", err)
	}

	items := make([]*submission.Submission, 0, len(rows))
	for _, row := range rows {
		item := &submission.Submission{}
		applyRow(item, row)
		items = append(items, item)
	}

	return items, total, nil
}

/**
 * ListByApplicantCursor devuelve la página siguiente de postulaciones de un
 * postulante usando la clave de orden (fecha de modificación, identificador).
 * Si before es cero devuelve la página más reciente.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {uuid.UUID} applicantID - Postulante cuyas postulaciones se listan.
 * @param {time.Time} before - Fecha del último elemento de la página anterior.
 * @param {uuid.UUID} beforeID - Identificador del último elemento de la página anterior.
 * @param {number} limit - Cantidad máxima de filas a leer.
 * @returns {Submission[]} Postulaciones de la página actual.
 */
func (r *Repository) ListByApplicantCursor(ctx context.Context, applicantID uuid.UUID, before time.Time, beforeID uuid.UUID, limit int) ([]*submission.Submission, error) {
	var rows []sqlcgen.Submission

	if before.IsZero() {
		result, err := r.queries.ListSubmissionsByApplicantCursorFirst(ctx, sqlcgen.ListSubmissionsByApplicantCursorFirstParams{
			ApplicantID: applicantID,
			Limit:       int32(limit),
		})
		if err != nil {
			return nil, fmt.Errorf("no se pudieron listar las postulaciones: %w", err)
		}
		rows = result
	} else {
		result, err := r.queries.ListSubmissionsByApplicantCursor(ctx, sqlcgen.ListSubmissionsByApplicantCursorParams{
			ApplicantID: applicantID,
			UpdatedAt:   before,
			ID:          beforeID,
			Limit:       int32(limit),
		})
		if err != nil {
			return nil, fmt.Errorf("no se pudieron listar las postulaciones: %w", err)
		}
		rows = result
	}

	items := make([]*submission.Submission, 0, len(rows))
	for _, row := range rows {
		item := &submission.Submission{}
		applyRow(item, row)
		items = append(items, item)
	}

	return items, nil
}

/**
 * Update guarda los cambios de una postulación.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {Submission} item - Postulación con los cambios aplicados.
 * @returns {void}
 */
func (r *Repository) Update(ctx context.Context, item *submission.Submission) error {
	params := sqlcgen.UpdateSubmissionParams{
		ID:          item.ID,
		Content:     item.Content,
		Status:      string(item.Status),
		Hash:        optionalString(item.Hash),
		SubmittedAt: timestamptz(item.SubmittedAt),
		UpdatedAt:   item.UpdatedAt,
	}

	row, err := r.queries.UpdateSubmission(ctx, params)
	if err != nil {
		return mapError(err)
	}

	applyRow(item, row)
	return nil
}

/**
 * Delete elimina una postulación.
 *
 * @param {Context} ctx - Contexto de la operación.
 * @param {uuid.UUID} id - Identificador de la postulación.
 * @returns {void}
 * @throws {Error} ErrNotFound si la postulación no existe.
 */
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteSubmission(ctx, id)
	if err != nil {
		return fmt.Errorf("no se pudo eliminar la postulación: %w", err)
	}

	if rowsAffected == 0 {
		return submission.ErrNotFound
	}

	return nil
}

/**
 * Mutate bloquea la fila desde la lectura hasta el commit para serializar
 * guardados y envíos. El callback no recibe infraestructura.
 * @param {function} change - Cambio y validación sobre la versión vigente.
 * @returns {Submission, error} Entidad confirmada o error con rollback.
 */
func (r *Repository) Mutate(ctx context.Context, id uuid.UUID, change func(*submission.Submission) error) (*submission.Submission, error) {
	if r.transactions == nil {
		return nil, errors.New("transacciones no configuradas")
	}
	var item *submission.Submission
	err := r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)
		row, err := queries.GetSubmissionForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return submission.ErrNotFound
		}
		if err != nil {
			return err
		}
		item = &submission.Submission{}
		applyRow(item, row)
		if err := change(item); err != nil {
			return err
		}
		adapter := &Repository{queries: queries}
		return adapter.Update(ctx, item)
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

/**
 * applyRow copia la fila de la base de datos sobre la entidad del dominio.
 *
 * @param {Submission} item - Entidad de destino.
 * @param {sqlcgen.Submission} row - Fila leída de PostgreSQL.
 */
func applyRow(item *submission.Submission, row sqlcgen.Submission) {
	item.ID = row.ID
	item.ApplicantID = row.ApplicantID
	item.Code = row.Code
	item.Status = submission.Status(row.Status)
	item.Content = row.Content
	item.CreatedAt = row.CreatedAt
	item.UpdatedAt = row.UpdatedAt

	if row.Hash != nil {
		item.Hash = *row.Hash
	}

	if row.SubmittedAt.Valid {
		submittedAt := row.SubmittedAt.Time.UTC()
		item.SubmittedAt = &submittedAt
	}
}

/**
 * optionalString convierte un texto vacío en NULL para respetarlo como
 * valor ausente.
 *
 * @param {string} value - Valor a convertir.
 * @returns {string} Puntero al valor o nil si está vacío.
 */
func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

/**
 * timestamptz convierte una fecha opcional al tipo que espera pgx.
 *
 * @param {time.Time} value - Fecha a convertir.
 * @returns {pgtype.Timestamptz} Valor con su indicador de nulidad.
 */
func timestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

/**
 * mapError traduce las violaciones de restricciones de PostgreSQL a errores
 * del dominio de postulaciones.
 *
 * @param {Error} err - Error devuelto por la base de datos.
 * @returns {Error} Error de dominio equivalente o el error original.
 */
func mapError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case "23505":
		return fmt.Errorf("%w: %s", submission.ErrDuplicateCode, pgErr.ConstraintName)
	case "23503":
		return fmt.Errorf("%w: %s", submission.ErrInvalidApplicant, pgErr.ConstraintName)
	default:
		return err
	}
}
