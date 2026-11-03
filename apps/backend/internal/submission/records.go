package submission

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

/** sectionRecordIDs recoge identidades estables sin depender de índices de pantalla.
 * @returns {string[]} IDs de los registros declarados en una sección.
 */
func sectionRecordIDs(f *Form, section int) []string {
	if f == nil {
		return nil
	}
	ids := []string{}
	switch section {
	case 3:
		if f.Additional != nil {
			for _, r := range f.Additional.Relatives {
				ids = append(ids, r.ID)
			}
		}
	case 5:
		if f.Education != nil {
			for _, r := range f.Education.Degrees {
				ids = append(ids, r.ID)
			}
		}
	case 6:
		if f.Experience != nil {
			for _, r := range f.Experience.Experiences {
				ids = append(ids, r.ID)
			}
		}
	case 7:
		if f.Publications != nil {
			for _, r := range f.Publications.Publications {
				ids = append(ids, r.ID)
			}
		}
	case 8:
		if f.Training != nil {
			for _, r := range f.Training.Events {
				ids = append(ids, r.ID)
			}
		}
	}
	return ids
}

/** validateRecordIDs evita atribuir un documento a dos registros con el mismo ID.
 * @param {bool} full - Permite el ID pendiente en borradores; lo exige al enviar.
 * @returns {error} Primera identidad vacía, duplicada o no admisible.
 */
func validateRecordIDs(ids []string, full bool, path string) error {
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" {
			if full {
				return rule(path, "cada registro debe tener un identificador estable")
			}
			continue
		}
		if strings.TrimSpace(id) != id || utf8.RuneCountInString(id) > 255 || strings.IndexFunc(id, unicode.IsControl) >= 0 {
			return rule(path, "el identificador del registro no es válido")
		}
		if seen[id] {
			return rule(path, "no se admiten identificadores de registro duplicados")
		}
		seen[id] = true
	}
	return nil
}
