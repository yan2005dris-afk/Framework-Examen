package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

const ddl = `CREATE TABLE IF NOT EXISTS catalogs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    kind VARCHAR(32) NOT NULL,
    parent_key VARCHAR(320) NOT NULL DEFAULT '',
    parent_id UUID REFERENCES catalogs(id) ON DELETE RESTRICT,
    value VARCHAR(160) NOT NULL CHECK (length(btrim(value)) > 0),
    position INTEGER NOT NULL CHECK (position >= 0),
    CONSTRAINT catalogs_value_unique UNIQUE (kind, parent_key, value),
    CONSTRAINT catalogs_kind_check CHECK (kind IN (
        'countries', 'provinces', 'cities', 'generos', 'estadosCiviles',
        'respuestas', 'parentescos', 'instituciones', 'sectores', 'niveles',
        'tiposInstitucion', 'areas', 'tiposExperiencia', 'tiposPublicacion', 'tiposEvento'
    )),
    CONSTRAINT catalogs_parent_check CHECK (
        (kind NOT IN ('provinces', 'cities') AND parent_key = '' AND parent_id IS NULL)
        OR (kind = 'provinces' AND parent_key = 'EC' AND parent_id IS NOT NULL)
        OR (kind = 'cities' AND parent_key LIKE 'EC/%' AND parent_id IS NOT NULL)
    )
);
CREATE INDEX IF NOT EXISTS catalogs_lookup_idx ON catalogs(kind, parent_key, position, value);

`

func literal(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }

/** generate produce una semilla determinista desde la definición y geografía de EF-12. */
func generate(definition submission.Definition) string {
	var sql strings.Builder
	sql.WriteString("-- Generado con go run ./cmd/catalog-seed; fuente: submission.FormDefinition().\n")
	sql.WriteString("-- PostgreSQL 18; 194 países RF, 24 provincias y 221 cantones del catálogo del proyecto.\n")
	sql.WriteString(ddl)
	rows := []string{}
	for index, name := range definition.Options["paises"] {
		rows = append(rows, fmt.Sprintf("    ('countries', '', NULL, %s, %d)", literal(name), index))
	}
	names := []string{}
	for name := range definition.Options {
		if name != "paises" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		for index, value := range definition.Options[name] {
			rows = append(rows, fmt.Sprintf("    (%s, '', NULL, %s, %d)", literal(name), literal(value), index))
		}
	}
	insert := func(rows []string) {
		sql.WriteString("INSERT INTO catalogs (kind, parent_key, parent_id, value, position) VALUES\n")
		sql.WriteString(strings.Join(rows, ",\n"))
		sql.WriteString("\nON CONFLICT (kind, parent_key, value) DO NOTHING;\n\n")
	}
	insert(rows)
	provinces := []string{}
	for province := range definition.Provinces {
		provinces = append(provinces, province)
	}
	sort.Strings(provinces)
	rows = nil
	for index, province := range provinces {
		rows = append(rows, fmt.Sprintf("    ('provinces', 'EC', (SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = 'Ecuador'), %s, %d)", literal(province), index))
	}
	insert(rows)
	rows = nil
	for _, province := range provinces {
		for index, city := range definition.Provinces[province] {
			rows = append(rows, fmt.Sprintf("    ('cities', %s, (SELECT id FROM catalogs WHERE kind = 'provinces' AND parent_key = 'EC' AND value = %s), %s, %d)", literal("EC/"+province), literal(province), literal(city), index))
		}
	}
	insert(rows)
	return sql.String()
}

/** main escribe los SQL de migración y entrega autónoma desde apps/backend. */
func main() {
	definition, err := submission.FormDefinition()
	if err != nil {
		fmt.Fprintln(os.Stderr, "No se pudo leer la definición EF-12.")
		os.Exit(1)
	}
	count := 0
	for _, cities := range definition.Provinces {
		count += len(cities)
	}
	if len(definition.Options["paises"]) != 194 || len(definition.Provinces) != 24 || count != 221 {
		fmt.Fprintln(os.Stderr, "Revise la cobertura de países, provincias y cantones antes de regenerar la semilla.")
		os.Exit(1)
	}
	if _, err := os.Stat("sql/migrations/migrations.go"); err != nil {
		fmt.Fprintln(os.Stderr, "Ejecute este comando desde apps/backend.")
		os.Exit(1)
	}
	body := generate(definition)
	files := map[string]string{
		"sql/migrations/00008_catalogs.sql": "-- +goose Up\n" + body + "-- +goose Down\nDROP TABLE catalogs;\n",
		"sql/seeds/catalogs.sql":            "-- Entrega autónoma: DDL y semilla idempotente; no requiere tablas de postulantes.\nBEGIN;\n" + body + "COMMIT;\n",
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
