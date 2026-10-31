package submission

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var referenceTime = time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)

func completeForm() *Form {
	yes := true
	return &Form{
		Personal:          &PersonalSection{FirstName: "Persona", LastName: "Prueba", IDCard: "1710034008", Gender: "Femenino", BirthDate: "1990-01-01", Nationality: "Ecuatoriana", MaritalStatus: "Soltero/a"},
		Contact:           &ContactSection{Country: "Ecuador", Province: "Pichincha", City: "Quito", Mobile: "0999999999", Email: "prueba@example.test", MainStreet: "Principal", Number: "S/N", SecondaryStreet: "Secundaria", Area: "Centro", Reference: "Parque", Phone: "022222222"},
		Additional:        &AdditionalSection{HasRelatives: "No", Relatives: []Relative{}},
		Position:          &PositionSection{Sector: "afiliados", Eligible: &yes, HasOrganization: "No"},
		Education:         &EducationSection{Degrees: []Degree{{Level: "Tercer Nivel de Grado", Institution: "Universidad de prueba", Type: "Nacional", Title: "Economista", Area: "Economía", Registry: "PRUEBA", RegistryDate: "2015-01-01"}}},
		Experience:        &ExperienceSection{Experiences: []Experience{{Type: "General", Position: "Analista", Institution: "Entidad de prueba", StartDate: "2020-01-01", EndDate: "2021-03-15", Activities: "Finanzas", Functions: "Funciones de prueba"}}},
		Publications:      &PublicationsSection{HasPublications: "No", Publications: []Publication{}},
		Training:          &TrainingSection{Events: []Training{}},
		Disqualifications: &DisqualificationsSection{Q1: "Sí", Q2: "No", Q3: "No", Q4: "No", Q5: "No", Q6: "No", Q7: "No", Q8: "No", Q9: "No", Q10: "No", Q11: "No", Q12: "No", Q13: "No"},
		Consent:           &Consent{DataProcessing: true, Truthfulness: true, CompetitionRules: true, NamePublication: true, PhysicalDelivery: true},
	}
}

