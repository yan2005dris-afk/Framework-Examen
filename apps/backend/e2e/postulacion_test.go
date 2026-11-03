package e2e

import (
	"net/http"
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/e2edata"
)

/**
 * TestGuardadoParcialPorSeccion comprueba que una sección se guarda y queda
 * disponible al releer el expediente.
 */
func TestGuardadoParcialPorSeccion(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2402740035")

	expediente := entorno.Dataset.SubmissionByScenario("minimo")
	respuesta := cliente.Post(t, "/api/v1/submissions/"+expediente.ID.String()+"/sections/1", map[string]any{
		"nombres":         "SINTETICO",
		"apellidos":       e2edata.Mark,
		"cedula":          "2402740035",
		"genero":          "Masculino",
		"fechaNacimiento": "1985-05-20",
		"nacionalidad":    "Ecuatoriana",
		"estadoCivil":     "Soltero/a",
	})
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el guardado devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}

	respuesta = cliente.Get(t, "/api/v1/submissions/"+expediente.ID.String())
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la lectura devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	contenido, ok := respuesta.JSON()["content"].(map[string]any)
	if !ok {
		t.Fatalf("la respuesta no trae el contenido: %s", respuesta.Cuerpo)
	}
	seccion1, ok := contenido["seccion1"].(map[string]any)
	if !ok {
		t.Fatalf("el expediente no tiene la sección guardada: %s", respuesta.Cuerpo)
	}
	if seccion1["nacionalidad"] != "Ecuatoriana" {
		t.Fatalf("la sección guardada no conserva los datos: %v", seccion1)
	}
}

/**
 * TestNacionalidadNoEcuatorianaSeRechaza comprueba la regla EF-12: sólo la
 * nacionalidad ecuatoriana permite continuar.
 */
func TestNacionalidadNoEcuatorianaSeRechaza(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2402740035")

	expediente := entorno.Dataset.SubmissionByScenario("minimo")
	respuesta := cliente.Post(t, "/api/v1/submissions/"+expediente.ID.String()+"/sections/1", map[string]any{
		"nombres":         "SINTETICO",
		"apellidos":       e2edata.Mark,
		"cedula":          "2402740035",
		"genero":          "Masculino",
		"fechaNacimiento": "1985-05-20",
		"nacionalidad":    "Extranjera",
		"estadoCivil":     "Soltero/a",
	})
	if respuesta.Status == http.StatusOK {
		t.Fatalf("el servidor aceptó una nacionalidad no ecuatoriana: %s", respuesta.Cuerpo)
	}
	if respuesta.CodigoError() == "" {
		t.Fatalf("el rechazo no usa el envelope de error: %s", respuesta.Cuerpo)
	}
}

/**
 * TestExpedienteEnviadoQuedaSellado comprueba que el envío sella el
 * expediente con un hash criptográfico y lo vuelve inmutable.
 */
func TestExpedienteEnviadoQuedaSellado(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	expediente := entorno.Dataset.SubmissionByScenario("aprobado")
	respuesta := cliente.Get(t, "/api/v1/submissions/"+expediente.ID.String())
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la lectura devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	cuerpo := respuesta.JSON()
	if cuerpo["status"] != "submitted" {
		t.Fatalf("el expediente no está enviado: %v", cuerpo["status"])
	}
	sello, _ := cuerpo["hash"].(string)
	if len(sello) != 64 {
		t.Fatalf("el sello no es un hash SHA-256: %q", sello)
	}
	if expediente.Hash != "" && expediente.Hash != sello {
		t.Fatalf("el sello sembrado no coincide con el que expone la API: %q vs %q", expediente.Hash, sello)
	}
	if cuerpo["submittedAt"] == nil {
		t.Fatalf("el expediente enviado no tiene fecha de envío: %s", respuesta.Cuerpo)
	}
}

/**
 * TestExpedienteEnviadoNoAdmiteCambios comprueba que el sellado congela el
 * contenido: una sección de un expediente enviado se rechaza.
 */
func TestExpedienteEnviadoNoAdmiteCambios(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	expediente := entorno.Dataset.SubmissionByScenario("aprobado")
	respuesta := cliente.Post(t, "/api/v1/submissions/"+expediente.ID.String()+"/sections/1", map[string]any{
		"nombres":      "MODIFICADO",
		"nacionalidad": "Ecuatoriana",
	})
	if respuesta.Status == http.StatusOK {
		t.Fatalf("el servidor aceptó modificar un expediente enviado: %s", respuesta.Cuerpo)
	}
}

/**
 * TestImpresionDevuelvePDFSellado comprueba que la hoja imprimible del
 * expediente enviado es un PDF real.
 */
func TestImpresionDevuelvePDFSellado(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	expediente := entorno.Dataset.SubmissionByScenario("aprobado")
	respuesta := cliente.Get(t, "/api/v1/submissions/"+expediente.ID.String()+"/print.pdf")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la impresión devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if len(respuesta.Cuerpo) < 4 || string(respuesta.Cuerpo[:4]) != "%PDF" {
		t.Fatalf("la impresión no devolvió un PDF: %q", respuesta.Cuerpo[:min(16, len(respuesta.Cuerpo))])
	}
}

/**
 * TestBorradorEsVisibleParaSuTitular comprueba que el borrador se puede leer
 * y que el progreso se calcula por secciones completas.
 */
func TestBorradorEsVisibleParaSuTitular(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2402740035")

	expediente := entorno.Dataset.SubmissionByScenario("minimo")
	respuesta := cliente.Get(t, "/api/v1/submissions/"+expediente.ID.String())
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la lectura devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	progreso, ok := respuesta.JSON()["progress"].(map[string]any)
	if !ok {
		t.Fatalf("la respuesta no trae el progreso: %s", respuesta.Cuerpo)
	}
	if total, ok := progreso["total"].(float64); !ok || int(total) != 9 {
		t.Fatalf("el progreso no cuenta las nueve secciones: %v", progreso["total"])
	}
}

/**
 * TestHistorialDeEventosRegistraLaAccion comprueba que la auditoría deja
 * rastro de los accesos administrativos.
 */
func TestHistorialDeEventosRegistraLaAccion(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesion(t, "analista")

	expediente := entorno.Dataset.SubmissionByScenario("aprobado")
	respuesta := cliente.Get(t, "/api/v1/admin/submissions/"+expediente.ID.String()+"/events")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el historial devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if respuesta.JSON()["items"] == nil {
		t.Fatalf("el historial no trae los eventos: %s", respuesta.Cuerpo)
	}
}

/**
 * TestConvocatoriaPublicaVisibleSinSesion comprueba que la convocatoria
 * publicada se puede consultar sin sesión.
 */
func TestConvocatoriaPublicaVisibleSinSesion(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Get(t, "/api/v1/competitions")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el listado público devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestPoliticaDePrivacidadPublicaVisible comprueba que la política vigente
 * de la convocatoria se publica sin sesión.
 */
func TestPoliticaDePrivacidadPublicaVisible(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Get(t, "/api/v1/public/privacy/competitions/"+entorno.Dataset.Competition.String())
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la política pública devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
}

// min devuelve el menor de dos números.
func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
