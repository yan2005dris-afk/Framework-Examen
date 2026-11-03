// Command e2e-seed siembra el conjunto de datos de extremo a extremo.
//
// Es el contrato que comparten las dos suites de pruebas: la de interfaz
// (Playwright, contra el stack de desarrollo) y la de API (Go, contra un
// servidor de prueba). La siembra es idempotente, así que puede ejecutarse
// antes de cada corrida sin duplicar cuentas ni expedientes.
//
// El comando imprime el conjunto sembrado en JSON para que el runner de la
// suite lo consuma sin volver a consultar la base.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/e2edata"
)

/**
 * main aplica la siembra y publica el resultado.
 */
func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres@localhost:5432/biess?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fail("no se pudo abrir la base de pruebas: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fail("no se pudo conectar a la base de pruebas: %v", err)
	}

	dataset, err := e2edata.Seed(ctx, pool)
	if err != nil {
		fail("no se pudo sembrar el conjunto de pruebas: %v", err)
	}

	// La simulación de elegibilidad la carga el servidor al arrancar. Sin
	// ella, toda cuenta recibe `503 elegibilidad_no_disponible` y el
	// formulario EF-12 no se puede completar en el stack de desarrollo.
	rutaSimulacion := os.Getenv("E2E_ELIGIBILITY_FILE")
	if rutaSimulacion == "" {
		// El compose monta esta carpeta dentro del contenedor, de modo que la
		// API lea el mismo archivo que se escribe aquí.
		rutaSimulacion = "e2e-data/eligibility.json"
	}
	if err := e2edata.EscribirSimulacionElegibilidad(dataset, rutaSimulacion); err != nil {
		fail("no se pudo publicar la simulación de elegibilidad: %v", err)
	}

	raw, err := json.MarshalIndent(resumir(dataset), "", "  ")
	if err != nil {
		fail("no se pudo serializar el conjunto: %v", err)
	}
	fmt.Println(string(raw))
}

/**
 * resumen es la vista pública del conjunto sembrado: lo que las suites
 * necesitan saber sin depender de la estructura interna del paquete.
 */
type resumen struct {
	// Competition es la convocatoria sembrada.
	Competition string `json:"convocatoria"`
	// CompetitionCode es el código visible de la convocatoria.
	CompetitionCode string `json:"convocatoriaCodigo"`
	// Password es la contraseña compartida por las cuentas.
	Password string `json:"password"`
	// Accounts son las cuentas por rol.
	Accounts map[string]resumenCuenta `json:"cuentas"`
	// Submissions son los expedientes por escenario.
	Submissions map[string]resumenExpediente `json:"expedientes"`
	// PolicyID es la política de privacidad vigente.
	PolicyID string `json:"politicaId"`
	// PolicyHash es el hash de la política de privacidad.
	PolicyHash string `json:"politicaHash"`
}

/**
 * resumenCuenta describe una cuenta sembrada.
 */
type resumenCuenta struct {
	// Cedula es la clave de acceso de la cuenta.
	Cedula string `json:"cedula"`
	// Correo es el correo declarado de la cuenta.
	Correo string `json:"correo"`
	// Rol es el rol que el servidor reconoce.
	Rol string `json:"rol"`
}

/**
 * resumenExpediente describe un expediente sembrado.
 */
type resumenExpediente struct {
	// ID es el identificador del expediente.
	ID string `json:"id"`
	// Estado es el estado final del expediente.
	Estado string `json:"estado"`
	// Hash es el sello criptográfico del expediente enviado.
	Hash string `json:"hash"`
}

/**
 * resumir convierte el conjunto sembrado en su vista pública.
 *
 * @param {e2edata.Dataset} dataset - Conjunto sembrado.
 * @returns {resumen} Vista serializable.
 */
func resumir(dataset e2edata.Dataset) resumen {
	cuentas := map[string]resumenCuenta{}
	for clave, cuenta := range dataset.Accounts {
		cuentas[clave] = resumenCuenta{
			Cedula: cuenta.Cedula,
			Correo: cuenta.Correo,
			Rol:    cuenta.Rol,
		}
	}
	expedientes := map[string]resumenExpediente{}
	for clave, expediente := range dataset.Submissions {
		expedientes[clave] = resumenExpediente{
			ID:     expediente.ID.String(),
			Estado: expediente.Estado,
			Hash:   expediente.Hash,
		}
	}
	return resumen{
		Competition:     dataset.Competition.String(),
		CompetitionCode: e2edata.CompetitionCode,
		Password:        e2edata.Password,
		Accounts:        cuentas,
		Submissions:     expedientes,
		PolicyID:        dataset.PolicyID.String(),
		PolicyHash:      dataset.PolicyHash,
	}
}

/**
 * fail termina el proceso con un mensaje de error.
 *
 * @param {string} formato - Formato del mensaje.
 * @param {any} args - Valores del formato.
 */
func fail(formato string, args ...any) {
	fmt.Fprintf(os.Stderr, formato+"\n", args...)
	os.Exit(1)
}
