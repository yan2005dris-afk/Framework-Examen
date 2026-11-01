// Package testfixture produce exclusivamente expedientes sintéticos para pruebas de listas.
package testfixture

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"strings"
	"time"
)

func Policy(period uuid.UUID) *evaluation.Policy {
	maximum := int64(100)
	p := &evaluation.Policy{ID: uuid.New(), CompetitionID: period, Sector: "afiliados", Kind: "oposicion", Approved: true, Version: 1, Definition: evaluation.Definition{Opposition: &evaluation.OppositionRules{Instrument: "QA NO OFICIAL", InstrumentVersion: "QA-1", PsychometricMode: "puntaje", PsychometricMaximum: &maximum, Conclusions: []string{"QA"}, AcceptedConclusions: []string{"QA"}, TheoryMaximum: 100, MeritWeight: 4000, TheoryWeight: 4000, PsychometricWeight: 2000, TotalMaximum: 10000, Threshold: 6000, Rounding: "inferior"}}}
	p.Hash, _ = evaluation.Digest(p.Definition)
	return p
}
func Source(period uuid.UUID, p *evaluation.Policy, score int64) (results.Source, error) {
	now := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	definition, err := submission.FormDefinition()
	if err != nil {
		return results.Source{}, err
	}
	consent := definition.Consent
	consent.NamePublication = true
	consent.AcceptedAt = now.Format(time.RFC3339Nano)
	form := submission.Form{Personal: &submission.PersonalSection{FirstName: "Persona QA", LastName: "Apellido QA", IDCard: "0000000000"}, Position: &submission.PositionSection{Sector: "afiliados"}, Contact: &submission.ContactSection{Email: "privado@example.test", Mobile: "NO-PUBLICAR"}, Consent: consent}
	raw, err := json.Marshal(form)
	if err != nil {
		return results.Source{}, err
	}
	raw, err = submission.CanonicalContent(raw)
	if err != nil {
		return results.Source{}, err
	}
	snap := &submission.Submission{ID: uuid.New(), ApplicantID: uuid.New(), CompetitionID: &period, Code: "QA-" + uuid.NewString()[:28], Content: raw, Hash: submission.HashContent(raw), Status: submission.StatusSubmitted, SubmittedAt: &now, CreatedAt: now, UpdatedAt: now}
	maximum := int64(100)
	meritResult := &merit.Publication{ID: uuid.New(), SubmissionID: snap.ID, Version: 1, CalculationID: uuid.New(), RuleSetID: uuid.New(), RulesHash: strings.Repeat("a", 64), SnapshotHash: snap.Hash, Decision: "avanza", CreatedAt: now, Evaluation: merit.Evaluation{Total: score, Maximum: 100, CanAdvance: true}}
	sources := map[string]evaluation.Result{}
	for _, phase := range []string{"meritos", "oposicion_teorica", "oposicion_psicotecnica"} {
		id := uuid.New()
		if phase == "meritos" {
			id = meritResult.ID
		}
		sources[phase] = evaluation.Result{ID: id, SubmissionID: snap.ID, Phase: phase, Version: 1, SourceID: uuid.New(), PolicyID: p.ID, PolicyHash: p.Hash, SnapshotHash: snap.Hash, Score: &score, Maximum: &maximum, CanAdvance: true, PublishedAt: now}
	}
	parts, total, advance, err := evaluation.Consolidate(&evaluation.Case{Snapshot: snap, OppositionPolicy: p, Sources: sources}, *p.Definition.Opposition)
	if err != nil {
		return results.Source{}, err
	}
	maxTotal := int64(10000)
	final := evaluation.Result{ID: uuid.New(), SubmissionID: snap.ID, Phase: "resultados", Version: 1, PolicyID: p.ID, PolicyHash: p.Hash, SnapshotHash: snap.Hash, Score: &total, Maximum: &maxTotal, CanAdvance: advance, Breakdown: parts, PublishedAt: now}
	theory := sources["oposicion_teorica"]
	return results.Source{Snapshot: snap, Merit: meritResult, Theory: &theory, State: evaluation.State{Results: map[string]evaluation.Result{"oposicion_psicotecnica": sources["oposicion_psicotecnica"], "resultados": final}}}, nil
}
