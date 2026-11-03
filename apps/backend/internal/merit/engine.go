package merit

import (
	"fmt"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"sort"
	"time"
	_ "time/tzdata"
)

type record struct {
	section  int
	id       string
	document *submission.DocumentReference
}

func recordKey(section int, id string) string { return fmt.Sprintf("%d:%s", section, id) }
func records(form *submission.Form) []record {
	result := []record{}
	if form.Position != nil && (form.Position.Document != nil || form.Position.HasOrganization == "Sí") {
		result = append(result, record{4, "", form.Position.Document})
	}
	if form.Education != nil {
		for _, r := range form.Education.Degrees {
			result = append(result, record{5, r.ID, r.Document})
		}
	}
	if form.Experience != nil {
		for _, r := range form.Experience.Experiences {
			result = append(result, record{6, r.ID, r.Document})
		}
	}
	if form.Publications != nil {
		for _, r := range form.Publications.Publications {
			result = append(result, record{7, r.ID, r.Document})
		}
	}
	if form.Training != nil {
		for _, r := range form.Training.Events {
			result = append(result, record{8, r.ID, r.Document})
		}
	}
	return result
}

/** ValidateReview vincula cada veredicto al registro y SHA del snapshot, no a valores numéricos declarados por el revisor. */
func ValidateReview(c *Case, input ReviewInput) error {
	if c == nil || c.Snapshot == nil {
		return ErrEvidence
	}
	if c.ActiveRules == nil || !c.ActiveRules.Approved || c.ActiveRules.Definition.Validate() != nil {
		return ErrRulesPending
	}
	hash, hashErr := definitionHash(c.ActiveRules.Definition)
	if hashErr != nil || hash != c.ActiveRules.Hash {
		return ErrEvidence
	}
	if input.RuleSetID != c.ActiveRules.ID {
		return ErrConflict
	}
	if !validText(input.Motive, 4000) {
		return ErrInvalid
	}
	if err := submission.ValidateSeal(c.Snapshot); err != nil {
		return err
	}
	form, err := submission.DecodeForm(c.Snapshot.Content)
	if err != nil {
		return ErrEvidence
	}
	manifest := map[string]record{}
	for _, r := range records(form) {
		key := recordKey(r.section, r.id)
		if manifest[key].section != 0 || r.section != 4 && r.id == "" {
			return ErrEvidence
		}
		manifest[key] = r
	}
	if len(input.Documents) != len(manifest) || len(input.Requirements) != len(c.ActiveRules.Definition.Requirements) || len(input.Questions) != 13 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, review := range input.Documents {
		key := recordKey(review.Section, review.RecordID)
		r, ok := manifest[key]
		if !ok || seen[key] || len(review.Observation) > 4000 {
			return ErrEvidence
		}
		seen[key] = true
		id := ""
		if r.document != nil {
			id = r.document.ID
		}
		if review.DocumentID != id {
			return ErrEvidence
		}
		switch review.Verdict {
		case "valido":
			if r.document == nil || len(r.document.SHA256) != 64 || !matchingDocument(c.Documents, r) {
				return ErrEvidence
			}
		case "invalido":
			if !validText(review.Observation, 4000) {
				return ErrInvalid
			}
		case "no_presentado":
			if id != "" || !validText(review.Observation, 4000) {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
	}
	reqs := map[string]bool{}
	for _, r := range c.ActiveRules.Definition.Requirements {
		reqs[r.Code] = true
	}
	seen = map[string]bool{}
	for _, r := range input.Requirements {
		if !reqs[r.Code] || seen[r.Code] || len(r.Observation) > 4000 || !r.Satisfied && !validText(r.Observation, 4000) {
			return ErrInvalid
		}
		seen[r.Code] = true
	}
	seen = map[string]bool{}
	for _, q := range input.Questions {
		if !questionID(q.ID) || seen[q.ID] || len(q.Observation) > 4000 || !q.Confirmed && !validText(q.Observation, 4000) {
			return ErrInvalid
		}
		seen[q.ID] = true
	}
	return nil
}
func matchingDocument(documents []Document, r record) bool {
	for _, d := range documents {
		if d.ID == r.document.ID && d.Section == r.section && d.RecordID == r.id && d.SHA256 == r.document.SHA256 {
			return true
		}
	}
	return false
}

/** Evaluate usa aritmética exacta, fechas del envío y sólo registros con respaldo revisado como válido. */
func Evaluate(c *Case, review *Review) (Evaluation, error) {
	if c == nil || c.Snapshot == nil {
		return Evaluation{}, ErrEvidence
	}
	if c.ActiveRules == nil || c.ActiveRules.Definition.Validate() != nil {
		return Evaluation{}, ErrRulesPending
	}
	hash, err := definitionHash(c.ActiveRules.Definition)
	if err != nil || hash != c.ActiveRules.Hash {
		return Evaluation{}, ErrEvidence
	}
	if review == nil || review.SnapshotHash != c.Snapshot.Hash || review.SubmissionID != c.Snapshot.ID {
		return Evaluation{}, ErrConflict
	}
	if err := ValidateReview(c, review.Input); err != nil {
		return Evaluation{}, err
	}
	form, err := submission.DecodeForm(c.Snapshot.Content)
	if err != nil {
		return Evaluation{}, ErrEvidence
	}
	def := c.ActiveRules.Definition
	valid := map[string]bool{}
	for _, r := range review.Input.Documents {
		valid[recordKey(r.Section, r.RecordID)] = r.Verdict == "valido"
	}
	result := Evaluation{Maximum: *def.MaximumTotal, Threshold: *def.Threshold, Breakdown: []Breakdown{}, Blockers: []string{}, TieValues: []Breakdown{}, ManualTie: true}
	for _, r := range review.Input.Requirements {
		if !r.Satisfied {
			result.Blockers = append(result.Blockers, "requisito_no_cumplido:"+r.Code)
		}
	}
	rawQuestions := map[string]string{}
	if q := form.Disqualifications; q != nil {
		for i, v := range []string{q.Q1, q.Q2, q.Q3, q.Q4, q.Q5, q.Q6, q.Q7, q.Q8, q.Q9, q.Q10, q.Q11, q.Q12, q.Q13} {
			rawQuestions[fmt.Sprintf("q%d", i+1)] = v
		}
	}
	confirmed := map[string]bool{}
	for _, q := range review.Input.Questions {
		confirmed[q.ID] = q.Confirmed
	}
	for _, q := range def.Questions {
		if !confirmed[q.ID] {
			result.Blockers = append(result.Blockers, "inhabilidad_sin_verificar:"+q.ID)
		} else if rawQuestions[q.ID] != q.AcceptedAnswer {
			result.Blockers = append(result.Blockers, "respuesta_no_admitida:"+q.ID)
		}
	}
	checklistApproved := len(result.Blockers) == 0
	byCode := map[string]Breakdown{}
	for _, criterion := range def.Criteria {
		units, err := metric(form, c.Snapshot, valid, criterion, def)
		if criterion.Metric == "requirements_approved" {
			units = 0
			if checklistApproved {
				units = 1
			}
		}
		if err != nil {
			return Evaluation{}, err
		}
		points := score(units, criterion, def.Rounding)
		entry := Breakdown{criterion.Code, units, points}
		result.Breakdown = append(result.Breakdown, entry)
		byCode[criterion.Code] = entry
		result.Total += points
		if criterion.Required && units == 0 {
			result.Blockers = append(result.Blockers, "evidencia_requerida:"+criterion.Code)
		}
	}
	if result.Total < result.Threshold {
		result.Blockers = append(result.Blockers, "puntaje_bajo_umbral")
	}
	for _, code := range def.TieBreakers {
		if code != "resolucion_manual" {
			result.TieValues = append(result.TieValues, byCode[code])
		}
	}
	result.CanAdvance = len(result.Blockers) == 0
	return result, nil
}
func metric(form *submission.Form, item *submission.Submission, valid map[string]bool, c Criterion, def RuleDefinition) (int64, error) {
	units := int64(0)
	switch c.Metric {
	case "requirements_approved":
		return 0, nil
	case "education_count":
		if form.Education != nil {
			for _, r := range form.Education.Degrees {
				if valid[recordKey(5, r.ID)] && (c.Filter == "" || c.Filter == r.Level) {
					units++
				}
			}
		}
	case "publication_count":
		if form.Publications != nil {
			for _, r := range form.Publications.Publications {
				if valid[recordKey(7, r.ID)] && (c.Filter == "" || c.Filter == r.Type) {
					units++
				}
			}
		}
	case "training_hours":
		if form.Training != nil {
			for _, r := range form.Training.Events {
				if valid[recordKey(8, r.ID)] && (c.Filter == "" || c.Filter == r.Type) {
					if r.Hours < 0 || r.Hours > 1_000_000 {
						return 0, ErrEvidence
					}
					units += int64(r.Hours)
				}
			}
		}
	case "experience_days":
		location, err := time.LoadLocation("America/Guayaquil")
		if err != nil {
			return 0, err
		}
		cutoff, err := time.Parse(time.DateOnly, item.SubmittedAt.In(location).Format(time.DateOnly))
		if err != nil {
			return 0, err
		}
		type interval struct{ start, end time.Time }
		intervals := []interval{}
		if form.Experience != nil {
			for _, r := range form.Experience.Experiences {
				if !valid[recordKey(6, r.ID)] || c.Filter != "" && c.Filter != r.Type {
					continue
				}
				start, err := time.Parse(time.DateOnly, r.StartDate)
				if err != nil {
					return 0, ErrEvidence
				}
				end := cutoff
				if !r.Current {
					end, err = time.Parse(time.DateOnly, r.EndDate)
				}
				if err != nil || end.Before(start) || end.After(cutoff) {
					return 0, ErrEvidence
				}
				if def.DayCounting == "dias_inclusivos" {
					end = end.AddDate(0, 0, 1)
				}
				intervals = append(intervals, interval{start, end})
			}
		}
		sort.Slice(intervals, func(i, j int) bool { return intervals[i].start.Before(intervals[j].start) })
		if def.ExperienceAggregation == "unir_intervalos" {
			merged := []interval{}
			for _, r := range intervals {
				if len(merged) == 0 || r.start.After(merged[len(merged)-1].end) {
					merged = append(merged, r)
				} else if r.end.After(merged[len(merged)-1].end) {
					merged[len(merged)-1].end = r.end
				}
			}
			intervals = merged
		}
		for _, r := range intervals {
			units += (r.end.Unix() - r.start.Unix()) / 86400
		}
	default:
		return 0, ErrInvalid
	}
	if units > 1_000_000_000 {
		return 0, ErrEvidence
	}
	return units, nil
}
