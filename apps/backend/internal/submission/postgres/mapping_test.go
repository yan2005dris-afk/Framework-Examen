package postgres

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

// sharedCases arma los formularios válidos de form_cases.json, la fuente
// compartida con la validación del frontend.
func sharedCases(t *testing.T) (time.Time, map[string]json.RawMessage) {
	t.Helper()
	raw, err := os.ReadFile("../testdata/form_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Now   time.Time       `json:"ahora"`
		Base  json.RawMessage `json:"base"`
		Cases []struct {
			Name    string          `json:"nombre"`
			Valid   bool            `json:"valido"`
			Draft   bool            `json:"borrador"`
			Content json.RawMessage `json:"contenido"`
			Delete  []string        `json:"borrar"`
			Patches []struct {
				Path  string          `json:"ruta"`
				Value json.RawMessage `json:"valor"`
			} `json:"parches"`
		} `json:"casos"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	cases := map[string]json.RawMessage{}
	for _, tc := range fixture.Cases {
		if !tc.Valid {
			continue
		}
		content := tc.Content
		if content == nil {
			content = fixture.Base
			if tc.Draft {
				content = json.RawMessage(`{}`)
			}
			for _, patch := range tc.Patches {
				content = patchJSON(t, content, strings.Split(patch.Path, "."), patch.Value, false)
			}
			for _, path := range tc.Delete {
				content = patchJSON(t, content, strings.Split(path, "."), nil, true)
			}
		}
		cases[tc.Name] = content
	}
	return fixture.Now, cases
}

// patchJSON aplica un parche de ruta con puntos, igual que la prueba de paridad del dominio.
func patchJSON(t *testing.T, raw json.RawMessage, path []string, value json.RawMessage, remove bool) json.RawMessage {
	t.Helper()
	if len(path) == 0 {
		return value
	}
	if len(raw) > 0 && raw[0] == '[' {
		var list []json.RawMessage
		if err := json.Unmarshal(raw, &list); err != nil {
			t.Fatal(err)
		}
		index, err := strconv.Atoi(path[0])
		if err != nil || index < 0 || index >= len(list) {
			t.Fatal("índice del caso compartido fuera de rango")
		}
		list[index] = patchJSON(t, list[index], path[1:], value, remove)
		result, _ := json.Marshal(list)
		return result
	}
	fields := map[string]json.RawMessage{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
	}
	if len(path) == 1 && remove {
		delete(fields, path[0])
	} else {
		fields[path[0]] = patchJSON(t, fields[path[0]], path[1:], value, remove)
	}
	result, _ := json.Marshal(fields)
	return result
}

// canonicalForm serializa el formulario con las reglas del sello.
func canonicalForm(t *testing.T, form *submission.Form) string {
	t.Helper()
	raw, err := json.Marshal(form)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := submission.CanonicalContent(raw)
	if err != nil {
		t.Fatal(err)
	}
	return string(canonical)
}

// expectedForm aplica a un formulario lo que el servicio fija en cada guardado,
// que es justo lo que la persistencia no guarda y recompone al leer.
func expectedForm(form *submission.Form, now time.Time) (*submission.Form, formIdentity) {
	identity := formIdentity{}
	if form.Personal != nil {
		form.Personal.CivilRegistryVerified = false
		identity.IDCard = form.Personal.IDCard
	}
	if form.Contact != nil {
		identity.Email = form.Contact.Email
	}
	if form.Position != nil {
		identity.Sector = form.Position.Sector
	}
	if form.Consent != nil {
		submission.ApplyConsentLiterals(form.Consent)
	}
	submission.NormalizeDurations(form, now)
	return form, identity
}

// TestFormRowsRoundTrip exige que descomponer y recomponer un formulario no
// pierda ni altere ningún dato: el hash del sello depende de estos bytes.
func TestFormRowsRoundTrip(t *testing.T) {
	now, cases := sharedCases(t)
	// Bordes del borrador que los casos compartidos no cubren.
	cases["listas null y vacías"] = json.RawMessage(`{"seccion3":{"tieneFamiliaresIessBiess":""},"seccion5":{"titulos":[]},"seccion6":{"experiencias":null},"seccion7":{"cuentaConPublicaciones":"Sí","publicaciones":null},"seccion8":{"capacitaciones":null}}`)
	cases["registros sin identificador"] = json.RawMessage(`{"seccion5":{"titulos":[{"id":"","nivel":""},{"id":"","nivel":"Tercer Nivel de Grado"}]},"seccion8":{"capacitaciones":[{"id":"","horas":0}]}}`)
	cases["remoción declarada y empleo vigente"] = json.RawMessage(`{"seccion6":{"experiencias":[{"id":"e1","tipo":"General","fechaInicio":"2020-01-01","actualmenteLaborando":true}]},"seccion9":{"q7":"Sí","q7Detalles":{"institucion":"SB","cargo":"","fechaFin":"2021-01-01","motivo":"Fin"}}}`)
	cases["q7 Sí sin detalle"] = json.RawMessage(`{"seccion9":{"q7":"Sí"}}`)
	cases["respaldos y consentimiento sellado"] = json.RawMessage(`{"seccion4":{"sector":"empleadores","verificacionIessCumplida":false,"cuentaConRespaldoOrganizacion":"Sí","nombreOrganizacion":"Cámara","documentoRespaldoPdf":{"id":"0193a1b2-0000-7000-8000-000000000001","sha256":"` + strings.Repeat("a", 64) + `","nombre":"r.pdf","tamano":"10","fechaSubida":"2026-10-30T10:00:00Z"}},"consentimiento":{"aceptadoEn":"2026-10-31T16:00:00Z","aceptacionTratamientoDatos":true,"firmaManuscritaDataUrl":"data:image/png;base64,AA==","fechaFirma":"2026-10-31"}}`)
	cases["sólo consentimiento"] = json.RawMessage(`{"consentimiento":{}}`)

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			form, err := submission.DecodeForm(content)
			if err != nil {
				t.Fatalf("caso inválido para DecodeForm: %v", err)
			}
			want, identity := expectedForm(form, now)
			expected := canonicalForm(t, want)

			original, _ := submission.DecodeForm(content)
			got := rowsToForm(formToRows(original), identity)
			submission.NormalizeDurations(got, now)
			if actual := canonicalForm(t, got); actual != expected {
				t.Fatalf("la ida y vuelta altera el formulario:\nesperado %s\nobtenido %s", expected, actual)
			}
		})
	}
	// 15 casos válidos compartidos y 6 bordes propios: detecta un archivo vaciado.
	if len(cases) < 21 {
		t.Fatalf("se esperaban al menos 21 casos, hay %d", len(cases))
	}
}

// TestFormRowsDropDerivedData comprueba que las filas no guardan datos derivables.
func TestFormRowsDropDerivedData(t *testing.T) {
	_, cases := sharedCases(t)
	form, err := submission.DecodeForm(cases["Formulario completo sin firma digital"])
	if err != nil {
		t.Fatal(err)
	}
	submission.NormalizeDurations(form, time.Now())
	submission.ApplyConsentLiterals(form.Consent)
	rows := formToRows(form)
	if rows.Personal.IDCard != "" || rows.Contact.Email != "" || rows.Position.Sector != "" {
		t.Fatal("cédula, correo y sector deben leerse de su fuente, no guardarse en las filas")
	}
	if rows.Experiences[0].Duration != nil {
		t.Fatal("la duración es un dato derivado")
	}
	if rows.Consent.Version != "" || rows.Consent.Statements != nil {
		t.Fatal("los literales del consentimiento son constantes del RF")
	}
	if form.Personal.IDCard == "" || form.Experience.Experiences[0].Duration == nil {
		t.Fatal("formToRows no debe modificar el formulario de origen")
	}
}
