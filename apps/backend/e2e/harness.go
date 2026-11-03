// Package e2e contiene la suite de pruebas de extremo a extremo de la API.
//
// Las pruebas no reimplementan la API: levantan el mismo router que usa el
// binario de producción (internal/platform/appkit) sobre un esquema de
// PostgreSQL aislado, Redis real y almacenamiento local temporal. Cada prueba
// habla HTTP con cookies de sesión, como lo hace el navegador.
package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/e2edata"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/appkit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/httpx"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/storage"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission/print"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
)

/**
 * originPermitido es el origen que el guard de CSRF acepta en la suite. La
 * suite habla con la API como lo haría el navegador: mismo origen permitido.
 */
const originPermitido = "http://localhost:5173"

/**
 * base64Key es una clave de buzón válida para las pruebas: el cifrado de la
 * cola exige 32 bytes y no se puede reutilizar la clave de producción.
 */
const base64Key = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

/**
 * claveFirmaPruebas es la clave de firma de las URLs temporales del
 * almacenamiento local. Exige al menos 32 bytes.
 */
const claveFirmaPruebas = "clave-firma-solo-para-pruebas-e2e-2026"

/**
 * Entorno es la infraestructura compartida por las pruebas de una corrida.
 */
type Entorno struct {
	// Server es el servidor HTTP de pruebas.
	Server *httptest.Server
	// Pool es el pool sobre el esquema aislado.
	Pool *pgxpool.Pool
	// Queries son las consultas tipadas sobre el esquema aislado.
	Queries *sqlcgen.Queries
	// Dataset es el conjunto de datos sembrado.
	Dataset e2edata.Dataset
	// Correo captura los mensajes entregados, para leer los códigos OTP.
	Correo *CorreoMemoria
	// Redis es el cache real usado por los controles de identidad.
	Redis *cache.RedisClient
}

/**
 * Cliente es un cliente HTTP con tarro de cookies, igual que el navegador.
 */
type Cliente struct {
	// Base es la URL base del servidor de pruebas.
	Base string
	// HTTP es el cliente con tarro de cookies.
	HTTP *http.Client
}

/**
 * Respuesta es el resultado legible de una petición HTTP.
 */
type Respuesta struct {
	// Status es el código HTTP de la respuesta.
	Status int
	// Cuerpo es el cuerpo crudo de la respuesta.
	Cuerpo []byte
	// Headers son las cabeceras de la respuesta.
	Headers http.Header
}

/**
 * JSON devuelve el cuerpo de la respuesta como objeto genérico.
 *
 * @param {Respuesta} respuesta - Respuesta de la petición.
 * @returns {map[string]any} Cuerpo parseado, o un mapa vacío si no es JSON.
 */
func (r Respuesta) JSON() map[string]any {
	var payload map[string]any
	if err := json.Unmarshal(r.Cuerpo, &payload); err != nil {
		return map[string]any{}
	}
	return payload
}

/**
 * CodigoError devuelve el código estable del envelope de error, si lo hay.
 *
 * @param {Respuesta} respuesta - Respuesta de la petición.
 * @returns {string} Código de error o cadena vacía.
 */
func (r Respuesta) CodigoError() string {
	errorEnvelope, ok := r.JSON()["error"].(map[string]any)
	if !ok {
		return ""
	}
	codigo, _ := errorEnvelope["codigo"].(string)
	return codigo
}

/**
 * Preparar levanta el servidor de pruebas y siembra el conjunto de datos.
 *
 * Las pruebas se omiten cuando no hay una base de pruebas configurada: el
 * mismo criterio que usa internal/testutil para las pruebas de integración.
 *
 * @param {testing.T} t - Prueba en curso.
 * @returns {*Entorno} Entorno listo para usar.
 */