func TestFormBusinessRules(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Form)
		valid  bool
	}{
		{"completo", func(f *Form) {}, true},
		{"extranjero", func(f *Form) { f.Personal.Nationality = "Extranjera" }, false},
		{"nacionalidad vacía", func(f *Form) { f.Personal.Nationality = "" }, false},
		{"cédula inválida", func(f *Form) { f.Personal.IDCard = "1710034009" }, false},
		{"género inválido", func(f *Form) { f.Personal.Gender = "Otro" }, false},
		{"fecha imposible", func(f *Form) { f.Personal.BirthDate = "2000-02-30" }, false},
		{"fecha futura", func(f *Form) { f.Personal.BirthDate = "2027-01-01" }, false},
		{"pareja obligatoria", func(f *Form) { f.Personal.MaritalStatus = "Casado/a" }, false},
		{"pareja completa", func(f *Form) {
			f.Personal.MaritalStatus = "Unión de hecho"
			f.Personal.SpouseName = "Pareja de prueba"
			f.Personal.SpouseIDCard = "1710034016"
		}, true},
		{"provincia inventada", func(f *Form) { f.Contact.Province = "Inexistente" }, false},
		{"cantón de otra provincia", func(f *Form) { f.Contact.City = "Guayaquil" }, false},
		{"sin país", func(f *Form) { f.Contact.Country = "" }, false},
		{"residencia extranjera", func(f *Form) {
			f.Contact.Country = "España"
			f.Contact.Province = "Madrid"
			f.Contact.City = "Alcalá de Henares"
		}, true},
		{"ubicación inválida", func(f *Form) { f.Contact.Country = "España"; f.Contact.City = "<script>" }, false},
		{"correo inválido", func(f *Form) { f.Contact.Email = "prueba" }, false},
		{"correo secundario opcional", func(f *Form) { f.Contact.SecondaryEmail = "" }, true},
		{"correo secundario inválido", func(f *Form) { f.Contact.SecondaryEmail = "invalido" }, false},
		{"teléfono inválido", func(f *Form) { f.Contact.Mobile = "texto" }, false},
		{"familiares faltantes", func(f *Form) { f.Additional.HasRelatives = "Sí" }, false},
		{"familiar incompleto", func(f *Form) {
			f.Additional.HasRelatives = "Sí"
			f.Additional.Relatives = []Relative{{Name: "Prueba"}}
		}, false},
		{"sector inválido", func(f *Form) { f.Position.Sector = "otro" }, false},
		{"elegibilidad sin comprobar", func(f *Form) { f.Position.Eligible = nil }, false},
		{"sector no elegible", func(f *Form) { no := false; f.Position.Eligible = &no }, false},
		{"organización sin nombre", func(f *Form) { f.Position.HasOrganization = "Sí" }, false},
		{"sin títulos", func(f *Form) { f.Education.Degrees = nil }, false},
		{"nivel académico inválido", func(f *Form) { f.Education.Degrees[0].Level = "Bachiller" }, false},
		{"área inválida", func(f *Form) { f.Education.Degrees[0].Area = "otra" }, false},
		{"registro incompleto", func(f *Form) { f.Education.Degrees[0].Registry = "" }, false},
		{"sin experiencia", func(f *Form) { f.Experience.Experiences = nil }, false},
		{"1000 caracteres unicode", func(f *Form) { f.Experience.Experiences[0].Functions = strings.Repeat("á", 1000) }, true},
		{"1001 caracteres", func(f *Form) { f.Experience.Experiences[0].Functions = strings.Repeat("á", 1001) }, false},
		{"rango invertido", func(f *Form) { f.Experience.Experiences[0].EndDate = "2019-12-31" }, false},
		{"empleo actual", func(f *Form) { f.Experience.Experiences[0].Current = true; f.Experience.Experiences[0].EndDate = "" }, true},
		{"fin obligatorio", func(f *Form) { f.Experience.Experiences[0].EndDate = "" }, false},
		{"publicaciones faltantes", func(f *Form) { f.Publications.HasPublications = "Sí" }, false},
		{"publicación inválida", func(f *Form) {
			f.Publications.HasPublications = "Sí"
			f.Publications.Publications = []Publication{{Title: "Prueba"}}
		}, false},
		{"horas negativas", func(f *Form) { f.Training.Events = []Training{{Hours: -1}} }, false},
		{"evento completo", func(f *Form) {
			f.Training.Events = []Training{{Type: "Curso", Name: "Curso de prueba", Institution: "Entidad", StartDate: "2020-01-01", EndDate: "2020-01-01", Hours: 1}}
		}, true},
		{"capacitación sin declaración", func(f *Form) { f.Training.Events = nil }, false},
		{"13 respuestas obligatorias", func(f *Form) { f.Disqualifications.Q13 = "" }, false},
		{"respuesta inválida", func(f *Form) { f.Disqualifications.Q5 = "Tal vez" }, false},
		{"q7 requiere detalles", func(f *Form) { f.Disqualifications.Q7 = "Sí" }, false},
		{"q7 detalles incompletos", func(f *Form) {
			f.Disqualifications.Q7 = "Sí"
			f.Disqualifications.Q7Details = &RemovalDetails{Institution: "Prueba"}
		}, false},
		{"q7 detalles completos", func(f *Form) {
			f.Disqualifications.Q7 = "Sí"
			f.Disqualifications.Q7Details = &RemovalDetails{Institution: "Prueba", Position: "Cargo", EndDate: "2025-01-01", Reason: "Motivo"}
		}, true},
		{"consentimiento ausente", func(f *Form) { f.Consent = nil }, false},
	}
	for i := 0; i < 5; i++ {
		tests = append(tests, struct {
			name   string
			change func(*Form)
			valid  bool
		}{fmt.Sprintf("aceptación %d", i+1), func(f *Form) {
			values := []*bool{&f.Consent.DataProcessing, &f.Consent.Truthfulness, &f.Consent.CompetitionRules, &f.Consent.NamePublication, &f.Consent.PhysicalDelivery}
			*values[i] = false
		}, false})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := completeForm()
			test.change(f)
			err := ValidateForm(f, true, referenceTime)
			if (err == nil) != test.valid {
				t.Fatalf("válido=%v, error=%v", test.valid, err)
			}
			if err != nil && !errors.Is(err, ErrBusinessRule) {
				t.Fatalf("error de negocio esperado: %v", err)
			}
		})
	}
}

