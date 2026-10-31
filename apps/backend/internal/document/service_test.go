package document

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryRepo struct {
	Repository
	mu         sync.Mutex
	item       *submission.Submission
	docs       map[uuid.UUID]Document
	queued     []string
	otherBytes int64
	fail       bool
}

func (r *memoryRepo) FindSubmission(context.Context, uuid.UUID) (*submission.Submission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *r.item
	return &copy, nil
}
func (r *memoryRepo) WithinSubmission(ctx context.Context, actor, id uuid.UUID, change func(context.Context, *submission.Submission, Repository) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.item.ID != id {
		return submission.ErrNotFound
	}
	if r.item.ApplicantID != actor {
		return submission.ErrForbidden
	}
	if !r.item.IsEditable() {
		return submission.ErrNotEditable
	}
	copy := *r.item
	copy.Content = append(json.RawMessage(nil), r.item.Content...)
	docs := make(map[uuid.UUID]Document)
	for k, v := range r.docs {
		docs[k] = v
	}
	tx := &memoryRepo{item: &copy, docs: docs, queued: append([]string(nil), r.queued...), otherBytes: r.otherBytes}
	if err := change(ctx, &copy, tx); err != nil {
		return err
	}
	if r.fail {
		return errors.New("commit fallido")
	}
	r.item = &copy
	r.docs = tx.docs
	r.queued = tx.queued
	return nil
}
func (r *memoryRepo) Find(_ context.Context, id uuid.UUID) (*Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.docs[id]
	if !ok {
		return nil, nil
	}
	return &d, nil
}
func (r *memoryRepo) List(_ context.Context, id uuid.UUID) ([]Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := []Document{}
	for _, d := range r.docs {
		if d.SubmissionID == id {
			items = append(items, d)
		}
	}
	return items, nil
}
func (r *memoryRepo) TotalByApplicant(context.Context, uuid.UUID) (int64, error) {
	total := r.otherBytes
	for _, d := range r.docs {
		total += d.SizeBytes
	}
	return total, nil
}
func (r *memoryRepo) Create(_ context.Context, d *Document) error {
	for _, old := range r.docs {
		if old.SubmissionID == d.SubmissionID && old.Section == d.Section && old.RecordID == d.RecordID {
			return ErrDuplicate
		}
	}
	r.docs[d.ID] = *d
	return nil
}
func (r *memoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	d, ok := r.docs[id]
	if !ok {
		return ErrNotFound
	}
	delete(r.docs, id)
	r.queued = append(r.queued, d.StorageKey)
	return nil
}
func (r *memoryRepo) PendingDeletions(context.Context) ([]string, error) {
	return append([]string(nil), r.queued...), nil
}
func (r *memoryRepo) CompleteDeletion(_ context.Context, key string) error {
	for i, k := range r.queued {
		if k == key {
			r.queued = append(r.queued[:i], r.queued[i+1:]...)
			break
		}
	}
	return nil
}

type memoryStorage struct {
	mu                   sync.Mutex
	objects              map[string][]byte
	failSave, failDelete bool
}

func (s *memoryStorage) Save(_ context.Context, key string, content io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failSave {
		return errors.New("disco lleno")
	}
	data, err := io.ReadAll(content)
	s.objects[key] = data
	return err
}
func (s *memoryStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failDelete {
		return errors.New("almacenamiento no disponible")
	}
	delete(s.objects, key)
	return nil
}
func (s *memoryStorage) PresignGet(context.Context, string, string, time.Duration) (string, error) {
	return "https://objects.example.test/signed", nil
}