func Preparar(t *testing.T) *Entorno {
	t.Helper()

	pool := testutil.Postgres(t, "E2E_TEST_DATABASE_URL")
	redisURL := os.Getenv("REDIS_TEST_URL")
	if redisURL == "" {
		redisURL = os.Getenv("REDIS_URL")
	}
	if redisURL == "" {
		t.Skip("defina REDIS_TEST_URL o REDIS_URL para ejecutar la suite E2E de la API")
	}

	redisClient, err := cache.NewRedis(context.Background(), redisURL)
	if err != nil {
		t.Skipf("Redis no está disponible: %v", err)
	}
	t.Cleanup(func() { _ = redisClient.Close() })

	correo := &CorreoMemoria{}
	queries := sqlcgen.New(pool)
	outbox, err := mail.NewOutbox(queries, base64Key)
	if err != nil {
		t.Fatalf("no se pudo abrir la cola de correo: %v", err)
	}

	objetos, err := storage.NewSignedLocal(t.TempDir(), "http://localhost:8080/almacenamiento", claveFirmaPruebas)
	if err != nil {
		t.Fatalf("no se pudo preparar el almacenamiento local: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	attemptsStore := cache.NewAttemptsStore(redisClient)
	otpStore, err := cache.NewConfiguredOTPStore(redisClient, time.Second, 3)
	if err != nil {
		t.Fatalf("no se pudo configurar el almacén de códigos: %v", err)
	}

	// El seed se aplica antes de construir el router: el proveedor de
	// elegibilidad necesita los identificadores de las cuentas sembradas.
	dataset, err := e2edata.Seed(ctx, pool)
	if err != nil {
		t.Fatalf("no se pudo sembrar el conjunto de pruebas: %v", err)
	}
	evaluables := map[uuid.UUID]bool{}
	for _, cuenta := range dataset.Accounts {
		evaluables[cuenta.ID] = true
	}

	router, _, err := appkit.NewRouter(ctx, appkit.Dependencies{
		Logger:              testLogger(),
		Pool:                pool,
		Queries:             queries,
		Version:             "e2e",
		SessionTTL:          time.Hour,
		AttemptsStore:       attemptsStore,
		OTPStore:            otpStore,
		CatalogStore:        redisClient,
		Mailer:              correo,
		Outbox:              outbox,
		Storage:             objetos,
		Printer:             print.NewRenderer().WithMark(e2edata.Mark),
		EligibilityProvider: NewProveedorElegibilidad(evaluables),
		DemoMark:            e2edata.Mark,
	})
	if err != nil {
		t.Fatalf("no se pudo construir el router: %v", err)
	}

	handler := httpx.RequestID(
		httpx.Logging(
			httpx.Recover(
				httpx.CORS(
					httpx.CSRF(
						httpx.MaxBodySize(httpx.Deadline(router, 30*time.Second), 10<<20),
						[]string{originPermitido},
					),
					[]string{originPermitido},
				),
			),
		),
	)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	entorno := &Entorno{
		Server:  server,
		Pool:    pool,
		Queries: queries,
		Dataset: dataset,
		Correo:  correo,
		Redis:   redisClient,
	}

	// El bloqueo por intentos vive en Redis con diez minutos de vigencia y
	// sobrevive a la corrida anterior. Sin esta limpieza, una prueba que
	// agota los intentos dejaría bloqueada a la cuenta de la siguiente.
	for cedula := range dataset.Accounts {
		if err := entorno.ReiniciarIntentos(cedula); err != nil {
			t.Fatalf("no se pudo limpiar el estado de intentos de %s: %v", cedula, err)
		}
	}

	return entorno
}

/**
 * ReiniciarIntentos limpia el contador de intentos y el bloqueo de una cuenta.
 *
 * El bloqueo vive en Redis con una vigencia de diez minutos: sin esta
 * limpieza, una segunda corrida de la suite encontraría la cuenta ya
 * bloqueada y la prueba fallaría sin culpa del código bajo prueba.
 *
 * @param {string} cedula - Cédula de la cuenta a liberar.
 * @returns {error} Si Redis no responde.
 */
func (e *Entorno) ReiniciarIntentos(cedula string) error {
	huella := sha256.Sum256([]byte(cedula))
	prefijo := hex.EncodeToString(huella[:])
	ctx := context.Background()
	if err := e.Redis.Delete(ctx, "identity:auth:{"+prefijo+"}:attempts"); err != nil {
		return err
	}
	return e.Redis.Delete(ctx, "identity:auth:{"+prefijo+"}:lock")
}

/**
 * ClienteConSesionDe devuelve un cliente que ya inició sesión con la cuenta
 * de la cédula indicada.
 *
 * @param {testing.T} t - Prueba en curso.
 * @param {*Entorno} entorno - Entorno de pruebas.
 * @param {string} cedula - Cédula de la cuenta sembrada.
 * @returns {*Cliente} Cliente con sesión iniciada.
 */
func (e *Entorno) ClienteConSesionDe(t *testing.T, cedula string) *Cliente {
	t.Helper()
	cliente := e.Cliente()
	cuenta, ok := e.Dataset.Accounts[cedula]
	if !ok {
		t.Fatalf("no hay una cuenta sembrada con la cédula %s", cedula)
	}
	respuesta := cliente.Post(t, "/api/v1/auth/login", map[string]string{
		"cedula":   cuenta.Cedula,
		"password": e2edata.Password,
	})
	if respuesta.Status != http.StatusOK {
		t.Fatalf("no se pudo iniciar sesión con %s: %d %s", cedula, respuesta.Status, respuesta.Cuerpo)
	}
	return cliente
}

/**
 * Cliente nuevo devuelve un cliente HTTP con su propio tarro de cookies.
 *
 * @param {*Entorno} entorno - Entorno de pruebas.
 * @returns {*Cliente} Cliente listo para usar.
 */
func (e *Entorno) Cliente() *Cliente {
	tarro, _ := cookiejar.New(nil)
	return &Cliente{
		Base: e.Server.URL,
		HTTP: &http.Client{Jar: tarro, Timeout: 30 * time.Second},
	}
}

/**
 * ClienteConSesion devuelve un cliente que ya inició sesión con el rol dado.
 *
 * @param {testing.T} t - Prueba en curso.
 * @param {*Entorno} entorno - Entorno de pruebas.
 * @param {string} rol - Rol de la cuenta sembrada.
 * @returns {*Cliente} Cliente con sesión iniciada.
 */
func (e *Entorno) ClienteConSesion(t *testing.T, rol string) *Cliente {
	t.Helper()
	cliente := e.Cliente()
	cuenta := e.Dataset.AccountByRole(rol)
	if cuenta.Cedula == "" {
		t.Fatalf("no hay una cuenta sembrada con el rol %s", rol)
	}
	respuesta := cliente.Post(t, "/api/v1/auth/login", map[string]string{
		"cedula":   cuenta.Cedula,
		"password": e2edata.Password,
	})
	if respuesta.Status != http.StatusOK {
		t.Fatalf("no se pudo iniciar sesión con el rol %s: %d %s", rol, respuesta.Status, respuesta.Cuerpo)
	}
	return cliente
}

/**
 * Get ejecuta una petición GET con las cabeceras de una sesión.
 *
 * @param {*testing.T} t - Prueba en curso.
 * @param {string} ruta - Ruta relativa al servidor.
 * @returns {Respuesta} Respuesta leída.
 */
func (c *Cliente) Get(t *testing.T, ruta string) Respuesta {
	t.Helper()
	return c.do(t, http.MethodGet, ruta, nil, nil)
}

/**
 * Post ejecuta una petición POST con cuerpo JSON.
 *
 * @param {*testing.T} t - Prueba en curso.
 * @param {string} ruta - Ruta relativa al servidor.
 * @param {any} cuerpo - Cuerpo serializable a JSON.
 * @returns {Respuesta} Respuesta leída.
 */
func (c *Cliente) Post(t *testing.T, ruta string, cuerpo any) Respuesta {
	t.Helper()
	return c.do(t, http.MethodPost, ruta, cuerpo, nil)
}

/**
 * Patch ejecuta una petición PATCH con cuerpo JSON.
 *
 * @param {*testing.T} t - Prueba en curso.
 * @param {string} ruta - Ruta relativa al servidor.
 * @param {any} cuerpo - Cuerpo serializable a JSON.
 * @returns {Respuesta} Respuesta leída.
 */
func (c *Cliente) Patch(t *testing.T, ruta string, cuerpo any) Respuesta {
	t.Helper()
	return c.do(t, http.MethodPatch, ruta, cuerpo, nil)
}

/**
 * Put ejecuta una petición PUT con cuerpo JSON.
 *
 * @param {*testing.T} t - Prueba en curso.
 * @param {string} ruta - Ruta relativa al servidor.
 * @param {any} cuerpo - Cuerpo serializable a JSON.
 * @returns {Respuesta} Respuesta leída.
 */
func (c *Cliente) Put(t *testing.T, ruta string, cuerpo any) Respuesta {
	t.Helper()
	return c.do(t, http.MethodPut, ruta, cuerpo, nil)
}

/**
 * Delete ejecuta una petición DELETE.
 *
 * @param {*testing.T} t - Prueba en curso.
 * @param {string} ruta - Ruta relativa al servidor.
 * @returns {Respuesta} Respuesta leída.
 */
func (c *Cliente) Delete(t *testing.T, ruta string) Respuesta {
	t.Helper()
	return c.do(t, http.MethodDelete, ruta, nil, nil)
}

/**
 * do ejecuta la petición HTTP con las cabeceras que espera el servidor.
 *
 * @param {*testing.T} t - Prueba en curso.
 * @param {string} metodo - Método HTTP.
 * @param {string} ruta - Ruta relativa al servidor.
 * @param {any} cuerpo - Cuerpo serializable a JSON, opcional.
 * @param {map[string]string} cabeceras - Cabeceras adicionales, opcional.
 * @returns {Respuesta} Respuesta leída.
 */
func (c *Cliente) do(t *testing.T, metodo string, ruta string, cuerpo any, cabeceras map[string]string) Respuesta {
	t.Helper()

	var lector io.Reader
	if cuerpo != nil {
		raw, err := json.Marshal(cuerpo)
		if err != nil {
			t.Fatalf("no se pudo serializar el cuerpo: %v", err)
		}
		lector = strings.NewReader(string(raw))
	}

	peticion, err := http.NewRequest(metodo, c.Base+ruta, lector)
	if err != nil {
		t.Fatalf("no se pudo preparar la petición: %v", err)
	}
	peticion.Header.Set("Origin", originPermitido)
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	for clave, valor := range cabeceras {
		peticion.Header.Set(clave, valor)
	}

	respuesta, err := c.HTTP.Do(peticion)
	if err != nil {
		t.Fatalf("no se pudo ejecutar la petición %s %s: %v", metodo, ruta, err)
	}
	defer respuesta.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(respuesta.Body, 32<<20))
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}
	return Respuesta{Status: respuesta.StatusCode, Cuerpo: raw, Headers: respuesta.Header}
}

