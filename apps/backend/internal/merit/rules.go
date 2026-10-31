package merit

import (
	"encoding/json"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"math/big"
	"regexp"
	"strings"
)

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func validSector(s string) bool        { return s == "afiliados" || s == "jubilados" || s == "empleadores" }
func validText(s string, max int) bool { return strings.TrimSpace(s) != "" && len(s) <= max }

/** Validate exige una matriz explícita. No contiene ponderaciones, umbrales o filtros de exclusión precargados. */
func (r RuleDefinition) Validate() error {
	if len(r.Criteria) == 0 || len(r.Criteria) > 50 || len(r.Requirements) == 0 || len(r.Requirements) > 100 || len(r.Questions) != 13 || r.MaximumTotal == nil || r.Threshold == nil || *r.MaximumTotal <= 0 || *r.MaximumTotal > 1_000_000_000 || *r.Threshold < 0 || *r.Threshold > *r.MaximumTotal {
		return ErrInvalid
	}
	if r.Rounding != "inferior" && r.Rounding != "mas_cercano" {
		return ErrInvalid
	}
	if r.ExperienceAggregation != "unir_intervalos" && r.ExperienceAggregation != "sumar_registros" {
		return ErrInvalid
	}
	if r.DayCounting != "dias_transcurridos" && r.DayCounting != "dias_inclusivos" {
		return ErrInvalid
	}
	weights, total := int64(0), int64(0)
	codes := map[string]bool{}
	for _, c := range r.Criteria {
		if !codePattern.MatchString(c.Code) || codes[c.Code] || !validText(c.Label, 200) || c.UnitSize < 1 || c.UnitSize > 1_000_000_000 || c.PointsPerUnit < 1 || c.PointsPerUnit > 1_000_000_000 || c.MaximumPoints < 1 || c.MaximumPoints > 1_000_000_000 || c.Weight < 1 || c.Weight > 10_000 || !validMetric(c.Metric, c.Filter) {
			return ErrInvalid
		}
		codes[c.Code] = true
		weights += c.Weight
		total += weighted(c.MaximumPoints, c.Weight, r.Rounding)
	}
	if weights != 10_000 || total != *r.MaximumTotal {
		return ErrInvalid
	}
	reqs := map[string]bool{}
	for _, q := range r.Requirements {
		if !codePattern.MatchString(q.Code) || reqs[q.Code] || !validText(q.Label, 500) {
			return ErrInvalid
		}
		reqs[q.Code] = true
	}
	questions := map[string]bool{}
	for _, q := range r.Questions {
		if !questionID(q.ID) || questions[q.ID] || (q.AcceptedAnswer != "Sí" && q.AcceptedAnswer != "No") || (q.ID == "q1" && q.AcceptedAnswer != "Sí") {
			return ErrInvalid
		}
		questions[q.ID] = true
	}
	if len(r.TieBreakers) == 0 || len(r.TieBreakers) > len(codes)+1 || r.TieBreakers[len(r.TieBreakers)-1] != "resolucion_manual" {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, c := range r.TieBreakers {
		if seen[c] || c != "resolucion_manual" && !codes[c] {
			return ErrInvalid
		}
		seen[c] = true
	}
	return nil
}
func questionID(s string) bool {
	switch s {
	case "q1", "q2", "q3", "q4", "q5", "q6", "q7", "q8", "q9", "q10", "q11", "q12", "q13":
		return true
	}
	return false
}
func validMetric(metric, filter string) bool {
	switch metric {
	case "education_count":
		return filter == "" || filter == "Tercer Nivel de Grado" || filter == "Cuarto Nivel - Especialidad Académica" || filter == "Cuarto Nivel - Maestría Académica" || filter == "Cuarto Nivel PhD - Doctorado"
	case "experience_days":
		return filter == "" || filter == "General" || filter == "Dirección"
	case "publication_count":
		return filter == "" || filter == "Publicación" || filter == "Trabajo de investigación"
	case "training_hours":
		return filter == "" || filter == "Curso" || filter == "Seminario" || filter == "Taller"
	}
	return false
}
func weighted(points, weight int64, rounding string) int64 {
	return quotient(new(big.Int).Mul(big.NewInt(points), big.NewInt(weight)), big.NewInt(10_000), rounding)
}
func quotient(n, d *big.Int, rounding string) int64 {
	if rounding == "mas_cercano" {
		n.Add(n, new(big.Int).Div(new(big.Int).Set(d), big.NewInt(2)))
	}
	return n.Div(n, d).Int64()
}
func score(units int64, c Criterion, rounding string) int64 {
	n := new(big.Int).Mul(big.NewInt(units), big.NewInt(c.PointsPerUnit))
	d := big.NewInt(c.UnitSize)
	cap := new(big.Int).Mul(big.NewInt(c.MaximumPoints), d)
	if n.Cmp(cap) > 0 {
		n = cap
	}
	n.Mul(n, big.NewInt(c.Weight))
	d.Mul(d, big.NewInt(10_000))
	return quotient(n, d, rounding)
}
func definitionHash(def RuleDefinition) (string, error) {
	raw, err := json.Marshal(def)
	if err != nil {
		return "", err
	}
	canonical, err := submission.CanonicalContent(raw)
	if err != nil {
		return "", err
	}
	return submission.HashContent(canonical), nil
}

/** Validate comprueba integridad y configuración antes de aprobar o usar una versión. */
func (r RuleSet) Validate() error {
	if err := r.Definition.Validate(); err != nil {
		return err
	}
	hash, err := definitionHash(r.Definition)
	if err != nil || hash != r.Hash {
		return ErrEvidence
	}
	return nil
}
