package evaluation

import (
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"testing"
	"time"
)

func syntheticOpposition() OppositionRules {
	maximum := int64(100)
	return OppositionRules{Instrument: "INSTRUMENTO QA NO OFICIAL", InstrumentVersion: "QA-1", PsychometricMode: "puntaje", PsychometricMaximum: &maximum, Conclusions: []string{"acredita", "no_acredita"}, AcceptedConclusions: []string{"acredita"}, TheoryMaximum: 100, MeritWeight: 4000, TheoryWeight: 4000, PsychometricWeight: 2000, TotalMaximum: 10000, Threshold: 6000, Rounding: "inferior"}
}
func syntheticCalendar() Calendar {
	return Calendar{TimeZone: "America/Guayaquil", Weekdays: []int{1, 2, 3, 4, 5}, Holidays: []string{"2026-11-02"}, EndPolicy: "exigir_termino_completo"}
}
func TestDeadlineCalendarBusinessAndElapsedUnits(t *testing.T) {
	anchor := time.Date(2026, 10, 30, 15, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		term Term
		want string
	}{{"fin_de_semana_y_feriado", Term{Quantity: 2, Unit: "dias_habiles", Cutoff: "17:00"}, "2026-11-04T22:00:00Z"}, {"incluye_dia_inicial", Term{Quantity: 1, Unit: "dias_habiles", IncludeAnchorDay: true, Cutoff: "17:00"}, "2026-10-30T22:00:00Z"}, {"dias_calendario", Term{Quantity: 2, Unit: "dias_calendario", Cutoff: "17:00"}, "2026-11-01T22:00:00Z"}, {"minutos_exactos", Term{Quantity: 90, Unit: "minutos_transcurridos"}, "2026-10-30T16:30:00Z"}} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Deadline(anchor, tc.term, syntheticCalendar())
			if err != nil || got.Format(time.RFC3339) != tc.want {
				t.Fatal(got, err)
			}
		})
	}
}
func TestDeadlinesRequireExplicitApprovedComputation(t *testing.T) {
	calendar := syntheticCalendar()
	term := Term{Quantity: 1, Unit: "dias_calendario", Cutoff: "17:00"}
	anchor := time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)
	for _, name := range []string{"zero", "weekdays", "timezone", "holiday", "cutoff", "unit", "end_policy"} {
		t.Run(name, func(t *testing.T) {
			c := calendar
			v := term
			switch name {
			case "zero":
				v.Quantity = 0
			case "weekdays":
				v.Unit = "dias_habiles"
				c.Weekdays = nil
			case "timezone":
				c.TimeZone = "equipo_local"
			case "holiday":
				c.Holidays = []string{"2026-02-30"}
			case "cutoff":
				v.Cutoff = "24:00"
			case "unit":
				v.Unit = "dias_legales_supuesto"
			case "end_policy":
				c.EndPolicy = ""
			}
			if _, err := Deadline(anchor, v, c); !errors.Is(err, ErrInvalid) {
				t.Fatal("aceptó plazo sin definición", err)
			}
		})
	}
	end := anchor.Add(time.Hour)
	if _, err := boundedDeadline(anchor, term, calendar, end); !errors.Is(err, ErrDeadline) {
		t.Fatal("acortó término completo", err)
	}
	calendar.EndPolicy = "limitar_a_fase"
	got, err := boundedDeadline(anchor, term, calendar, end)
	if err != nil || !got.Equal(end) {
		t.Fatal(got, err)
	}
	c := &Case{Now: end, PhaseStarts: map[string]time.Time{"reconsideracion": anchor}, PhaseEnds: map[string]time.Time{"reconsideracion": end}}
	if err = phaseWindow(c, "reconsideracion"); !errors.Is(err, ErrPhase) {
		t.Fatal("aceptó instante de cierre")
	}
	c.Now = anchor
	if err = phaseWindow(c, "reconsideracion"); err != nil {
		t.Fatal("rechazó apertura", err)
	}
}
func TestOppositionRulesDoNotInventScales(t *testing.T) {
	for _, name := range []string{"instrument", "version", "weight", "maximum", "threshold", "mode", "conclusion", "concept_weight"} {
		t.Run(name, func(t *testing.T) {
			r := syntheticOpposition()
			switch name {
			case "instrument":
				r.Instrument = ""
			case "version":
				r.InstrumentVersion = ""
			case "weight":
				r.TheoryWeight = 1
			case "maximum":
				r.PsychometricMaximum = nil
			case "threshold":
				r.Threshold = 10001
			case "mode":
				r.PsychometricMode = "auto"
			case "conclusion":
				r.AcceptedConclusions = []string{"inventada"}
			case "concept_weight":
				r.PsychometricMode = "concepto"
				r.PsychometricMaximum = nil
			}
			if err := r.Validate(); !errors.Is(err, ErrInvalid) {
				t.Fatal("aceptó definición incompleta", err)
			}
		})
	}
}
func syntheticCase() *Case {
	c := &Case{Snapshot: &submission.Submission{ID: uuid.New(), Hash: "hash-de-prueba"}, OppositionPolicy: &Policy{ID: uuid.New(), Hash: "reglas-qa"}, Sources: map[string]Result{}}
	for _, phase := range []string{"meritos", "oposicion_teorica", "oposicion_psicotecnica"} {
		score, maximum := int64(80), int64(100)
		c.Sources[phase] = Result{ID: uuid.New(), SubmissionID: c.Snapshot.ID, Phase: phase, SnapshotHash: c.Snapshot.Hash, Score: &score, Maximum: &maximum, CanAdvance: true, PolicyID: c.OppositionPolicy.ID, PolicyHash: c.OppositionPolicy.Hash}
	}
	return c
}
func TestConsolidationUsesPublishedSourcesExactlyOnce(t *testing.T) {
	for _, incomplete := range []*Case{nil, {}, {Snapshot: &submission.Submission{}}} {
		if _, _, _, err := Consolidate(incomplete, syntheticOpposition()); !errors.Is(err, ErrPending) {
			t.Fatal("expediente incompleto aceptado", err)
		}
	}
	c := syntheticCase()
	parts, total, advance, err := Consolidate(c, syntheticOpposition())
	if err != nil || total != 8000 || len(parts) != 3 || !advance {
		t.Fatal(parts, total, advance, err)
	}
	for _, name := range []string{"missing_theory", "foreign", "seal", "duplicate", "maximum", "psycho_maximum", "obsolete_policy", "excluded"} {
		t.Run(name, func(t *testing.T) {
			c := syntheticCase()
			v := c.Sources["oposicion_teorica"]
			switch name {
			case "missing_theory":
				delete(c.Sources, "oposicion_teorica")
			case "foreign":
				v.SubmissionID = uuid.New()
				c.Sources[v.Phase] = v
			case "seal":
				v.SnapshotHash = "otro"
				c.Sources[v.Phase] = v
			case "duplicate":
				v.ID = c.Sources["meritos"].ID
				c.Sources[v.Phase] = v
			case "maximum":
				maximum := int64(200)
				v.Maximum = &maximum
				c.Sources[v.Phase] = v
			case "obsolete_policy":
				v = c.Sources["oposicion_psicotecnica"]
				v.PolicyID = uuid.New()
				c.Sources[v.Phase] = v
			case "psycho_maximum":
				v = c.Sources["oposicion_psicotecnica"]
				maximum := int64(200)
				v.Maximum = &maximum
				c.Sources[v.Phase] = v
			case "excluded":
				v.CanAdvance = false
				c.Sources[v.Phase] = v
			}
			_, _, advance, err := Consolidate(c, syntheticOpposition())
			if name == "excluded" {
				if err != nil || advance {
					t.Fatal("ignoró exclusión", err)
				}
			} else if err == nil {
				t.Fatal("consolidó evidencia inválida")
			}
		})
	}
}
func TestConsolidationConceptAndRounding(t *testing.T) {
	c := syntheticCase()
	r := syntheticOpposition()
	r.PsychometricMode = "concepto"
	r.PsychometricMaximum = nil
	r.PsychometricWeight = 0
	r.MeritWeight = 5000
	r.TheoryWeight = 5000
	v := c.Sources["oposicion_psicotecnica"]
	v.Score = nil
	v.Maximum = nil
	c.Sources[v.Phase] = v
	_, total, advance, err := Consolidate(c, r)
	if err != nil || total != 8000 || !advance {
		t.Fatal(total, advance, err)
	}
	if got := contribution(10000, 1, 3, 10000, "inferior"); got != 3333 {
		t.Fatal(got)
	}
	if got := contribution(10000, 2, 3, 10000, "mas_cercano"); got != 6667 {
		t.Fatal(got)
	}
}

func TestPsychometricMinimumCannotBeOverriddenByConclusion(t *testing.T) {
	r := syntheticOpposition()
	minimum := int64(60)
	r.PsychometricMinimum = &minimum
	for value, want := range map[int64]bool{59: false, 60: true, 100: true} {
		if r.AcceptsPsychometric(&value, "acredita") != want {
			t.Fatal("umbral no aplicado", value)
		}
	}
	if r.AcceptsPsychometric(nil, "acredita") {
		t.Fatal("umbral sin puntaje")
	}
	minimum = 101
	if r.Validate() == nil {
		t.Fatal("umbral fuera de escala")
	}
}
