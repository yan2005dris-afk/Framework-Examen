package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/db"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	submissionpostgres "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/postgres"
)

// Usa PostgreSQL real con DOCUMENT_TEST_DATABASE_URL y el esquema aislado de T4.
func TestSaveSubmissionCommitsAllTablesOrRollsBackOnDocumentFailure(t *testing.T) {
	for _, failDocument := range []bool{false, true} {
		t.Run(fmt.Sprintf("fallo_documento_%t", failDocument), func(t *testing.T) {
			documents, pool, existing := postgresFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			submissions := submissionpostgres.NewRepository(sqlcgen.New(pool), db.NewTxManager(pool))
			service := submission.NewService(submissions)
			var submissionID, firstDocumentID uuid.UUID
			code := "EF12-T5-AGGREGATE"
			saved, err := service.SaveSubmission(ctx, submission.CreateInput{
				ApplicantID: existing.ApplicantID,
				Code:        code,
				Content:     json.RawMessage(`{"seccion5":{"titulos":[{"id":"titulo-1"},{"id":"titulo-2"}]}}`),
				SaveRelated: func(txctx context.Context, current *submission.Submission) error {
					submissionID = current.ID
					form, err := submission.DecodeForm(current.Content)
					if err != nil {
						return err
					}
					for i := range 2 {
						id, err := uuid.NewV7()
						if err != nil {
							return err
						}
						if i == 0 {
							firstDocumentID = id
						}
						mimeType := "application/pdf"
						if failDocument && i == 1 {
							mimeType = "image/png"
						}
						item := &document.Document{
							ID: id, SubmissionID: current.ID, Section: 5,
							RecordID: fmt.Sprintf("titulo-%d", i+1), Name: "respaldo.pdf",
							MIME: mimeType, SizeBytes: 100, SHA256: strings.Repeat("0", 64),
							StorageKey: fmt.Sprintf("%s/5/%s.pdf", current.ID, id), CreatedAt: time.Now().UTC(),
						}
						// El primer INSERT debe ejecutarse antes del fallo real del segundo.
						if err := documents.Create(txctx, item); err != nil {
							return err
						}
						form.Education.Degrees[i].Document = &submission.DocumentReference{ID: id.String(), Name: item.Name, Size: "100", SHA256: item.SHA256}
					}
					current.Content, err = json.Marshal(form)
					return err
				},
			})
			if failDocument {
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
					t.Fatalf("se esperaba CHECK del segundo documento, obtenido %v", err)
				}
				if saved != nil {
					t.Fatal("se devolvió una entidad sin commit")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if submissionID == uuid.Nil || firstDocumentID == uuid.Nil {
				t.Fatal("no se alcanzó la escritura multi-tabla")
			}
			var headers, attachments, events, pending int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM submissions WHERE code = $1`, code).Scan(&headers); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM documents WHERE submission_id = $1`, submissionID).Scan(&attachments); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM submission_events WHERE submission_id = $1`, submissionID).Scan(&events); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM document_deletions WHERE storage_key LIKE $1`, submissionID.String()+"/%").Scan(&pending); err != nil {
				t.Fatal(err)
			}
			if failDocument {
				if headers != 0 || attachments != 0 || events != 0 || pending != 0 {
					t.Fatalf("rollback dejó cabeceras=%d documentos=%d eventos=%d bajas=%d", headers, attachments, events, pending)
				}
				return
			}
			if headers != 1 || attachments != 2 || events != 3 || pending != 0 {
				t.Fatalf("commit incompleto: cabeceras=%d documentos=%d eventos=%d bajas=%d", headers, attachments, events, pending)
			}
			persisted, err := submissions.FindByID(ctx, submissionID)
			if err != nil {
				t.Fatal(err)
			}
			form, err := submission.DecodeForm(persisted.Content)
			if err != nil {
				t.Fatal(err)
			}
			if form.Education.Degrees[0].Document.ID != firstDocumentID.String() {
				t.Fatal("no se confirmaron las secciones y sus referencias")
			}
		})
	}
}
