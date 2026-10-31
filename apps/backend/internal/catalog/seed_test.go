package catalog_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestSeedCountriesMatchFunctionalRequirementWithoutPageHeaders(t *testing.T) {
	raw := readFile(t, filepath.Join("..", "..", "..", "..", "docs", "requerimiento.txt"))
	match := regexp.MustCompile(`(?s)1\.2\.1 País: lista a elegir \((.*?)\),`).FindStringSubmatch(raw)
	if len(match) != 2 {
		t.Fatal("no se encontró la lista de países del RF")
	}
	text := regexp.MustCompile(`(?s)\f.*?PÁG: 4 de 9`).ReplaceAllString(match[1], "")
	text = strings.Join(strings.Fields(text), " ")
	names := strings.Split(text, ",")
	for index := range names {
		names[index] = strings.TrimSpace(names[index])
	}
	definition, err := submission.FormDefinition()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 194 || !reflect.DeepEqual(names, definition.Options["paises"]) {
		t.Fatal("los países no coinciden con el RF limpio")
	}
	seed := readFile(t, filepath.Join("..", "..", "sql", "seeds", "catalogs.sql"))
	rows := regexp.MustCompile(`\('countries', '', NULL, '([^']+)', [0-9]+\)`).FindAllStringSubmatch(seed, -1)
	seedNames := []string{}
	for _, row := range rows {
		seedNames = append(seedNames, row[1])
	}
	if !reflect.DeepEqual(seedNames, names) || !strings.Contains(seed, "'Bután'") || strings.Contains(seed, "DIRECCIÓN NACIONAL") {
		t.Fatal("semilla incompleta o con cabeceras como países")
	}
}

func TestSeedPreservesGeographyOptionsAndStandaloneDelivery(t *testing.T) {
	definition, err := submission.FormDefinition()
	if err != nil {
		t.Fatal(err)
	}
	seed := readFile(t, filepath.Join("..", "..", "sql", "seeds", "catalogs.sql"))
	up := readFile(t, filepath.Join("..", "..", "sql", "migrations", "00008_catalogs.sql"))
	up = strings.Split(strings.Split(up, "-- +goose Up\n")[1], "-- +goose Down")[0]
	body := strings.Split(strings.Split(seed, "BEGIN;\n")[1], "COMMIT;\n")[0]
	if up != body {
		t.Fatal("la entrega autónoma difiere de la migración")
	}
	if len(definition.Provinces) != 24 || strings.Count(seed, "    ('provinces',") != 24 || strings.Count(seed, "    ('cities',") != 221 {
		t.Fatal("se perdió cobertura geográfica")
	}
	for province, cities := range definition.Provinces {
		for _, city := range cities {
			if !strings.Contains(seed, "('cities', 'EC/"+province+"', (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = '"+province+"'), '"+city+"',") {
				t.Fatalf("se perdió la relación %s / %s", province, city)
			}
		}
	}
	for name, values := range definition.Options {
		if name == "paises" {
			continue
		}
		if strings.Count(seed, "    ('"+name+"',") != len(values) {
			t.Fatalf("catálogo incompleto: %s", name)
		}
		for _, value := range values {
			if !strings.Contains(seed, "('"+name+"', '', NULL, '"+value+"',") {
				t.Fatalf("opción perdida: %s", name)
			}
		}
	}
}
