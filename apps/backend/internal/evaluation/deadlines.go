package evaluation

import (
	"fmt"
	"slices"
	"time"
	_ "time/tzdata"
)

func (r AppealRules) Validate() error {
	if (r.Calendar.EndPolicy != "limitar_a_fase" && r.Calendar.EndPolicy != "exigir_termino_completo") || r.Calendar.TimeZone != "America/Guayaquil" || len(r.Calendar.Weekdays) > 7 || len(r.Calendar.Holidays) > 1000 {
		return ErrInvalid
	}
	seen := map[int]bool{}
	for _, d := range r.Calendar.Weekdays {
		if d < 0 || d > 6 || seen[d] {
			return ErrInvalid
		}
		seen[d] = true
	}
	holidays := map[string]bool{}
	for _, d := range r.Calendar.Holidays {
		if _, err := time.Parse(time.DateOnly, d); err != nil || holidays[d] {
			return ErrInvalid
		}
		holidays[d] = true
	}
	for _, t := range []Term{r.Applicant, r.Talent, r.Commission} {
		if t.Quantity < 1 {
			return ErrInvalid
		}
		switch t.Unit {
		case "minutos_transcurridos":
			if t.Quantity > 525600 || t.Cutoff != "" || t.IncludeAnchorDay {
				return ErrInvalid
			}
		case "dias_calendario", "dias_habiles":
			if t.Quantity > 366 {
				return ErrInvalid
			}
			if _, err := time.Parse("15:04", t.Cutoff); err != nil {
				return ErrInvalid
			}
			if t.Unit == "dias_habiles" && len(r.Calendar.Weekdays) == 0 {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
	}
	return nil
}

/** Deadline congela el cómputo por actor desde un instante del servidor; no inventa feriados ni plazos legales. */
func Deadline(anchor time.Time, term Term, calendar Calendar) (time.Time, error) {
	rules := AppealRules{Applicant: term, Talent: term, Commission: term, Calendar: calendar}
	if err := rules.Validate(); err != nil {
		return time.Time{}, err
	}
	if term.Unit == "minutos_transcurridos" {
		return anchor.Add(time.Duration(term.Quantity) * time.Minute), nil
	}
	location, err := time.LoadLocation(calendar.TimeZone)
	if err != nil {
		return time.Time{}, err
	}
	local := anchor.In(location)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	if !term.IncludeAnchorDay {
		day = day.AddDate(0, 0, 1)
	}
	count := 0
	for n := 0; n < 10000; n++ {
		eligible := term.Unit == "dias_calendario" || slices.Contains(calendar.Weekdays, int(day.Weekday())) && !slices.Contains(calendar.Holidays, day.Format(time.DateOnly))
		if eligible {
			count++
		}
		if count == term.Quantity {
			var hour, minute int
			if _, err = fmt.Sscanf(term.Cutoff, "%d:%d", &hour, &minute); err != nil {
				return time.Time{}, err
			}
			end := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, location)
			if !end.After(anchor) {
				return time.Time{}, ErrDeadline
			}
			return end.UTC(), nil
		}
		day = day.AddDate(0, 0, 1)
	}
	return time.Time{}, ErrInvalid
}
func boundedDeadline(anchor time.Time, term Term, calendar Calendar, end time.Time) (time.Time, error) {
	deadline, err := Deadline(anchor, term, calendar)
	if err != nil {
		return time.Time{}, err
	}
	if end.Before(deadline) {
		if calendar.EndPolicy == "exigir_termino_completo" {
			return time.Time{}, ErrDeadline
		}
		deadline = end
	}
	if !deadline.After(anchor) {
		return time.Time{}, ErrDeadline
	}
	return deadline, nil
}
func phaseWindow(c *Case, phase string) error {
	start, ok := c.PhaseStarts[phase]
	end := c.PhaseEnds[phase]
	if !ok || c.Now.Before(start) || !c.Now.Before(end) {
		return ErrPhase
	}
	return nil
}
