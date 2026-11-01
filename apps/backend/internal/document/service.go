package document

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/audit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

/** UploadInput contiene el flujo multipart y el registro al que corresponde. */
type UploadInput struct {
	ApplicantID  uuid.UUID
	SubmissionID uuid.UUID
	Section      int
	RecordID     string
	Name         string
	MIME         string
	Content      io.Reader
}

/** Service aplica límites, titularidad y consistencia de archivos y metadatos. */
type Service struct {
	repo        Repository
	storage     Storage
	now         func() time.Time
	auditor     audit.Recorder
	eligibility submission.EligibilityAssessor
}

/** NewService inyecta los puertos de documentos.
 * @returns {Service} Casos de uso con reloj del servidor.
 */
func NewService(repo Repository, storage Storage) *Service {
	return &Service{repo: repo, storage: storage, now: time.Now}
}

func (s *Service) WithEligibility(a submission.EligibilityAssessor) *Service {
	s.eligibility = a
	return s
}

/** Authorize comprueba la titularidad antes de leer o escribir un archivo.
 * @returns {Submission, error} Postulación propia, editable cuando se solicita.
 */
func (s *Service) Authorize(ctx context.Context, actor, id uuid.UUID, editable bool) (*submission.Submission, error) {
	item, err := s.repo.FindSubmission(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, submission.ErrNotFound
	}
	if actor == uuid.Nil || item.ApplicantID != actor {
		return nil, submission.ErrForbidden
	}
	if editable && item.Status == submission.StatusClosed {
		return nil, submission.ErrEligibilityClosed
	}
	if editable && !item.IsEditable() {
		return nil, submission.ErrNotEditable
	}
	return item, nil
}

/** Upload verifica el contenido real y confirma metadatos y referencia bajo bloqueo.
 * @param {UploadInput} input - Un PDF para un registro existente del EF-12.
 * @returns {Document, error} Metadatos confirmados o rechazo sin consumir cuota.
 */
func (s *Service) Upload(ctx context.Context, input UploadInput) (*Document, error) {
	limit, err := LimitForSection(input.Section)
	if err != nil {
		return nil, err
	}
	if !utf8.ValidString(input.RecordID) || utf8.RuneCountInString(input.RecordID) > 255 || strings.IndexFunc(input.RecordID, unicode.IsControl) >= 0 {
		return nil, ErrInvalidRecord
	}
	if _, err := s.Authorize(ctx, input.ApplicantID, input.SubmissionID, true); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 255 || strings.ContainsAny(name, "/\\:") || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return nil, ErrInvalidName
	}
	media, _, err := mime.ParseMediaType(input.MIME)
	if err != nil || media != "application/pdf" {
		return nil, ErrUnsupportedType
	}
	if input.Content == nil {
		return nil, ErrUnsupportedType
	}
	data, err := io.ReadAll(io.LimitReader(input.Content, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrFileTooLarge
	}
	if http.DetectContentType(data) != "application/pdf" || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, ErrUnsupportedType
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	// La clave usa UUIDv4: no revela la hora de carga y aporta 122 bits aleatorios.
	objectID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	d := &Document{ID: id, SubmissionID: input.SubmissionID, Section: input.Section, RecordID: input.RecordID, Name: name, MIME: "application/pdf", SizeBytes: int64(len(data)), SHA256: hex.EncodeToString(sum[:]), CreatedAt: s.now().UTC(), StorageKey: fmt.Sprintf("%s/%d/%s.pdf", input.SubmissionID, input.Section, objectID)}
	if err := s.storage.Save(ctx, d.StorageKey, bytes.NewReader(data)); err != nil {
		s.removeObject(d.StorageKey)
		return nil, ErrStorageUnavailable
	}
	var closure *submission.EligibilityAssessment
	err = s.repo.WithinSubmission(ctx, input.ApplicantID, input.SubmissionID, func(txctx context.Context, item *submission.Submission, repo Repository) error {
		checked, closed, err := submission.PrepareEligibility(txctx, s.eligibility, item, item.Content, s.now())
		if err != nil {
			return err
		}
		if closed {
			closure = item.Eligibility
			return nil
		}
		item.Content = checked
		total, err := repo.TotalByApplicant(txctx, input.ApplicantID)
		if err != nil {
			return err
		}
		if total > MaxApplicantBytes-d.SizeBytes {
			return ErrQuotaExceeded
		}
		form, err := submission.DecodeForm(item.Content)
		if err != nil {
			return err
		}
		slot, err := referenceSlot(form, d.Section, d.RecordID)
		if err != nil {
			return err
		}
		if err := repo.Create(txctx, d); err != nil {
			return err
		}
		*slot = documentReference(*d)
		raw, err := json.Marshal(form)
		if err != nil {
			return err
		}
		return item.UpdateContent(raw, s.now())
	})
	if err == nil && closure != nil {
		s.removeObject(d.StorageKey)
		return nil, &submission.ClosureError{Assessment: closure}
	}
	if err != nil {
		// Un fallo de COMMIT puede ser ambiguo. No borrar un objeto ya referenciado.
		checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		persisted, checkErr := s.repo.Find(checkCtx, d.ID)
		if checkErr == nil && persisted != nil {
			return persisted, nil
		}
		if checkErr == nil {
			s.removeObject(d.StorageKey)
		}
		return nil, err
	}
	return d, nil
}

