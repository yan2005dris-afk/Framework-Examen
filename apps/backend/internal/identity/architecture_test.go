package identity_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/**
 * TestDominioNoImportaLaCache vigila la frontera del contexto: el paquete de
 * dominio no puede depender de la infraestructura de caché. Los adaptadores
 * viven en internal/identity/redis y son el único puente permitido.
 */
func TestDominioNoImportaLaCache(t *testing.T) {
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entrada := range entradas {
		nombre := entrada.Name()
		if entrada.IsDir() || !strings.HasSuffix(nombre, ".go") {
			continue
		}
		ruta := filepath.Join(".", nombre)
		archivo, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("no se pudo analizar %s: %v", nombre, err)
		}
		// Las pruebas externas pueden construir adaptadores con la caché;
		// el paquete de dominio, no.
		if archivo.Name.Name != "identity" {
			continue
		}
		for _, importacion := range archivo.Imports {
			paquete := strings.Trim(importacion.Path.Value, `"`)
			if strings.Contains(paquete, "internal/platform/cache") {
				t.Errorf("%s importa la caché de infraestructura: %s", nombre, paquete)
			}
		}
	}
}
