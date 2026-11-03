package e2e

import (
	"net/http"
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/e2edata"
)

/**
 * TestRegistroConOTPReal comprueba el flujo completo de registro: se pide un
 * código, el servidor lo envía por correo, y la verificación crea la cuenta
 * con su sesión.
 */
func TestRegistroConOTPReal(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	const cedula = "1710034024"
	const correo = "e2e-registro@biess.local"

	// El registro también cuenta intentos: una corrida anterior podría
	// haber bloqueado la cuenta y la prueba mediría ese bloqueo, no el flujo.
	if err := entorno.ReiniciarIntentos(cedula); err != nil {
		t.Fatalf("no se pudo limpiar el estado de intentos: %v", err)
	}

	respuesta := cliente.Post(t, "/api/v1/auth/register/request", map[string]string{
		"cedula": cedula,
		"correo": correo,
	})
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la solicitud de registro devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	desafio, ok := respuesta.JSON()["desafio_id"].(string)
	if !ok || desafio == "" {
		t.Fatalf("la respuesta no trae el desafío: %s", respuesta.Cuerpo)
	}

	codigo := entorno.Correo.UltimoOTP(correo)
	if codigo == "" {
		t.Fatal("el servidor no envió el código por correo")
	}

	respuesta = cliente.Post(t, "/api/v1/auth/register/verify", map[string]string{
		"cedula":     cedula,
		"correo":     correo,
		"codigo_otp": codigo,
		"desafio_id": desafio,
		"nombres":    "SINTETICO",
		"apellidos":  "Registro",
		"password":   e2edata.Password,
	})
	if respuesta.Status != http.StatusCreated {
		t.Fatalf("la verificación devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	postulante, ok := respuesta.JSON()["postulante"].(map[string]any)
	if !ok || postulante["cedula"] != cedula {
		t.Fatalf("la verificación no devolvió el postulante: %s", respuesta.Cuerpo)
	}
	if postulante["rol"] != "postulante" {
		t.Fatalf("el registro no creó una cuenta de postulante: %v", postulante["rol"])
	}

	// La sesión queda abierta: el registro inicia sesión por sí mismo.
	respuesta = cliente.Get(t, "/api/v1/auth/session")
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el registro no dejó sesión abierta: %d %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestOTPInvalidoSeRechaza comprueba que un código que no es el emitido
 * no habilita la cuenta.
 */
func TestOTPInvalidoSeRechaza(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	const cedula = "1710034032"
	const correo = "e2e-otp-invalido@biess.local"

	if err := entorno.ReiniciarIntentos(cedula); err != nil {
		t.Fatalf("no se pudo limpiar el estado de intentos: %v", err)
	}

	respuesta := cliente.Post(t, "/api/v1/auth/register/request", map[string]string{
		"cedula": cedula,
		"correo": correo,
	})
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la solicitud de registro devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	desafio, _ := respuesta.JSON()["desafio_id"].(string)

	respuesta = cliente.Post(t, "/api/v1/auth/register/verify", map[string]string{
		"cedula":     cedula,
		"correo":     correo,
		"codigo_otp": "000000",
		"desafio_id": desafio,
		"nombres":    "SINTETICO",
		"apellidos":  "OTP",
		"password":   e2edata.Password,
	})
	if respuesta.Status == http.StatusCreated {
		t.Fatalf("el servidor aceptó un código inválido: %s", respuesta.Cuerpo)
	}
	if respuesta.CodigoError() == "" {
		t.Fatalf("el rechazo no usa el envelope de error: %s", respuesta.Cuerpo)
	}
}

/**
 * TestCedulaInvalidaSeRechazaEnElRegistro comprueba que la cédula se valida
 * con módulo 10 antes de crear la cuenta.
 */
func TestCedulaInvalidaSeRechazaEnElRegistro(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	respuesta := cliente.Post(t, "/api/v1/auth/register/request", map[string]string{
		"cedula": "1234567890",
		"correo": "e2e-cedula-mala@biess.local",
	})
	if respuesta.Status == http.StatusOK {
		t.Fatalf("el servidor aceptó una cédula inválida: %s", respuesta.Cuerpo)
	}
	if respuesta.CodigoError() == "" {
		t.Fatalf("el rechazo no usa el envelope de error: %s", respuesta.Cuerpo)
	}
}

/**
 * TestRecuperacionDeContrasenaConOTP comprueba el flujo de recuperación: se
 * pide un código al correo registrado, se confirma y la nueva contraseña sirve
 * para iniciar sesión.
 */
func TestRecuperacionDeContrasenaConOTP(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	const cedula = "2408380695"
	const correo = "e2e-reconsideracion@biess.local"
	const nueva = "NuevaContrasena.2026*"

	respuesta := cliente.Post(t, "/api/v1/auth/password/forgot", map[string]string{"cedula": cedula})
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la recuperación devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}
	desafio, _ := respuesta.JSON()["desafio_id"].(string)

	codigo := entorno.Correo.UltimoOTP(correo)
	if codigo == "" {
		t.Fatal("el servidor no envió el código de recuperación")
	}

	respuesta = cliente.Post(t, "/api/v1/auth/password/reset", map[string]string{
		"cedula":       cedula,
		"codigo_otp":   codigo,
		"desafio_id":   desafio,
		"new_password": nueva,
	})
	if respuesta.Status != http.StatusOK {
		t.Fatalf("el restablecimiento devolvió %d: %s", respuesta.Status, respuesta.Cuerpo)
	}

	// La cuenta sigue bloqueada por los intentos previos de otra prueba:
	// se libera para comprobar que la nueva contraseña sirve.
	if err := entorno.ReiniciarIntentos(cedula); err != nil {
		t.Fatalf("no se pudo limpiar el estado de intentos: %v", err)
	}

	respuesta = entorno.Cliente().Post(t, "/api/v1/auth/login", map[string]string{
		"cedula":   cedula,
		"password": nueva,
	})
	if respuesta.Status != http.StatusOK {
		t.Fatalf("la nueva contraseña no sirve: %d %s", respuesta.Status, respuesta.Cuerpo)
	}
}

/**
 * TestRecuperacionNoRevelaSiLaCuentaExiste comprueba la protección
 * anti-oráculo: la respuesta es la misma exista o no la cuenta.
 */
func TestRecuperacionNoRevelaSiLaCuentaExiste(t *testing.T) {
	entorno := Preparar(t)
	cliente := entorno.Cliente()

	// La cédula inexistente debe ser válida: si no, el servidor la rechaza
	// por formato y la prueba mediría eso en vez de la protección.
	const cedulaInexistente = "1710034040"
	if err := entorno.ReiniciarIntentos(cedulaInexistente); err != nil {
		t.Fatalf("no se pudo limpiar el estado de intentos: %v", err)
	}

	respuestaExistente := cliente.Post(t, "/api/v1/auth/password/forgot", map[string]string{"cedula": "2406425922"})
	respuestaInexistente := entorno.Cliente().Post(t, "/api/v1/auth/password/forgot", map[string]string{"cedula": cedulaInexistente})

	if respuestaExistente.Status != respuestaInexistente.Status {
		t.Fatalf("la respuesta difiere entre cuentas existentes e inexistentes: %d y %d",
			respuestaExistente.Status, respuestaInexistente.Status)
	}
	if respuestaExistente.JSON()["mensaje"] != respuestaInexistente.JSON()["mensaje"] {
		t.Fatalf("el mensaje difiere entre cuentas existentes e inexistentes: %s y %s",
			respuestaExistente.JSON()["mensaje"], respuestaInexistente.JSON()["mensaje"])
	}
}