/** List devuelve solamente metadatos de la postulación propia.
 * @returns {Document[], error} Respaldos vigentes.
 */
func (s *Service) List(ctx context.Context, actor, id uuid.UUID) ([]Document, error) {
	if _, err := s.Authorize(ctx, actor, id, false); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, id)
}

/** Download emite una URL de cinco minutos tras comprobar propietario.
 * @returns {DownloadResponse, error} Enlace directo al servidor de objetos.
 */
func (s *Service) Download(ctx context.Context, actor, id uuid.UUID) (DownloadResponse, error) {
	d, err := s.repo.Find(ctx, id)
	if err != nil {
		return DownloadResponse{}, err
	}
	if d == nil {
		return DownloadResponse{}, ErrNotFound
	}
	if _, err := s.Authorize(ctx, actor, d.SubmissionID, false); err != nil {
		return DownloadResponse{}, err
	}
	url, err := s.storage.PresignGet(ctx, d.StorageKey, d.Name, DownloadTTL)
	return DownloadResponse{URL: url, ExpiresAt: s.now().UTC().Add(DownloadTTL)}, err
}

/** Delete retira referencia y metadatos; la transacción encola el borrado del objeto.
 * @returns {error} Conflicto si el envío ganó el bloqueo.
 */
func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	d, err := s.repo.Find(ctx, id)
	if err != nil {
		return err
	}
	if d == nil {
		return ErrNotFound
	}
	var closure *submission.EligibilityAssessment
	err = s.repo.WithinSubmission(ctx, actor, d.SubmissionID, func(txctx context.Context, item *submission.Submission, repo Repository) error {
		checked, closed, err := submission.PrepareEligibility(txctx, s.eligibility, item, item.Content, s.now())
		if err != nil {
			return err
		}
		if closed {
			closure = item.Eligibility
			return nil
		}
		item.Content = checked
		current, err := repo.Find(txctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrNotFound
		}
		form, err := submission.DecodeForm(item.Content)
		if err != nil {
			return err
		}
		slot, slotErr := referenceSlot(form, d.Section, d.RecordID)
		if slotErr == nil && *slot != nil && (*slot).ID == id.String() {
			*slot = nil
		}
		raw, err := json.Marshal(form)
		if err != nil {
			return err
		}
		if err := item.UpdateContent(raw, s.now()); err != nil {
			return err
		}
		return repo.Delete(txctx, id)
	})
	if err == nil && closure != nil {
		return &submission.ClosureError{Assessment: closure}
	}
	return err
}

/** Cleanup reintenta bajas persistidas, incluidas las cascadas al borrar borradores.
 * @returns {error} Primer fallo; las operaciones pendientes sobreviven al reinicio.
 */
func (s *Service) Cleanup(ctx context.Context) error {
	keys, err := s.repo.PendingDeletions(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := s.storage.Delete(ctx, key); err != nil {
			return err
		}
		if err := s.repo.CompleteDeletion(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) removeObject(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.storage.Delete(ctx, key); err != nil && !errors.Is(err, context.Canceled) {
		retryCtx, retryCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer retryCancel()
		if queueErr := s.repo.QueueDeletion(retryCtx, key); queueErr != nil {
			slog.Error("no se pudo registrar la compensación de una carga fallida")
		}
	}
}