func TestDraftAllowsMissingFieldsButRejectsInvalidValues(t *testing.T) {
	for _, content := range []string{`{}`, `{"seccion1":{"nombres":"Prueba"}}`, `{"seccion9":{"q1":"Sí"}}`} {
		if _, err := validateContent(json.RawMessage(content), false, referenceTime); err != nil {
			t.Fatalf("borrador válido: %v", err)
		}
	}
	for _, content := range []string{`{"seccion1":{"nacionalidad":"Extranjera"}}`, `{"seccion6":{"experiencias":[{"fechaInicio":"2025-05-05","fechaFin":"2025-01-01"}]}}`, `{"seccion2":{"pais":"Ecuador","provincia":"Guayas","canton":"Quito"}}`} {
		if _, err := validateContent(json.RawMessage(content), false, referenceTime); !errors.Is(err, ErrBusinessRule) {
			t.Fatalf("borrador inválido aceptado: %v", err)
		}
	}
}

func TestStrictFormDecoding(t *testing.T) {
	for _, content := range []string{`null`, `[]`, `true`, `{} {}`, `{"desconocida":{}}`, `{"seccion1":{"desconocido":true}}`, `{"seccion6":{"experiencias":[{"cargo":2}]}}`, `{"consentimiento":{"declaracionVeracidad":"true"}}`} {
		if _, err := DecodeForm(json.RawMessage(content)); !errors.Is(err, ErrInvalidContent) {
			t.Fatalf("estructura inválida aceptada: %s (%v)", content, err)
		}
	}
}

func TestAllSectionsAreRequired(t *testing.T) {
	raw, _ := json.Marshal(completeForm())
	for n := 1; n <= 9; n++ {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(raw, &fields)
			delete(fields, fmt.Sprintf("seccion%d", n))
			content, _ := json.Marshal(fields)
			form, err := DecodeForm(content)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateForm(form, true, referenceTime); !errors.Is(err, ErrBusinessRule) {
				t.Fatalf("sección ausente aceptada: %v", err)
			}
		})
	}
}

func TestSectorEligibilityMessages(t *testing.T) {
	for sector, reason := range map[string]string{"afiliados": "no mantiene la calidad de afiliado al IESS", "jubilados": "no acredita jubilación", "empleadores": "no es empleador"} {
		form := completeForm()
		form.Position.Sector = sector
		no := false
		form.Position.Eligible = &no
		if err := ValidateForm(form, false, referenceTime); err == nil || !strings.Contains(err.Error(), reason) {
			t.Fatalf("mensaje RF faltante: %v", err)
		}
	}
}

func TestCanonicalContentRecursesAndPreservesNumbers(t *testing.T) {
	first, _ := CanonicalContent(json.RawMessage(`{"b":[{"z":1,"a":{"y":2,"x":9007199254740993}}],"a":true}`))
	second, _ := CanonicalContent(json.RawMessage(`{"a":true,"b":[{"a":{"x":9007199254740993,"y":2},"z":1}]}`))
	if string(first) != string(second) || HashContent(first) != HashContent(second) {
		t.Fatal("hash sensible al orden de claves anidadas")
	}
	if !strings.Contains(string(first), "9007199254740993") {
		t.Fatal("pérdida de precisión numérica")
	}
}
