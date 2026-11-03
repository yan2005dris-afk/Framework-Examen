package privacy

import (
	"testing"
	"time"
)

func qaRules() Rules {
	return Rules{Text: "Texto sintético, NO institucional", Treatment: "Información QA", Reference: "Acta QA", AccessDays: 30, DocumentDays: 10, ReportDays: 15, PublicDays: 5, EmailDays: 2, ReviewDays: 60, Historical: "revision_humana_sin_borrado_automatico", AllowExports: true, AllowEmail: true, AllowNames: true}
}
func TestPolicyRequiresExplicitDefinitions(t *testing.T) {
	for _, name := range []string{"text", "treatment", "reference", "retention", "disposal"} {
		t.Run(name, func(t *testing.T) {
			r := qaRules()
			switch name {
			case "text":
				r.Text = ""
			case "treatment":
				r.Treatment = ""
			case "reference":
				r.Reference = ""
			case "retention":
				r.ReviewDays = 0
			case "disposal":
				r.Historical = "borrar_todo"
			}
			if r.Validate() == nil {
				t.Fatal("inventó política")
			}
		})
	}
}
func TestPolicyChannelWindowsAndRestrictions(t *testing.T) {
	accepted := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	for _, channel := range []string{"datos", "documentos", "reportes", "nombres", "correo", "exportacion"} {
		t.Run(channel, func(t *testing.T) {
			r := qaRules()
			if r.Allow(accepted, accepted, channel) != nil {
				t.Fatal("rechazó apertura")
			}
			days := 30
			switch channel {
			case "documentos":
				days = 10
			case "reportes":
				days = 15
			case "nombres":
				days = 5
			case "correo":
				days = 2
			}
			if r.Allow(accepted, accepted.AddDate(0, 0, days), channel) == nil {
				t.Fatal("aceptó después de cierre")
			}
		})
	}
	r := qaRules()
	r.AllowNames = false
	if r.Allow(accepted, accepted, "nombres") == nil {
		t.Fatal("publicó sin permiso")
	}
	if r.Allow(accepted, accepted.Add(-time.Second), "datos") == nil {
		t.Fatal("aceptación futura")
	}
}
