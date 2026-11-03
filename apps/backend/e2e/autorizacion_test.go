package e2e

import (
	"net/http"
	"testing"
)

/**
 * TestPostulanteNoAccedeAlPanelAdministrativo comprueba que un postulante
 * recibe 403 en las rutas administrativas y que la respuesta explica el
 * motivo con un código estable.
 */
func TestPostulanteNoAccedeAlPanelAdministrativo(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	respuesta := cliente.Get(t, "/api/v1/admin/competitions")
	if respuesta.Status != http.StatusForbidden {
		t.Fatalf("un postulante obtuvo %d en el panel administrativo: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if codigo := respuesta.CodigoError(); codigo != "acceso_denegado" {
		t.Fatalf("el código de error inesperado: %q", codigo)
	}
}

/**
 * TestPostulanteNoCambiaElRolDeOtroComprueba que el rol es de账户 exclusiva
 * de la administración: un postulante no puede ascender a administrador.
 */
func TestPostulanteNoCambiaElRolDeOtro(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	objetivo := entorno.Dataset.AccountByRole("analista")
	respuesta := cliente.Patch(t, "/api/v1/admin/applicants/"+objetivo.ID.String()+"/role", map[string]string{
		"rol":    "administrador",
		"motivo": "intento no autorizado",
	})
	if respuesta.Status != http.StatusForbidden {
		t.Fatalf("un postulante obtuvo %d al cambiar un rol: %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestAnalistaAccedeAlPanelYNoAAdministracion comprueba la política de roles:
 * el analista consulta, pero no gestiona la estructura del concurso.
 */
func TestAnalistaAccedeAlPanelYNoAAdministracion(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesion(t, "analista")

	respuesta := cliente.Get(t, "/api/v1/admin/competitions")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el analista obtuvo %d al listar convocatorias: %s", respuesta.Status, respuesta.Cuerpo)
	}

	// El banco de preguntas sólo lo gestiona la comisión de selección.
	respuesta = cliente.Get(t, "/api/v1/admin/question-bank/competitions/"+entorno.Dataset.Competition.String()+"/sectors/afiliados/versions")
	if respuesta.Status != http.StatusForbidden {
		t.Fatalf("el analista obtuvo %d al consultar el banco de preguntas: %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestAdministradorAccedeAlPanel comprueba que el administrador llega a las
 * rutas de gestión.
 */
func TestAdministradorAccedeAlPanel(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesion(t, "administrador")

	respuesta := cliente.Get(t, "/api/v1/admin/competitions")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el administrador obtuvo %d al listar convocatorias: %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestCatalogosPublicosDeCatalogo comprueba que los catálogos responden con
 * las listas esperadas: países, provincias, cantones y opciones.
 */
func TestCatalogosDeCatalogo(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	respuesta := cliente.Get(t, "/api/v1/catalogs/countries")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el catálogo de países devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	lista, ok := respuesta.JSON()["paises"].([]any)
	if !ok || len(lista) == 0 {
		t.Fatalf("el catálogo de países vino vacío: %s", respuesta.Cuerpo)
	}

	// La cascada país → provincia → cantón es el contrato EF-12.
	respuesta = cliente.Get(t, "/api/v1/catalogs/provinces/EC")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el catálogo de provincias devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	listaProvincias, ok := respuesta.JSON()["provincias"].([]any)
	if !ok || len(listaProvincias) != 24 {
		t.Fatalf("Ecuador debe tener 24 provincias, llegaron %d: %s", len(listaProvincias), respuesta.Cuerpo)
	}

	// Un país que no es Ecuador no tiene provincias. El catálogo identifica
	// los países por su nombre; "EC" es el único alias de código admitido.
	respuesta = cliente.Get(t, "/api/v1/catalogs/provinces/Colombia")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el catálogo de provincias devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	listaNoEC, ok := respuesta.JSON()["provincias"].([]any)
	if !ok || len(listaNoEC) != 0 {
		t.Fatalf("un país sin provincias devolvió %d: %s", len(listaNoEC), respuesta.Cuerpo)
	}

	// Un país que no existe en el catálogo es un filtro inválido, no una
	// lista vacía: el servidor no adivina la geografía.
	respuesta = cliente.Get(t, "/api/v1/catalogs/provinces/CO")
	if respuesta.Status != http.StatusBadRequest {
		t.Fatalf("un país desconocido devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if codigo := respuesta.CodigoError(); codigo != "filtro_catalogo_invalido" {
		t.Fatalf("el código de error inesperado: %q", codigo)
	}
}

/**
 * TestCatalogoRequiereSesion comprueba que los catálogos no son públicos.
 */
func TestCatalogoRequiereSesion(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Get(t, "/api/v1/catalogs/countries")
	if respuesta.Status != http.StatusUnauthorized {
		t.Fatalf("el catálogo sin sesión devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestPostulanteSoloVeSusExpedientes comprueba la titularidad: un
 * postulante no puede leer el expediente de otro.
 */
func TestPostulanteSoloVeSusExpedientes(t *testing.T) {
	entorno := Preparar(t)
	// La cuenta es explícita: hay varios postulantes sembrados y el
	// expediente ajeno debe pertenecer a otro titular.
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	ajeno := entorno.Dataset.SubmissionByScenario("reconsideracion")
	respuesta := cliente.Get(t, "/api/v1/submissions/"+ajeno.ID.String())
	if respuesta.Status != http.StatusForbidden && respuesta.Status != http.StatusNotFound {
		t.Fatalf("un postulante leyó el expediente ajeno con %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestListadoDeExpedientesUsaPaginacionPorOffset comprueba que el listado
 * acotado pagina con page y pageSize y que devuelve el total.
 */
func TestListadoDeExpedientesUsaPaginacionPorOffset(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	respuesta := cliente.Get(t, "/api/v1/submissions/?page=1&pageSize=10")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el listado devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	cuerpo := respuesta.JSON()
	if cuerpo["page"] == nil {
		t.Fatalf("el listado no informa la página: %s", respuesta.Cuerpo)
	}
	if cuerpo["total"] == nil {
		t.Fatalf("el listado no informa el total: %s", respuesta.Cuerpo)
	}
}

/**
 * TestPaginacionPorCursorCuandoLlegaCursor comprueba que la estrategia de
 * paginación cambia a keyset cuando el cliente envía limit: la respuesta usa
 * la forma de cursor y no la de página.
 */
func TestPaginacionPorCursorCuandoLlegaCursor(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	respuesta := cliente.Get(t, "/api/v1/submissions/?limit=10")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el listado devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	cuerpo := respuesta.JSON()
	if cuerpo["items"] == nil {
		t.Fatalf("el listado por cursor no trae los elementos: %s", respuesta.Cuerpo)
	}
	if cuerpo["limit"] == nil {
		t.Fatalf("el listado por cursor no informa el límite: %s", respuesta.Cuerpo)
	}
	// El modo cursor no pagina por página: no debe traer el total.
	if cuerpo["page"] != nil {
		t.Fatalf("el listado por cursor trae paginación por offset: %s", respuesta.Cuerpo)
	}
	if _, ok := cuerpo["hasMore"].(bool); !ok {
		t.Fatalf("el listado por cursor no informa si hay más: %s", respuesta.Cuerpo)
	}
}
