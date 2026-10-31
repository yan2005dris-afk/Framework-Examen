package simulated

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"os"
	"path/filepath"
	"testing"
)

func TestSimulationRequiresExplicitServerFixture(t *testing.T) {
	id := uuid.New()
	path := filepath.Join(t.TempDir(), "conditions.json")
	if err := os.WriteFile(path, []byte(`{"`+id.String()+`":{"afiliados":true,"jubilados":false}}`), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		sector   string
		eligible bool
	}{{"afiliados", true}, {"jubilados", false}} {
		c, err := p.Resolve(context.Background(), id, tt.sector)
		if err != nil || c.Eligible != tt.eligible || c.Source != "simulacion_controlada_servidor" {
			t.Fatal(c, err)
		}
	}
	if _, err = p.Resolve(context.Background(), id, "empleadores"); !errors.Is(err, submission.ErrEligibilityUnavailable) {
		t.Fatal("dato ausente convertido en resultado", err)
	}
	if _, err = p.Resolve(context.Background(), uuid.New(), "afiliados"); !errors.Is(err, submission.ErrEligibilityUnavailable) {
		t.Fatal("cuenta inventada elegible", err)
	}
}
func TestInvalidFixtureFailsInsteadOfEnablingAllApplicants(t *testing.T) {
	for _, raw := range []string{`null`, `{"01900000-0000-7000-8000-000000000001":{"afiliados":null}}`, `{"cuenta":{"afiliados":true}}`, `{} {}`, `{"01900000-0000-7000-8000-000000000001":{"arbitrario":true}}`} {
		p := filepath.Join(t.TempDir(), "invalid.json")
		if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(p); err == nil {
			t.Fatal("fixture inválida aceptada")
		}
	}
}
