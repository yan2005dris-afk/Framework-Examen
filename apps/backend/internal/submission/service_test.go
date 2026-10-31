package submission

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// El doble publica copias y confirma únicamente si el callback termina bien.
type memoryRepository struct {
	Repository
	mu   sync.Mutex
	item *Submission
	fail bool
}

func copySubmission(item *Submission) *Submission {
	if item == nil {
		return nil
	}
	copy := *item
	copy.Content = append(json.RawMessage(nil), item.Content...)
	return &copy
}
func (r *memoryRepository) FindByID(_ context.Context, id uuid.UUID) (*Submission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.item == nil || r.item.ID != id {
		return nil, nil
	}
	return copySubmission(r.item), nil
}
func (r *memoryRepository) Create(_ context.Context, item *Submission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.item = copySubmission(item)
	return nil
}
func (r *memoryRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.item == nil || r.item.ID != id {
		return ErrNotFound
	}
	if !r.item.IsEditable() {
		return ErrNotEditable
	}
	r.item = nil
	return nil
}
func (r *memoryRepository) Mutate(ctx context.Context, id uuid.UUID, change func(context.Context, *Submission) error) (*Submission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.item == nil || r.item.ID != id {
		return nil, ErrNotFound
	}
	item := copySubmission(r.item)
	if err := change(ctx, item); err != nil {
		return nil, err
	}
	if r.fail {
		return nil, fmt.Errorf("fallo de persistencia de prueba")
	}
	r.item = copySubmission(item)
	return item, nil
}

type recordingNotifier struct {
	calls int
	fail  bool
	repo  *memoryRepository
}

type verifyingDocuments struct {
	failure error
	calls   int
}

func (v *verifyingDocuments) VerifyReferences(_ context.Context, _ uuid.UUID, form *Form) error {
	v.calls++
	if v.failure != nil {
		return v.failure
	}
	form.Position.Document = &DocumentReference{ID: "documento-verificado", Name: "respaldo.pdf", Size: "100", SHA256: "hash-del-archivo"}
	return nil
}

