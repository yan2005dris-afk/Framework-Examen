package competition

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func validCompetition() Competition {
	start := time.Date(2026, 10, 1, 5, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	return Competition{ID: uuid.New(), Code: "QA", Title: "Convocatoria sintética", OpensAt: start, ClosesAt: end, Configuration: Configuration{Version: 1, Approved: true, Published: true, FormEnabled: true, Sectors: []string{"afiliados"}, Phases: []Phase{{Code: "postulacion", Title: "Postulación", OpensAt: start, ClosesAt: end, Published: true, Enabled: true}}}}
}
func TestCalendarIsInclusiveAtOpeningAndExclusiveAtClosure(t *testing.T) {
	c := validCompetition()
	for _, tt := range []struct {
		name   string
		now    time.Time
		sector string
		want   bool
	}{{"antes", c.OpensAt.Add(-time.Nanosecond), "", false}, {"apertura", c.OpensAt, "afiliados", true}, {"instante_anterior_cierre", c.ClosesAt.Add(-time.Nanosecond), "", true}, {"cierre", c.ClosesAt, "", false}, {"sector_no_habilitado", c.OpensAt, "empleadores", false}} {
		t.Run(tt.name, func(t *testing.T) {
			if c.Writable(tt.now, tt.sector) != tt.want {
				t.Fatal("ventana incorrecta")
			}
		})
	}
}
func TestConfigurationRequiresApprovalPublicationAndRealPhase(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*Competition)
	}{{"sin_aprobacion", func(c *Competition) { c.Approved = false }}, {"sin_publicacion", func(c *Competition) { c.Published = false }}, {"sin_fase", func(c *Competition) { c.Phases = nil }}, {"fase_fuera_del_periodo", func(c *Competition) { c.Phases[0].ClosesAt = c.ClosesAt.Add(time.Second) }}, {"sector_duplicado", func(c *Competition) { c.Sectors = []string{"afiliados", "afiliados"} }}, {"fase_inventada", func(c *Competition) { c.Phases[0].Code = "arbitraria" }}} {
		t.Run(tt.name, func(t *testing.T) {
			c := validCompetition()
			tt.change(&c)
			if c.Validate() == nil {
				t.Fatal("configuración inválida aceptada")
			}
		})
	}
	c := validCompetition()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.FormEnabled = false
	c.Published = false
	c.Approved = false
	if c.Writable(c.OpensAt, "") {
		t.Fatal("borrador habilitado")
	}
}
func TestPublicViewNeverIncludesUnpublishedPhasesOrOperator(t *testing.T) {
	c := validCompetition()
	c.Phases = append(c.Phases, Phase{Code: "meritos", Title: "Privada"})
	v := publicView(c)
	if len(v.Phases) != 1 || v.TimeZone != TimeZone || v.Interval != "[apertura,cierre)" {
		t.Fatal("vista pública incorrecta")
	}
}
