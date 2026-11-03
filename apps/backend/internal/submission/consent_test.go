package submission

import "testing"

func TestNamePublicationOnlyRecognizesAcceptedSealedLiteral(t *testing.T) {
	for _, name := range []string{"reconocido", "nil", "no_aceptado", "otra_version", "literal_distinto", "sin_fecha", "fecha_invalida", "sin_literales"} {
		t.Run(name, func(t *testing.T) {
			def, err := FormDefinition()
			if err != nil {
				t.Fatal(err)
			}
			c := def.Consent
			c.NamePublication = true
			c.AcceptedAt = "2026-10-31T15:00:00-05:00"
			switch name {
			case "nil":
				c = nil
			case "no_aceptado":
				c.NamePublication = false
			case "otra_version":
				c.Version = "no-reconocida"
			case "literal_distinto":
				c.Statements[3] = "Acepto todo"
			case "sin_fecha":
				c.AcceptedAt = ""
			case "fecha_invalida":
				c.AcceptedAt = "falso"
			case "sin_literales":
				c.Statements = nil
			}
			if got := AllowsNamePublication(c); got != (name == "reconocido") {
				t.Fatal(name, got)
			}
		})
	}
}