func fixture(t *testing.T) (*Service, *memoryRepo, *memoryStorage, UploadInput) {
	t.Helper()
	item, err := submission.NewSubmission(uuid.New(), uuid.New(), "PRUEBA", json.RawMessage(`{"seccion4":{"cuentaConRespaldoOrganizacion":"Sí"},"seccion5":{"titulos":[{"id":"titulo-1"},{"id":"titulo-2"}]}}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	repo := &memoryRepo{item: item, docs: make(map[uuid.UUID]Document)}
	store := &memoryStorage{objects: make(map[string][]byte)}
	return NewService(repo, store), repo, store, UploadInput{ApplicantID: item.ApplicantID, SubmissionID: item.ID, Section: 5, RecordID: "titulo-1", Name: "titulo.pdf", MIME: "application/pdf", Content: strings.NewReader("%PDF-1.7\n%%EOF")}
}

func TestUploadRejectsForgedMIMEAndLimits(t *testing.T) {
	for _, tc := range []struct {
		name    string
		section int
		mime    string
		data    []byte
		want    error
	}{
		{"mime", 5, "text/plain", []byte("%PDF-1.7"), ErrUnsupportedType},
		{"firma", 5, "application/pdf", []byte("<html>no PDF</html>"), ErrUnsupportedType},
		{"vacio", 5, "application/pdf", nil, ErrUnsupportedType},
		{"dosMiB", 5, "application/pdf", append([]byte("%PDF-1.7"), make([]byte, MaxFileBytes)...), ErrFileTooLarge},
		{"cuatroMiB", 4, "application/pdf", append([]byte("%PDF-1.7"), make([]byte, MaxOrganizationBytes)...), ErrFileTooLarge},
		{"seccion", 9, "application/pdf", []byte("%PDF-1.7"), ErrInvalidSection},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, repo, store, in := fixture(t)
			in.Section = tc.section
			in.MIME = tc.mime
			in.Content = bytes.NewReader(tc.data)
			_, err := service.Upload(context.Background(), in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("esperado %v, obtenido %v", tc.want, err)
			}
			if len(repo.docs) != 0 || len(store.objects) != 0 {
				t.Fatal("quedaron archivos rechazados")
			}
		})
	}
}

func TestUploadBoundaryHashAndReferences(t *testing.T) {
	for _, section := range []int{4, 5, 6, 7, 8} {
		t.Run(string(rune('0'+section)), func(t *testing.T) {
			service, repo, store, in := fixture(t)
			in.Section = section
			var raw string
			switch section {
			case 4:
				in.RecordID = ""
			case 6:
				raw = `{"seccion6":{"experiencias":[{"id":"titulo-1"}]}}`
			case 7:
				raw = `{"seccion7":{"publicaciones":[{"id":"titulo-1"}]}}`
			case 8:
				raw = `{"seccion8":{"capacitaciones":[{"id":"titulo-1"}]}}`
			}
			if raw != "" {
				repo.item.Content = json.RawMessage(raw)
			}
			limit, _ := LimitForSection(section)
			data := make([]byte, limit)
			copy(data, "%PDF-1.7\n")
			in.Content = bytes.NewReader(data)
			d, err := service.Upload(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(data)
			if d.SHA256 != hex.EncodeToString(sum[:]) || d.SizeBytes != limit || d.ID.Version() != 7 || len(store.objects) != 1 {
				t.Fatal("metadatos incorrectos")
			}
			parts := strings.Split(d.StorageKey, "/")
			random, err := uuid.Parse(strings.TrimSuffix(parts[2], ".pdf"))
			if err != nil || random.Version() != 4 || parts[0] != in.SubmissionID.String() {
				t.Fatal("clave predecible")
			}
			f, _ := submission.DecodeForm(repo.item.Content)
			slot, err := referenceSlot(f, section, in.RecordID)
			if err != nil || (*slot).ID != d.ID.String() || (*slot).SHA256 != d.SHA256 {
				t.Fatal("referencia sin confirmar")
			}
		})
	}
}

func TestQuotaAggregatesFilesAndOtherSubmissions(t *testing.T) {
	service, repo, store, in := fixture(t)
	ctx := context.Background()
	repo.otherBytes = MaxApplicantBytes - 2*MaxFileBytes
	data := make([]byte, MaxFileBytes)
	copy(data, "%PDF-1.7")
	in.Content = bytes.NewReader(data)
	if _, err := service.Upload(ctx, in); err != nil {
		t.Fatal(err)
	}
	in.RecordID = "titulo-2"
	in.Content = bytes.NewReader(data)
	if _, err := service.Upload(ctx, in); err != nil {
		t.Fatal("no admite cuota exacta", err)
	}
	in.Section = 4
	in.RecordID = ""
	in.Content = strings.NewReader("%PDF-1.7")
	if _, err := service.Upload(ctx, in); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatal(err)
	}
	if len(repo.docs) != 2 || len(store.objects) != 2 {
		t.Fatal("la carga rechazada consumió cuota")
	}
}

func TestConcurrentQuotaAndSealedDocuments(t *testing.T) {
	service, repo, _, in := fixture(t)
	repo.otherBytes = MaxApplicantBytes - MaxFileBytes
	data := make([]byte, MaxFileBytes)
	copy(data, "%PDF-1.7")
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, record := range []string{"titulo-1", "titulo-2"} {
		wg.Go(func() {
			input := in
			input.RecordID = record
			input.Content = bytes.NewReader(data)
			_, err := service.Upload(context.Background(), input)
			results <- err
		})
	}
	wg.Wait()
	close(results)
	success, quota := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrQuotaExceeded) {
			quota++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || quota != 1 {
		t.Fatalf("éxitos %d, cuotas %d", success, quota)
	}
	repo.item.Status = submission.StatusSubmitted
	for id := range repo.docs {
		if err := service.Delete(context.Background(), in.ApplicantID, id); !errors.Is(err, submission.ErrNotEditable) {
			t.Fatal(err)
		}
		if _, err := service.Download(context.Background(), in.ApplicantID, id); err != nil {
			t.Fatal(err)
		}
	}
	in.Content = strings.NewReader("%PDF-1.7")
	if _, err := service.Upload(context.Background(), in); !errors.Is(err, submission.ErrNotEditable) {
		t.Fatal(err)
	}
}

func TestRollbackDuplicateOwnershipAndDeletionRetry(t *testing.T) {
	service, repo, store, in := fixture(t)
	ctx := context.Background()
	in.ApplicantID = uuid.New()
	if _, err := service.Upload(ctx, in); !errors.Is(err, submission.ErrForbidden) {
		t.Fatal(err)
	}
	in.ApplicantID = repo.item.ApplicantID
	in.RecordID = "no-existe"
	if _, err := service.Upload(ctx, in); !errors.Is(err, ErrInvalidRecord) {
		t.Fatal(err)
	}
	in.RecordID = "titulo-1"
	repo.fail = true
	in.Content = strings.NewReader("%PDF-1.7")
	if _, err := service.Upload(ctx, in); err == nil || len(store.objects) != 0 || len(repo.docs) != 0 {
		t.Fatal("rollback sin compensar")
	}
	repo.fail = false
	in.Content = strings.NewReader("%PDF-1.7")
	d, err := service.Upload(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	in.Content = strings.NewReader("%PDF-1.7")
	if _, err := service.Upload(ctx, in); !errors.Is(err, ErrDuplicate) || len(store.objects) != 1 {
		t.Fatal(err)
	}
	if _, err := service.Download(ctx, uuid.New(), d.ID); !errors.Is(err, submission.ErrForbidden) {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, in.ApplicantID, d.ID); err != nil {
		t.Fatal(err)
	}
	if len(repo.queued) != 1 || len(repo.docs) != 0 {
		t.Fatal("no se encoló la baja")
	}
	store.failDelete = true
	if err := service.Cleanup(ctx); err == nil || len(repo.queued) != 1 {
		t.Fatal("se perdió el reintento")
	}
	store.failDelete = false
	if err := service.Cleanup(ctx); err != nil || len(repo.queued) != 0 || len(store.objects) != 0 {
		t.Fatal("baja no completada", err)
	}
}

func TestSealRejectsFakeReferencesAndNormalizesMetadata(t *testing.T) {
	service, repo, _, in := fixture(t)
	ctx := context.Background()
	d, err := service.Upload(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	f := &submission.Form{Education: &submission.EducationSection{Degrees: []submission.Degree{{ID: in.RecordID, Document: &submission.DocumentReference{Name: "inventado.pdf"}}}}}
	if err := service.VerifyReferences(ctx, in.SubmissionID, f); !errors.Is(err, submission.ErrBusinessRule) {
		t.Fatal("aceptó referencia falsa", err)
	}
	f.Education.Degrees[0].Document.ID = d.ID.String()
	if err := service.VerifyReferences(ctx, in.SubmissionID, f); err != nil {
		t.Fatal(err)
	}
	if f.Education.Degrees[0].Document.Name != d.Name || f.Education.Degrees[0].Document.SHA256 != d.SHA256 {
		t.Fatal("conservó metadatos del cliente")
	}
	f.Education.Degrees[0].ID = "otro"
	if err := service.VerifyReferences(ctx, in.SubmissionID, f); !errors.Is(err, submission.ErrBusinessRule) {
		t.Fatal("aceptó documento de otro registro")
	}
	_ = repo
}