/**
 * CorreoMemoria captura los mensajes enviados para poder leer los códigos.
 */
type CorreoMemoria struct {
	// mu protege los mensajes capturados.
	mu sync.Mutex
	// mensajes son los correos entregados.
	mensajes []mail.Message
}

/**
 * Send registra el mensaje entregado.
 *
 * @param {context.Context} ctx - Contexto del envío.
 * @param {mail.Message} msg - Mensaje entregado.
 * @returns {error} Siempre nil: la entrega en memoria no falla.
 */
func (c *CorreoMemoria) Send(_ context.Context, msg mail.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mensajes = append(c.mensajes, msg)
	return nil
}

/**
 * UltimoOTP devuelve el último código de seis dígitos enviado a un correo.
 *
 * @param {string} correo - Destinatario buscado.
 * @returns {string} Código encontrado o cadena vacía.
 */
func (c *CorreoMemoria) UltimoOTP(correo string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.mensajes) - 1; i >= 0; i-- {
		msg := c.mensajes[i]
		if msg.To != correo {
			continue
		}
		if codigo := buscarCodigo(msg.Body); codigo != "" {
			return codigo
		}
	}
	return ""
}

/**
 * buscarCodigo extrae un código de seis dígitos del cuerpo del mensaje.
 *
 * @param {string} cuerpo - Cuerpo del correo.
 * @returns {string} Código encontrado o cadena vacía.
 */
func buscarCodigo(cuerpo string) string {
	for _, parte := range strings.Fields(cuerpo) {
		limpio := strings.Trim(parte, ".,;:!?()[]<>")
		if len(limpio) == 6 {
			soloDigitos := true
			for _, caracter := range limpio {
				if caracter < '0' || caracter > '9' {
					soloDigitos = false
					break
				}
			}
			if soloDigitos {
				return limpio
			}
		}
	}
	return ""
}

/**
 * testLogger devuelve un registrador silencioso: la salida de la suite es el
 * resultado de las aserciones, no el ruido del servidor.
 *
 * @returns {*slog.Logger} Registrador con destino descartado.
 */
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
