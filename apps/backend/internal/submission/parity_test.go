package submission

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// Los mismos casos los consume validar-formulario.test.ts; no requieren servicios.
func TestSharedFormValidationCases(t *testing.T) {
	raw, err := os.ReadFile("testdata/form_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Now   time.Time       `json:"ahora"`
		Base  json.RawMessage `json:"base"`
		Cases []struct {
			Name    string          `json:"nombre"`
			Full    bool            `json:"completo"`
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
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			content := tc.Content
			if content == nil {
				content = fixture.Base
				if tc.Draft {
					content = json.RawMessage(`{}`)
				}
				for _, patch := range tc.Patches {
					content = patchFixture(t, content, strings.Split(patch.Path, "."), patch.Value, false)
				}
				for _, path := range tc.Delete {
					content = patchFixture(t, content, strings.Split(path, "."), nil, true)
				}
			}
			form, err := DecodeForm(content)
			if err == nil {
				err = ValidateForm(form, tc.Full, fixture.Now)
			}
			if (err == nil) != tc.Valid {
				t.Fatalf("válido esperado=%v, error=%v", tc.Valid, err)
			}
		})
	}
}

func patchFixture(t *testing.T, raw json.RawMessage, path []string, value json.RawMessage, remove bool) json.RawMessage {
	t.Helper()
	if len(path) == 0 {
		return value
	}
	if len(raw) > 0 && raw[0] == '[' {
		var list []json.RawMessage
		if err := json.Unmarshal(raw, &list); err != nil {
			t.Fatal(err)
		}
		index := 0
		for _, digit := range path[0] {
			index = index*10 + int(digit-'0')
		}
		if index < 0 || index >= len(list) {
			t.Fatal("índice del caso compartido fuera de rango")
		}
		list[index] = patchFixture(t, list[index], path[1:], value, remove)
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
		fields[path[0]] = patchFixture(t, fields[path[0]], path[1:], value, remove)
	}
	result, _ := json.Marshal(fields)
	return result
}
