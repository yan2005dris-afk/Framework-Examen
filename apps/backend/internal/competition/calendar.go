package competition

import (
	"github.com/google/uuid"
	"strings"
	"time"
)

const TimeZone = "America/Guayaquil"

/** Validate aplica la propuesta de calendario [apertura,cierre) sin imponer fases o duraciones ficticias. */
func (c *Competition) Validate() error {
	// PostgreSQL conserva microsegundos: normalizar antes de comparar evita rechazos por redondeo.
	c.OpensAt = c.OpensAt.UTC().Truncate(time.Microsecond)
	c.ClosesAt = c.ClosesAt.UTC().Truncate(time.Microsecond)
	for i := range c.Phases {
		c.Phases[i].OpensAt = c.Phases[i].OpensAt.UTC().Truncate(time.Microsecond)
		c.Phases[i].ClosesAt = c.Phases[i].ClosesAt.UTC().Truncate(time.Microsecond)
	}
	if c.ID == uuid.Nil || strings.TrimSpace(c.Code) == "" || len(c.Code) > 64 || strings.TrimSpace(c.Title) == "" || len(c.Title) > 200 || c.OpensAt.IsZero() || !c.ClosesAt.After(c.OpensAt) || len(c.Sectors) == 0 || len(c.Sectors) > 3 || len(c.Phases) > 12 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, s := range c.Sectors {
		if (s != "afiliados" && s != "jubilados" && s != "empleadores") || seen[s] {
			return ErrInvalid
		}
		seen[s] = true
	}
	seen = map[string]bool{}
	form := false
	for _, p := range c.Phases {
		if !validPhase(p.Code) || seen[p.Code] || strings.TrimSpace(p.Title) == "" || len(p.Title) > 200 || p.OpensAt.Before(c.OpensAt) || p.ClosesAt.After(c.ClosesAt) || !p.ClosesAt.After(p.OpensAt) || (p.Enabled && !p.Published) {
			return ErrInvalid
		}
		seen[p.Code] = true
		if p.Code == "postulacion" && p.Published && p.Enabled {
			form = true
		}
	}
	if c.Published && !c.Approved || c.FormEnabled && (!c.Approved || !c.Published || !form) {
		return ErrInvalid
	}
	return nil
}
func validPhase(code string) bool {
	switch code {
	case "postulacion", "meritos", "oposicion_teorica", "oposicion_psicotecnica", "reconsideracion", "resultados":
		return true
	}
	return false
}

/** Writable usa instantes absolutos; la zona declarada sirve para presentación, no para reinterpretar fechas. */
func (c Competition) Writable(now time.Time, sector string) bool {
	if !c.Approved || !c.Published || !c.FormEnabled || now.Before(c.OpensAt) || !now.Before(c.ClosesAt) {
		return false
	}
	if sector != "" {
		found := false
		for _, s := range c.Sectors {
			found = found || s == sector
		}
		if !found {
			return false
		}
	}
	for _, p := range c.Phases {
		if p.Code == "postulacion" && p.Published && p.Enabled && !now.Before(p.OpensAt) && now.Before(p.ClosesAt) {
			return true
		}
	}
	return false
}
