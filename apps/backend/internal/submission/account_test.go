package submission

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

type accountSource struct{ account *identity.Applicant }

func (r accountSource) GetByID(_ context.Context, id uuid.UUID) (*identity.Applicant, error) {
	if r.account == nil || id != r.account.ID {
		return nil, identity.ErrApplicantNotFound
	}
	copy := *r.account
	return &copy, nil
}

func TestAccountPrefillAndProtectionForEveryWrite(t *testing.T) {
	s, repo, _ := serviceFixture(t, false)
	account := &identity.Applicant{ID: repo.item.ApplicantID, IDCard: "1710034008", FirstName: "Nombre de cuenta", LastName: "Apellido de cuenta", Email: "cuenta@example.test"}
	s.WithApplicants(accountSource{account})
	ctx := context.Background()
	created, err := s.Create(ctx, CreateInput{ApplicantID: account.ID})
	if err != nil {
		t.Fatal(err)
	}
	form, _ := DecodeForm(created.Content)
	if form.Personal.FirstName != account.FirstName || form.Personal.LastName != account.LastName || form.Personal.IDCard != account.IDCard.String() || form.Contact.Email != account.Email {
		t.Fatal("faltan datos de la cuenta")
	}
	if form.Personal.CivilRegistryVerified || form.Personal.Nationality != "" || form.Personal.BirthDate != "" {
		t.Fatal("se certificaron o inventaron datos personales")
	}
	if form.Consent == nil || len(form.Consent.Statements) != 5 || form.Consent.AcceptedAt != "" || form.Consent.Truthfulness {
		t.Fatal("consentimiento automático o sin literales")
	}
	for _, raw := range []string{`{"seccion1":{"cedula":"1710034016"}}`, `{"seccion2":{"correo1":"otra@example.test"}}`} {
		before := string(repo.item.Content)
		if _, err := s.Update(ctx, created.ID, json.RawMessage(raw)); err == nil {
			t.Fatal("se aceptó el cambio de identidad o correo")
		}
		if string(repo.item.Content) != before {
			t.Fatal("se escribió un rechazo")
		}
	}
	if _, err := s.SaveSection(ctx, created.ID, 1, json.RawMessage(`{"cedula":"1710034016"}`)); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	updated, err := s.SaveSection(ctx, created.ID, 1, json.RawMessage(`{"nombres":"Nombre declarado corregido","verificadoRegistroCivil":true}`))
	if err != nil {
		t.Fatal(err)
	}
	form, _ = DecodeForm(updated.Content)
	if form.Personal.FirstName != "Nombre declarado corregido" || form.Personal.CivilRegistryVerified {
		t.Fatal("no se respetó la declaración OTP")
	}
}

func TestConsentTextIsPublishedBeforeSubmitButOnlySubmitDatesAcceptance(t *testing.T) {
	s, repo, _ := serviceFixture(t, true)
	item, err := s.Update(context.Background(), repo.item.ID, repo.item.Content)
	if err != nil {
		t.Fatal(err)
	}
	form, _ := DecodeForm(item.Content)
	if form.Consent.Version != consentVersion || len(form.Consent.Statements) != 5 || form.Consent.AcceptedAt != "" {
		t.Fatal("faltan literales al guardar")
	}
	form.Personal.CivilRegistryVerified = true
	form.Consent.Version = "inventada"
	form.Consent.Statements = []string{"texto del cliente"}
	form.Consent.AcceptedAt = "1999-01-01T00:00:00Z"
	raw, _ := json.Marshal(form)
	if _, err := s.Update(context.Background(), item.ID, raw); err != nil {
		t.Fatal(err)
	}
	sealed, err := s.Submit(context.Background(), item.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	form, _ = DecodeForm(sealed.Content)
	if form.Personal.CivilRegistryVerified || form.Consent.Version != consentVersion || len(form.Consent.Statements) != 5 || !strings.HasPrefix(form.Consent.AcceptedAt, "2026-10-31T") {
		t.Fatal("sello con metadatos del cliente")
	}
	before := string(sealed.Content)
	read, err := s.Get(context.Background(), item.ID)
	if err != nil || string(read.Content) != before || read.Hash != sealed.Hash {
		t.Fatal("se modificó el snapshot al leer", err)
	}
}

func TestDefinitionHasLiteralQuestionsCatalogsAndDefensiveCopies(t *testing.T) {
	definition, err := FormDefinition()
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.Sections) != 9 || len(definition.Questions) != 13 || len(definition.Consent.Statements) != 5 || len(definition.Provinces) != 24 {
		t.Fatal("definición incompleta")
	}
	if !strings.Contains(definition.Questions[0].Text, "pleno goce") || !strings.Contains(definition.Questions[12].Text, "Unidad de Análisis Financiero") {
		t.Fatal("preguntas del RF incompletas")
	}
	definition.Consent.Statements[0] = "alterado"
	definition.Options["parentescos"][0] = "alterado"
	definition.Provinces["Azuay"][0] = "alterado"
	next, err := FormDefinition()
	if err != nil {
		t.Fatal(err)
	}
	if next.Consent.Statements[0] == "alterado" || next.Options["parentescos"][0] == "alterado" || next.Provinces["Azuay"][0] == "alterado" {
		t.Fatal("se modificó el catálogo compartido")
	}
}