func TestSubmitVerifiesDocumentsBeforeHashAndCommit(t *testing.T) {
	s, repo, notifier := serviceFixture(t, true)
	verifier := &verifyingDocuments{failure: ErrBusinessRule}
	s.WithDocuments(verifier)
	if _, err := s.Submit(context.Background(), repo.item.ID, nil); !errors.Is(err, ErrBusinessRule) {
		t.Fatal(err)
	}
	if repo.item.Status == StatusSubmitted || notifier.calls != 0 {
		t.Fatal("se selló sin respaldo")
	}
	verifier.failure = nil
	sealed, err := s.Submit(context.Background(), repo.item.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	f, err := DecodeForm(sealed.Content)
	if err != nil {
		t.Fatal(err)
	}
	if f.Position.Document.SHA256 != "hash-del-archivo" || verifier.calls != 2 {
		t.Fatal("el hash no incorporó los metadatos verificados")
	}
	canonical, err := CanonicalContent(sealed.Content)
	if err != nil || HashContent(canonical) != sealed.Hash {
		t.Fatal("sello no reproducible", err)
	}
}

func (n *recordingNotifier) Confirm(ctx context.Context, item *Submission) error {
	n.calls++
	stored, _ := n.repo.FindByID(ctx, item.ID)
	if stored.Hash != item.Hash || stored.Status != StatusSubmitted {
		return fmt.Errorf("aviso antes del commit")
	}
	if n.fail {
		return fmt.Errorf("fallo SMTP de prueba")
	}
	return nil
}
func serviceFixture(t *testing.T, full bool) (*Service, *memoryRepository, *recordingNotifier) {
	t.Helper()
	raw := json.RawMessage(`{}`)
	if full {
		raw, _ = json.Marshal(completeForm())
	}
	item, err := NewSubmission(uuid.New(), uuid.New(), "EF12-2026-PRUEBA", raw, referenceTime)
	if err != nil {
		t.Fatal(err)
	}
	repo := &memoryRepository{item: item}
	notifier := &recordingNotifier{repo: repo}
	service := NewService(repo, notifier)
	service.now = func() time.Time { return referenceTime }
	return service, repo, notifier
}

func TestSaveSectionMergesAndCalculatesProgress(t *testing.T) {
	s, repo, _ := serviceFixture(t, false)
	ctx := context.Background()
	id := repo.item.ID
	item, err := s.SaveSection(ctx, id, 1, json.RawMessage(`{"nombres":"Persona"}`))
	if err != nil {
		t.Fatal(err)
	}
	if p := FormProgress(item.Content, referenceTime); p.Completed != 0 {
		t.Fatalf("progreso: %+v", p)
	}
	item, err = s.SaveSection(ctx, id, 1, json.RawMessage(`{"apellidos":"Prueba","cedula":"1710034008","genero":"Femenino","fechaNacimiento":"1990-01-01","nacionalidad":"Ecuatoriana","estadoCivil":"Soltero/a"}`))
	if err != nil {
		t.Fatal(err)
	}
	form, _ := DecodeForm(item.Content)
	if form.Personal.FirstName != "Persona" {
		t.Fatal("se perdió el campo guardado")
	}
	if p := FormProgress(item.Content, referenceTime); p.Completed != 1 || p.Percentage != 11 || !p.Sections[0].Complete {
		t.Fatalf("progreso: %+v", p)
	}
	item, err = s.SaveSection(ctx, id, 3, json.RawMessage(`{"tieneFamiliaresIessBiess":"No","familiares":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if p := FormProgress(item.Content, referenceTime); p.Completed != 2 {
		t.Fatalf("se perdió otra sección: %+v", p)
	}
	before := string(item.Content)
	if _, err := s.SaveSection(ctx, id, 1, json.RawMessage(`{"nacionalidad":"Extranjera"}`)); !errors.Is(err, ErrBusinessRule) {
		t.Fatalf("regla esperada: %v", err)
	}
	if string(repo.item.Content) != before {
		t.Fatal("se persistió un cambio inválido")
	}
}

func TestSubmitSealsConsentAndLocksEveryMutation(t *testing.T) {
	s, repo, n := serviceFixture(t, true)
	id := repo.item.ID
	ctx := context.Background()
	item, err := s.Submit(ctx, id, nil)
	if err != nil {
		t.Fatal(err)
	}
	canonical, _ := CanonicalContent(item.Content)
	if item.Status != StatusSubmitted || item.Hash != HashContent(canonical) || item.SubmittedAt == nil || !item.SubmittedAt.Equal(referenceTime) {
		t.Fatal("sello incompleto")
	}
	form, _ := DecodeForm(item.Content)
	if len(form.Consent.Statements) != 5 || form.Consent.Version == "" || form.Consent.AcceptedAt == "" {
		t.Fatal("no se conservaron los cinco literales y su fecha")
	}
	if form.Experience.Experiences[0].Duration.Text != "1 año, 2 meses, 14 días" {
		t.Fatal("duración no calculada")
	}
	if n.calls != 1 {
		t.Fatal("correo faltante")
	}
	for name, action := range map[string]func() error{
		"editar":   func() error { _, err := s.Update(ctx, id, json.RawMessage(`{}`)); return err },
		"sección":  func() error { _, err := s.SaveSection(ctx, id, 1, json.RawMessage(`{}`)); return err },
		"reenviar": func() error { _, err := s.Submit(ctx, id, nil); return err },
		"eliminar": func() error { return s.Delete(ctx, id) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := action(); !errors.Is(err, ErrNotEditable) {
				t.Fatalf("bloqueo esperado: %v", err)
			}
		})
	}
	if n.calls != 1 {
		t.Fatal("correo duplicado")
	}
}

func TestSubmitRejectsIncompleteAndDoesNotNotify(t *testing.T) {
	s, repo, n := serviceFixture(t, false)
	if _, err := s.Submit(context.Background(), repo.item.ID, nil); !errors.Is(err, ErrBusinessRule) {
		t.Fatal(err)
	}
	if repo.item.Status != StatusDraft || n.calls != 0 {
		t.Fatal("se selló o notificó un formulario incompleto")
	}
}

func TestSubmitPersistenceAndMailFailures(t *testing.T) {
	t.Run("rollback", func(t *testing.T) {
		s, repo, n := serviceFixture(t, true)
		repo.fail = true
		if _, err := s.Submit(context.Background(), repo.item.ID, nil); err == nil {
			t.Fatal("se esperaba fallo")
		}
		if repo.item.Status != StatusDraft || repo.item.Hash != "" || n.calls != 0 {
			t.Fatal("efectos antes del commit")
		}
	})
	t.Run("SMTP posterior", func(t *testing.T) {
		s, repo, n := serviceFixture(t, true)
		n.fail = true
		item, err := s.Submit(context.Background(), repo.item.ID, nil)
		if err != nil || item.Status != StatusSubmitted || n.calls != 1 {
			t.Fatalf("el fallo SMTP no debe deshacer el sello: %v", err)
		}
	})
}

func TestSubmitCanReceiveFinalConsent(t *testing.T) {
	s, repo, _ := serviceFixture(t, true)
	form := completeForm()
	consent := form.Consent
	form.Consent = nil
	repo.item.Content, _ = json.Marshal(form)
	if _, err := s.Submit(context.Background(), repo.item.ID, nil); !errors.Is(err, ErrBusinessRule) {
		t.Fatal("se aceptó sin consentimiento")
	}
	if _, err := s.Submit(context.Background(), repo.item.ID, consent); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentSectionSavesDoNotLoseOtherSections(t *testing.T) {
	s, repo, _ := serviceFixture(t, false)
	id := repo.item.ID
	var wg sync.WaitGroup
	for n, raw := range map[int]string{1: `{"nombres":"Persona"}`, 3: `{"tieneFamiliaresIessBiess":"No"}`, 7: `{"cuentaConPublicaciones":"No"}`, 8: `{"capacitaciones":[]}`} {
		wg.Go(func() {
			if _, err := s.SaveSection(context.Background(), id, n, json.RawMessage(raw)); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	f, _ := DecodeForm(repo.item.Content)
	if f.Personal == nil || f.Additional == nil || f.Publications == nil || f.Training == nil {
		t.Fatal("se perdió un guardado concurrente")
	}
}

func TestConcurrentSubmitsOnlySealAndNotifyOnce(t *testing.T) {
	s, repo, n := serviceFixture(t, true)
	id := repo.item.ID
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, err := s.Submit(context.Background(), id, nil); results <- err })
	}
	wg.Wait()
	close(results)
	ok, conflicts := 0, 0
	for err := range results {
		if err == nil {
			ok++
		} else if errors.Is(err, ErrNotEditable) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflicts != 1 || n.calls != 1 {
		t.Fatalf("éxitos=%d conflictos=%d correos=%d", ok, conflicts, n.calls)
	}
}

func TestSaveSectionRejectsInvalidStructureAndNumber(t *testing.T) {
	s, repo, _ := serviceFixture(t, false)
	ctx := context.Background()
	for _, n := range []int{0, 10} {
		if _, err := s.SaveSection(ctx, repo.item.ID, n, json.RawMessage(`{}`)); !errors.Is(err, ErrInvalidSection) {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{`null`, `[]`, `{"nombreMalEscrito":"Prueba"}`, `{"campoDesconocido":null}`} {
		if _, err := s.SaveSection(ctx, repo.item.ID, 1, json.RawMessage(raw)); !errors.Is(err, ErrInvalidContent) {
			t.Fatalf("estructura aceptada: %s: %v", raw, err)
		}
	}
}

func TestSectionPatchMergesNestedDetailsAndCanClearThem(t *testing.T) {
	s, repo, _ := serviceFixture(t, true)
	id := repo.item.ID
	ctx := context.Background()
	for _, raw := range []string{`{"q7":"Sí","q7Detalles":{"institucion":"Entidad","cargo":"Cargo","fechaFin":"2025-01-01","motivo":"Inicial"}}`, `{"q7Detalles":{"motivo":"Actualizado"}}`} {
		if _, err := s.SaveSection(ctx, id, 9, json.RawMessage(raw)); err != nil {
			t.Fatal(err)
		}
	}
	f, _ := DecodeForm(repo.item.Content)
	if f.Disqualifications.Q7Details.Institution != "Entidad" || f.Disqualifications.Q7Details.Reason != "Actualizado" {
		t.Fatal("se perdieron detalles al mezclar")
	}
	item, err := s.SaveSection(ctx, id, 9, json.RawMessage(`{"q7":"No","q7Detalles":null}`))
	if err != nil {
		t.Fatal(err)
	}
	f, _ = DecodeForm(item.Content)
	if f.Disqualifications.Q7Details != nil {
		t.Fatal("no se eliminaron los detalles")
	}
}
