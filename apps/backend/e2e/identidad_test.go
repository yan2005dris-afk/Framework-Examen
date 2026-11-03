package e2e

import (
	"net/http"
	"strings"
	"testing"
)

/**
 * TestHealthzYReadyz comprueba que los healthchecks responden sin sesión.
 */
func TestHealthzYReadyz(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Get(t, "/healthz")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("healthz devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if respuesta.JSON()["status"] != "ok" {
		t.Fatalf("healthz no informa estado activo: %s", respuesta.Cuerpo)
	}

	respuesta = cliente.Get(t, "/readyz")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("readyz devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestCabeceraDeCorrelacion comprueba que la API devuelve el identificador
 * de petición y que conserva el que envía el cliente.
 */
func TestCabeceraDeCorrelacion(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Get(t, "/healthz")
	generado := respuesta.Headers.Get("X-Request-ID")
	if generado == "" {
		t.Fatal("la API no devolvió X-Request-ID")
	}

	propio := "6f3a1c22-9d0e-4a4b-8f5e-2c1d3e4f5a6b"
	respuesta = cliente.do(t, http.MethodGet, "/healthz", nil, map[string]string{"X-Request-ID": propio})
	if respuesta.Headers.Get("X-Request-ID") != propio {
		t.Fatalf("la API no conservó el identificador enviado: %s", respuesta.Headers.Get("X-Request-ID"))
	}
}

/**
 * TestRutaDesconocidaDevuelveEnvelope comprueba que el 404 usa el envelope
 * estándar de error con su código estable.
 */
func TestRutaDesconocidaDevuelveEnvelope(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Get(t, "/api/v1/no-existe")
	if respuesta.Status != http.StatusNotFound {
		t.Fatalf("una ruta inexistente devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if codigo := respuesta.CodigoError(); codigo != "ruta_no_encontrada" {
		t.Fatalf("el código de error inesperado: %q", codigo)
	}
}

/**
 * TestSinSesionResponde401 comprueba que las rutas protegidas rechazan la
 * petición sin cookie de sesión.
 */
func TestSinSesionResponde401(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Get(t, "/api/v1/submissions/")
	if respuesta.Status != http.StatusUnauthorized {
		t.Fatalf("sin sesión la API devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if codigo := respuesta.CodigoError(); codigo != "no_autenticado" {
		t.Fatalf("el código de error inesperado: %q", codigo)
	}
}

/**
 * TestCSRFRechazaOrigenNoPermitido comprueba que una escritura con un origen
 * no permitido es rechazada antes de tocar el dominio.
 */
func TestCSRFRechazaOrigenNoPermitido(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.do(t, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"cedula": "2406425922", "password": "TestPassword.2026*"},
		map[string]string{"Origin": "http://sitio-malicioso.local"})
	if respuesta.Status != http.StatusForbidden {
		t.Fatalf("un origen no permitido obtuvo %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if codigo := respuesta.CodigoError(); codigo != "origen_no_permitido" {
		t.Fatalf("el código de error inesperado: %q", codigo)
	}
}

/**
 * TestLoginExitosoEstableceCookie comprueba que el login devuelve el
 * postulante y establece la cookie de sesión HttpOnly.
 */
func TestLoginExitosoEstableceCookie(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Post(t, "/api/v1/auth/login", map[string]string{
		"cedula":   "2406425922",
		"password": "TestPassword.2026*",
	})
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el login devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}

	postulante, ok := respuesta.JSON()["postulante"].(map[string]any)
	if !ok {
		t.Fatalf("la respuesta no trae el postulante: %s", respuesta.Cuerpo)
	}
	if postulante["cedula"] != "2406425922" {
		t.Fatalf("la respuesta trae otra cédula: %v", postulante["cedula"])
	}
	if postulante["rol"] != "postulante" {
		t.Fatalf("la respuesta no declara el rol: %v", postulante["rol"])
	}

	cookies := respuesta.Headers.Values("Set-Cookie")
	encontroSesion := false
	for _, cookie := range cookies {
		if len(cookie) > 7 && cookie[:7] == "session" {
			encontroSesion = true
			if !strings.Contains(cookie, "HttpOnly") {
				t.Fatalf("la cookie de sesión no es HttpOnly: %s", cookie)
			}
		}
	}
	if !encontroSesion {
		t.Fatalf("el login no estableció la cookie de sesión: %v", cookies)
	}
}

/**
 * TestCredencialesInvalidasResponde401 comprueba que una contraseña
 * incorrecta no abre sesión.
 */
func TestCredencialesInvalidasResponde401(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Post(t, "/api/v1/auth/login", map[string]string{
		"cedula":   "2406425922",
		"password": "ContrasenaIncorrecta.2026*",
	})
	if respuesta.Status != http.StatusUnauthorized {
		t.Fatalf("una contraseña incorrecta devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if codigo := respuesta.CodigoError(); codigo != "credenciales_invalidas" {
		t.Fatalf("el código de error inesperado: %q", codigo)
	}
}

/**
 * TestSesionEntregadaDevuelvePerfil comprueba que la sesión creada sirve
 * para consultar el perfil y para cerrar sesión.
 */
func TestSesionEntregadaDevuelvePerfil(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.ClienteConSesionDe(t, "2406425922")

	respuesta := cliente.Get(t, "/api/v1/auth/session")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la sesión devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	postulante, ok := respuesta.JSON()["postulante"].(map[string]any)
	if !ok || postulante["cedula"] != "2406425922" {
		t.Fatalf("el perfil no corresponde a la cuenta: %s", respuesta.Cuerpo)
	}

	respuesta = cliente.Post(t, "/api/v1/auth/logout", nil)
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el cierre de sesión devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}

	respuesta = cliente.Get(t, "/api/v1/auth/session")
	if respuesta.Status != http.StatusUnauthorized {
		t.Fatalf("la sesión sobrevivió al cierre: %d %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestBloqueoTrasIntentosFallidos comprueba que la cuenta se bloquea tras
 * agotar los intentos permitidos y que el servidor responde 429.
 */
func TestBloqueoTrasIntentosFallidos(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	// El bloqueo anterior debe quedar limpio: esta prueba es la que lo provoca.
	if err := entorno.ReiniciarIntentos("2402740035"); err != nil {
		t.Fatalf("no se pudo limpiar el estado de intentos: %v", err)
	}

	for intento := 1; intento <= 3; intento++ {
		respuesta := cliente.Post(t, "/api/v1/auth/login", map[string]string{
			"cedula":   "2402740035",
			"password": "ContrasenaIncorrecta.2026*",
		})
		// El tercer fallo agota los intentos permitidos: la cuenta queda
		// bloqueada en ese momento y el servidor lo dice con 429.
		esperado := http.StatusUnauthorized
		if intento == 3 {
			esperado = http.StatusTooManyRequests
		}
		if respuesta.Status != esperado {
			t.Fatalf("el intento %d devolvió %d (se esperaba %d): %s", intento, respuesta.Status, esperado, respuesta.Cuerpo)
		}
	}

	// Con la cuenta bloqueada, ni la contraseña correcta ni la incorrecta
	// abren sesión: el bloqueo manda sobre la verificación.
	respuesta := cliente.Post(t, "/api/v1/auth/login", map[string]string{
		"cedula":   "2402740035",
		"password": "TestPassword.2026*",
	})
	if respuesta.Status != http.StatusTooManyRequests {
		t.Fatalf("una cuenta bloqueada devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	if codigo := respuesta.CodigoError(); codigo != "intentos_excedidos" {
		t.Fatalf("el código de error inesperado: %q", codigo)
	}
	if respuesta.Headers.Get("Retry-After") == "" {
		t.Fatal("la respuesta de bloqueo no informa el tiempo de espera")
	}
}
