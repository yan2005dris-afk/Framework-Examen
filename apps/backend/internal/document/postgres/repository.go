// Package postgres implementa los metadatos y la cuota transaccional de documentos.
package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	submissionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/postgres"
)

/** Repository comparte los bloqueos del sello y serializa la cuota por postulante. */
type Repository struct {
	executor     db.DBTX
	transactions *db.TxManager
}

/** NewRepository conecta el pool y el gestor transaccional.
 * @returns {Repository} Persistencia de documentos.
 */
func NewRepository(executor db.DBTX, transactions *db.TxManager) *Repository {
	return &Repository{executor: executor, transactions: transactions}
}

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(db.Executor(ctx, r.executor))
}

/** FindSubmission consulta sin bloquear; la mutación repite permisos bajo bloqueo.
 * @returns {Submission, error} Postulación o nil.
 */
func (r *Repository) FindSubmission(ctx context.Context, id uuid.UUID) (*submission.Submission, error) {
	return submissionpostgres.NewRepository(r.queries(ctx)).FindByID(ctx, id)
}

/** WithinSubmission comprueba titularidad y estado sobre filas bloqueadas.
 * @returns {error} Commit de documentos y contenido o rollback de ambos.
 */
func (r *Repository) WithinSubmission(ctx context.Context, actor, id uuid.UUID, change func(context.Context, *submission.Submission, document.Repository) error) error {
	return r.transactions.WithinTx(ctx, func(tx pgx.Tx) error {
		txctx := db.WithTransaction(ctx, tx)
		queries := sqlcgen.New(tx)
		if _, err := queries.LockDocumentApplicant(txctx, actor); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return submission.ErrForbidden
			}
			return err
		}
		row, err := queries.GetSubmissionForUpdate(txctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return submission.ErrNotFound
		}
		if err != nil {
			return err
		}
		if row.ApplicantID != actor {
			return submission.ErrForbidden
		}
		item := &submission.Submission{ID: row.ID, ApplicantID: row.ApplicantID, Code: row.Code, Status: submission.Status(row.Status), Content: row.Content, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
		if row.CompetitionID.Valid {
			id := uuid.UUID(row.CompetitionID.Bytes)
			item.CompetitionID = &id
		}
		if !item.IsEditable() {
			return submission.ErrNotEditable
		}
		if err := change(txctx, item, r); err != nil {
			return err
		}
		submissions := submissionpostgres.NewRepository(queries)
		if err := submissions.Update(txctx, item); err != nil {
			return err
		}
		return submissions.RecordEvent(txctx, item, submission.EventSaved)
	})
}

/** Find obtiene metadatos sin exponerlos antes de comprobar propietario.
 * @returns {Document, error} Documento o nil.
 */
func (r *Repository) Find(ctx context.Context, id uuid.UUID) (*document.Document, error) {
	row, err := r.queries(ctx).GetDocument(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d := fromRow(row)
	return &d, nil
}

/** List usa la transacción del sello cuando existe.
 * @returns {Document[], error} Respaldos de una postulación.
 */
func (r *Repository) List(ctx context.Context, id uuid.UUID) ([]document.Document, error) {
	rows, err := r.queries(ctx).ListDocuments(ctx, id)
	if err != nil {
		return nil, err
	}
	items := make([]document.Document, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromRow(row))
	}
	return items, nil
}

/** TotalByApplicant suma todas las postulaciones bajo el mismo bloqueo de cuota.
 * @returns {int64, error} Bytes vigentes del postulante.
 */
func (r *Repository) TotalByApplicant(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.queries(ctx).TotalDocumentBytes(ctx, id)
}

/** Create inserta un respaldo único por registro.
 * @returns {error} Duplicado o fallo de persistencia.
 */
func (r *Repository) Create(ctx context.Context, d *document.Document) error {
	// También se usa desde SaveSubmission, bajo el bloqueo previo del postulante.
	item, err := r.FindSubmission(ctx, d.SubmissionID)
	if err != nil {
		return err
	}
	if item == nil {
		return submission.ErrNotFound
	}
	total, err := r.TotalByApplicant(ctx, item.ApplicantID)
	if err != nil {
		return err
	}
	if d.SizeBytes > document.MaxApplicantBytes-total {
		return document.ErrQuotaExceeded
	}
	err = r.queries(ctx).CreateDocument(ctx, sqlcgen.CreateDocumentParams{ID: d.ID, SubmissionID: d.SubmissionID, Section: int32(d.Section), RecordID: d.RecordID, Name: d.Name, Mime: d.MIME, SizeBytes: d.SizeBytes, Sha256: d.SHA256, StorageKey: d.StorageKey, CreatedAt: d.CreatedAt})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return document.ErrDuplicate
	}
	return err
}

/** Delete activa la cola transaccional de borrado mediante el trigger.
 * @returns {error} Documento inexistente o fallo SQL.
 */
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.queries(ctx).DeleteDocument(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return document.ErrNotFound
	}
	return nil
}

/** PendingDeletions lee un lote acotado de bajas confirmadas.
 * @returns {string[], error} Claves por eliminar.
 */
func (r *Repository) PendingDeletions(ctx context.Context) ([]string, error) {
	return r.queries(ctx).PendingDocumentDeletions(ctx)
}

/** CompleteDeletion confirma un borrado idempotente del servidor de objetos.
 * @returns {error} Fallo SQL; se reintentará en la siguiente pasada.
 */
func (r *Repository) CompleteDeletion(ctx context.Context, key string) error {
	return r.queries(ctx).CompleteDocumentDeletion(ctx, key)
}

func fromRow(row sqlcgen.Document) document.Document {
	return document.Document{ID: row.ID, SubmissionID: row.SubmissionID, Section: int(row.Section), RecordID: row.RecordID, Name: row.Name, MIME: row.Mime, SizeBytes: row.SizeBytes, SHA256: row.Sha256, StorageKey: row.StorageKey, CreatedAt: row.CreatedAt}
}
