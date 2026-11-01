package evaluation

import (
	"encoding/json"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"math/big"
	"slices"
	"strings"
)

func validText(s string, n int) bool { return strings.TrimSpace(s) != "" && len(s) <= n }
func Digest(v interface{}) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	canonical, err := submission.CanonicalContent(raw)
	if err != nil {
		return "", err
	}
	return submission.HashContent(canonical), nil
}
func (p Policy) Validate() error {
	if p.Kind == "oposicion" {
		if p.Definition.Opposition == nil || p.Definition.Reconsideration != nil {
			return ErrInvalid
		}
		if err := p.Definition.Opposition.Validate(); err != nil {
			return err
		}
	} else if p.Kind == "reconsideracion" {
		if p.Definition.Reconsideration == nil || p.Definition.Opposition != nil {
			return ErrInvalid
		}
		if err := p.Definition.Reconsideration.Validate(); err != nil {
			return err
		}
	} else {
		return ErrInvalid
	}
	hash, err := Digest(p.Definition)
	if err != nil || hash != p.Hash {
		return ErrInvalid
	}
	return nil
}

/** Validate exige escalas y ponderaciones explícitas, nunca una nota o umbral institucional precargado. */
func (r OppositionRules) Validate() error {
	if !validText(r.Instrument, 200) || !validText(r.InstrumentVersion, 100) || r.TheoryMaximum < 1 || r.TheoryMaximum > 1e9 || r.TotalMaximum < 1 || r.TotalMaximum > 1e9 || r.Threshold < 0 || r.Threshold > r.TotalMaximum || (r.Rounding != "inferior" && r.Rounding != "mas_cercano") {
		return ErrInvalid
	}
	if r.MeritWeight < 0 || r.MeritWeight > 10000 || r.TheoryWeight < 0 || r.TheoryWeight > 10000 || r.PsychometricWeight < 0 || r.PsychometricWeight > 10000 || r.MeritWeight+r.TheoryWeight+r.PsychometricWeight != 10000 {
		return ErrInvalid
	}
	if r.PsychometricMode == "puntaje" {
		if r.PsychometricMaximum == nil || *r.PsychometricMaximum < 1 || *r.PsychometricMaximum > 1e9 {
			return ErrInvalid
		}
	} else if r.PsychometricMode == "concepto" {
		if r.PsychometricMaximum != nil || r.PsychometricWeight != 0 {
			return ErrInvalid
		}
	} else {
		return ErrInvalid
	}
	if len(r.Conclusions) == 0 || len(r.Conclusions) > 20 || len(r.AcceptedConclusions) == 0 || len(r.AcceptedConclusions) > len(r.Conclusions) {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, v := range r.Conclusions {
		if !validText(v, 100) || seen[v] {
			return ErrInvalid
		}
		seen[v] = true
	}
	seen = map[string]bool{}
	for _, v := range r.AcceptedConclusions {
		if !slices.Contains(r.Conclusions, v) || seen[v] {
			return ErrInvalid
		}
		seen[v] = true
	}
	maxima := int64(0)
	for _, w := range []int64{r.MeritWeight, r.TheoryWeight, r.PsychometricWeight} {
		maxima += contribution(r.TotalMaximum, 1, 1, w, r.Rounding)
	}
	if maxima != r.TotalMaximum {
		return ErrInvalid
	}
	return nil
}
func contribution(total, score, maximum, weight int64, rounding string) int64 {
	n := new(big.Int).Mul(big.NewInt(total), big.NewInt(score))
	n.Mul(n, big.NewInt(weight))
	d := new(big.Int).Mul(big.NewInt(maximum), big.NewInt(10000))
	if rounding == "mas_cercano" {
		n.Add(n, new(big.Int).Div(new(big.Int).Set(d), big.NewInt(2)))
	}
	return n.Div(n, d).Int64()
}

/** Consolidate sólo usa tres resultados publicados vinculados al mismo sello y no acepta puntajes del cliente. */
func Consolidate(c *Case, r OppositionRules) ([]Component, int64, bool, error) {
	if c == nil || c.Snapshot == nil || c.OppositionPolicy == nil {
		return nil, 0, false, ErrPending
	}
	if err := r.Validate(); err != nil {
		return nil, 0, false, err
	}
	parts := []Component{}
	sum := int64(0)
	advance := true
	ids := map[string]bool{}
	for i, phase := range []string{"meritos", "oposicion_teorica", "oposicion_psicotecnica"} {
		source, ok := c.Sources[phase]
		if !ok || source.ID == [16]byte{} || source.SubmissionID != c.Snapshot.ID || source.Phase != phase || source.SnapshotHash != c.Snapshot.Hash || ids[source.ID.String()] {
			return nil, 0, false, ErrPending
		}
		ids[source.ID.String()] = true
		weight := []int64{r.MeritWeight, r.TheoryWeight, r.PsychometricWeight}[i]
		score, maximum := int64(0), int64(1)
		if phase != "oposicion_psicotecnica" || r.PsychometricMode == "puntaje" {
			if source.Score == nil || source.Maximum == nil || *source.Maximum <= 0 || *source.Score < 0 || *source.Score > *source.Maximum {
				return nil, 0, false, ErrPending
			}
			score, maximum = *source.Score, *source.Maximum
		}
		if phase == "oposicion_teorica" && maximum != r.TheoryMaximum {
			return nil, 0, false, ErrPolicy
		}
		if phase == "oposicion_psicotecnica" && (source.PolicyID != c.OppositionPolicy.ID || source.PolicyHash != c.OppositionPolicy.Hash) {
			return nil, 0, false, ErrConflict
		}
		if phase == "oposicion_psicotecnica" && ((r.PsychometricMode == "puntaje" && maximum != *r.PsychometricMaximum) || (r.PsychometricMode == "concepto" && (source.Score != nil || source.Maximum != nil))) {
			return nil, 0, false, ErrPolicy
		}
		points := contribution(r.TotalMaximum, score, maximum, weight, r.Rounding)
		parts = append(parts, Component{Phase: phase, ResultID: source.ID, Score: score, Maximum: maximum, Weight: weight, Contribution: points})
		sum += points
		advance = advance && source.CanAdvance
	}
	return parts, sum, advance && sum >= r.Threshold, nil
}
