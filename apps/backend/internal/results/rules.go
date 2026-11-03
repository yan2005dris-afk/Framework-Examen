package results

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"math/big"
	"slices"
	"sort"
	"strings"
)

func text(s string, n int) bool { return strings.TrimSpace(s) != "" && len(s) <= n }
func ValidSector(s string) bool {
	return slices.Contains([]string{"afiliados", "jubilados", "empleadores"}, s)
}
func (r Rules) Validate() error {
	if !text(r.Reference, 2000) || len(r.Order) < 1 || len(r.Order) > 4 || r.Order[0] != "resultados" {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, v := range r.Order {
		if !slices.Contains([]string{"resultados", "meritos", "oposicion_teorica", "oposicion_psicotecnica"}, v) || seen[v] {
			return ErrInvalid
		}
		seen[v] = true
	}
	return nil
}

/** Build exige fuentes actuales del mismo sello y congela las resoluciones. No inventa notas ausentes ni adjudica desempates. */
func Build(period uuid.UUID, sector string, r Rules, policy *evaluation.Policy, sources []Source) ([]Entry, string, error) {
	if err := r.Validate(); err != nil {
		return nil, "", err
	}
	if len(sources) > MaxRows {
		return nil, "", ErrTooLarge
	}
	items := []Entry{}
	seen := map[uuid.UUID]bool{}
	for _, s := range sources {
		snap := s.Snapshot
		if submission.ValidateSeal(snap) != nil || snap.CompetitionID == nil || *snap.CompetitionID != period {
			return nil, "", ErrInvalid
		}
		form, err := submission.DecodeForm(snap.Content)
		if err != nil || form.Position == nil || form.Position.Sector != sector || form.Personal == nil || !text(form.Personal.FirstName, 200) || !text(form.Personal.LastName, 200) {
			return nil, "", ErrInvalid
		}
		if seen[snap.ApplicantID] {
			return nil, "", ErrConflict
		}
		seen[snap.ApplicantID] = true
		m := s.Merit
		if m == nil {
			return nil, "", ErrPending
		}
		if m.ID == uuid.Nil || m.SubmissionID != snap.ID || m.SnapshotHash != snap.Hash || m.Version < 1 || m.Evaluation.Maximum <= 0 || m.Evaluation.Total < 0 || m.Evaluation.Total > m.Evaluation.Maximum {
			return nil, "", ErrInvalid
		}
		score, maximum := m.Evaluation.Total, m.Evaluation.Maximum
		e := Entry{SubmissionID: snap.ID, Code: snap.Code, Hash: snap.Hash, FirstName: form.Personal.FirstName, LastName: form.Personal.LastName, NameConsent: submission.AllowsNamePublication(form.Consent), Results: map[string]evaluation.Result{}, Appeals: []AppealRef{}}
		e.Results["meritos"] = evaluation.Result{ID: m.ID, SubmissionID: snap.ID, Phase: "meritos", Version: m.Version, SourceID: m.CalculationID, PolicyID: m.RuleSetID, PolicyHash: m.RulesHash, SnapshotHash: snap.Hash, Score: &score, Maximum: &maximum, CanAdvance: m.Decision == "avanza", PublishedAt: m.CreatedAt}
		for phase, v := range s.State.Results {
			if v.SubmissionID != snap.ID || v.Phase != phase || v.SnapshotHash != snap.Hash {
				return nil, "", ErrInvalid
			}
			e.Results[phase] = v
		}
		if s.Theory != nil {
			e.Results["oposicion_teorica"] = *s.Theory
		}
		switch m.Decision {
		case "excluido":
			e.Eligible = false
		case "avanza":
			if policy == nil || !policy.Approved || policy.Kind != "oposicion" || policy.CompetitionID != period || policy.Sector != sector || policy.Validate() != nil {
				return nil, "", ErrPending
			}
			c := &evaluation.Case{Snapshot: snap, Sources: e.Results, OppositionPolicy: policy}
			parts, total, advance, err := evaluation.Consolidate(c, *policy.Definition.Opposition)
			if err != nil {
				return nil, "", ErrPending
			}
			result, ok := e.Results["resultados"]
			if !ok {
				return nil, "", ErrPending
			}
			if result.PolicyID != policy.ID || result.PolicyHash != policy.Hash || result.Score == nil || result.Maximum == nil || *result.Score != total || *result.Maximum != policy.Definition.Opposition.TotalMaximum || result.CanAdvance != advance {
				return nil, "", ErrConflict
			}
			if !slices.Equal(parts, result.Breakdown) {
				return nil, "", ErrConflict
			}
			e.Eligible = advance
		default:
			return nil, "", ErrInvalid
		}
		for _, a := range s.Appeals {
			if a.Status != "resuelta" || a.Resolution == nil {
				return nil, "", ErrPending
			}
			v, ok := e.Results[a.Original.Phase]
			effective := a.Resolution.EffectiveResult
			if !ok || a.SubmissionID != snap.ID || a.Original.SnapshotHash != snap.Hash || effective.SubmissionID != snap.ID || effective.SnapshotHash != snap.Hash || effective.Phase != a.Original.Phase || effective.Version > v.Version || (effective.Version == v.Version && effective.ID != v.ID) {
				return nil, "", ErrConflict
			}
			e.Appeals = append(e.Appeals, AppealRef{ID: a.ID, Version: a.Version, EffectiveID: effective.ID})
		}
		sort.Slice(e.Appeals, func(i, j int) bool { return e.Appeals[i].ID.String() < e.Appeals[j].ID.String() })
		if e.Eligible {
			for _, phase := range r.Order {
				v := e.Results[phase]
				if v.Score == nil || v.Maximum == nil || *v.Maximum <= 0 || *v.Score < 0 || *v.Score > *v.Maximum {
					return nil, "", ErrInvalid
				}
			}
		}
		items = append(items, e)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Eligible != items[j].Eligible {
			return items[i].Eligible
		}
		v := compare(items[i], items[j], r.Order)
		if v != 0 {
			return v > 0
		}
		return items[i].SubmissionID.String() < items[j].SubmissionID.String()
	})
	for i := range items {
		if !items[i].Eligible {
			continue
		}
		items[i].Rank = i + 1
		if i > 0 && items[i-1].Eligible && compare(items[i-1], items[i], r.Order) == 0 {
			items[i].Rank = items[i-1].Rank
			items[i].Tied = true
			items[i-1].Tied = true
		}
	}
	hash, err := evaluation.Digest(struct {
		Rules   Rules
		Entries []Entry
	}{r, items})
	return items, hash, err
}
func compare(a, b Entry, order []string) int {
	for _, phase := range order {
		x, y := a.Results[phase], b.Results[phase]
		if x.Score == nil || y.Score == nil || x.Maximum == nil || y.Maximum == nil {
			continue
		}
		l := new(big.Int).Mul(big.NewInt(*x.Score), big.NewInt(*y.Maximum))
		r := new(big.Int).Mul(big.NewInt(*y.Score), big.NewInt(*x.Maximum))
		if v := l.Cmp(r); v != 0 {
			return v
		}
	}
	return 0
}
func EncodeList(v *List) ([]byte, error) {
	return json.Marshal(struct {
		*List
		Items []Entry `json:"items"`
	}{v, v.Entries})
}
func DecodeList(raw []byte) (*List, error) {
	var v struct {
		List
		Items []Entry `json:"items"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	v.List.Entries = v.Items
	return &v.List, nil
}

/** PublicNames usa únicamente nombres autorizados y orden alfabético, nunca puntajes, ranking ni identificadores de postulantes. */
func PublicNames(items []Entry) ([]string, error) {
	names := []string{}
	for _, e := range items {
		if !e.Eligible {
			continue
		}
		if !e.NameConsent {
			return nil, ErrPrivacy
		}
		names = append(names, strings.TrimSpace(e.LastName+" "+e.FirstName))
	}
	sort.Strings(names)
	return names, nil
}
