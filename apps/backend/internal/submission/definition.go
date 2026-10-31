package submission

import (
	_ "embed"
	"encoding/json"
)

//go:embed definition.json
var definitionJSON []byte

/** Definition agrupa los catálogos y campos del EF-12 compartidos con la SPA. */
type Definition struct {
	Version   string                                `json:"version"`
	Options   map[string][]string                   `json:"opciones"`
	Sections  []SectionDefinition                   `json:"secciones"`
	Questions []Question                            `json:"preguntasInhabilidades"`
	Limits    map[string]int64                      `json:"limites"`
	Identity  json.RawMessage                       `json:"identidad"`
	Types     map[string]map[string]FieldDefinition `json:"tipos"`
	Provinces map[string][]string                   `json:"provincias"`
	Consent   *Consent                              `json:"consentimiento"`
}

/** FieldDefinition describe tipos JSON para rechazar campos inválidos también en la SPA. */
type FieldDefinition struct {
	Type      string `json:"tipo"`
	Structure string `json:"estructura,omitempty"`
	Nullable  bool   `json:"anulable,omitempty"`
}

/** SectionDefinition distingue requisitos de envío y campos condicionales del borrador. */
type SectionDefinition struct {
	Number         int      `json:"numero"`
	Title          string   `json:"titulo"`
	Required       []string `json:"obligatorios,omitempty"`
	Optional       []string `json:"opcionales,omitempty"`
	List           string   `json:"lista,omitempty"`
	RecordRequired []string `json:"registroObligatorios,omitempty"`
	Conditions     []string `json:"condiciones,omitempty"`
}

/** Question conserva el ID y el texto literal de cada declaración de inhabilidad. */
type Question struct {
	ID   string `json:"id"`
	Text string `json:"texto"`
}

var formDefinition, definitionError = loadDefinition()

func loadDefinition() (Definition, error) {
	var definition Definition
	err := json.Unmarshal(definitionJSON, &definition)
	return definition, err
}

/** FormDefinition devuelve una copia de catálogos y consentimiento sin aceptaciones.
 * @returns {Definition, error} Contrato de captura; no implica certificación de identidad.
 */
func FormDefinition() (Definition, error) {
	definition, err := loadDefinition()
	if err != nil {
		return Definition{}, err
	}
	definition.Provinces = make(map[string][]string, len(ecuadorProvinces))
	for province, cities := range ecuadorProvinces {
		definition.Provinces[province] = append([]string(nil), cities...)
	}
	definition.Consent = &Consent{Version: consentVersion, Statements: append([]string(nil), consentStatements...)}
	return definition, nil
}
