package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

/**
 * DefaultMaxBodyBytes es el tamaño máximo admitido por defecto para el cuerpo
 * de una petición: 10 MiB, suficiente para los PDF adjuntos del formulario.
 */
const DefaultMaxBodyBytes int64 = 10 * 1024 * 1024

/**
 * Config agrupa la configuración de la aplicación leída desde el entorno.
 */
type Config struct {
	Environment               string
	Port                      int
	PublicURL                 string
	DatabaseURL               string
	DatabaseRequired          bool
	RedisURL                  string
	S3Endpoint                string
	S3PublicEndpoint          string
	S3Region                  string
	S3AccessKey               string
	S3SecretKey               string
	S3SessionToken            string
	CORSOrigins               []string
	SessionTTL                time.Duration
	ReadTimeout               time.Duration
	WriteTimeout              time.Duration
	IdleTimeout               time.Duration
	RequestTimeout            time.Duration
	MaxBodyBytes              int64
	Version                   string
	SMTPHost                  string
	SMTPPort                  int
	SMTPUser                  string
	SMTPPassword              string
	SMTPFrom                  string
	SMTPUseTLS                bool
	ResendAPIKey              string
	MailOutboxKey             string
	OTPResendDelay            time.Duration
	OTPMaxSends               int
	EligibilitySimulationFile string
	RunEmbeddedWorkers        bool
}

/**
 * Load construye la configuración leyendo variables de entorno y aplicando
 * valores por defecto aptos para el entorno local de desarrollo.
 *
 * @returns {Config} Configuración con los valores por defecto ya aplicados.
 * @throws {Error} Si una variable existe pero no se puede convertir al tipo esperado.
 */
func Load() (Config, error) {
	otpDelay, err := getDuration("OTP_RESEND_DELAY", time.Minute)
	if err != nil {
		return Config{}, err
	}
	otpSends, err := getInt("OTP_MAX_SENDS", 3)
	if err != nil {
		return Config{}, err
	}
	if otpDelay < time.Second || otpDelay >= 10*time.Minute || otpSends < 1 || otpSends > 10 {
		return Config{}, fmt.Errorf("OTP_RESEND_DELAY debe ser entre 1s y menos de 10m; OTP_MAX_SENDS entre 1 y 10")
	}
	port, err := getInt("PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	databaseRequired, err := getBool("DB_REQUIRED", false)
	if err != nil {
		return Config{}, err
	}

	sessionTTL, err := getDuration("SESSION_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := getDuration("HTTP_READ_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := getDuration("HTTP_WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}

	requestTimeout, err := getDuration("HTTP_REQUEST_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	maxBodyBytes, err := getInt64("MAX_REQUEST_BODY_BYTES", DefaultMaxBodyBytes)
	if err != nil {
		return Config{}, err
	}

	smtpPort, err := getInt("SMTP_PORT", 1025)
	if err != nil {
		return Config{}, err
	}

	smtpUseTLS, err := getBool("SMTP_TLS", false)
	if err != nil {
		return Config{}, err
	}

	environment := getString("APP_ENV", "development")

	return Config{
		Environment:       environment,
		Port:              port,
		PublicURL:         getString("APP_URL", "http://localhost:5173"),
		DatabaseURL:       getString("DATABASE_URL", ""),
		DatabaseRequired:  databaseRequired,
		RedisURL:          getString("REDIS_URL", ""),
		S3Endpoint:        getString("S3_ENDPOINT", "http://rustfs:9000/biess"),
		S3PublicEndpoint:  getString("S3_PUBLIC_ENDPOINT", ""),
		S3Region:          getString("S3_REGION", "us-east-1"),
		S3AccessKey:       getString("S3_ACCESS_KEY", "rustfsadmin"),
		S3SecretKey:       getString("S3_SECRET_KEY", "rustfsadmin"),
		S3SessionToken:    getString("S3_SESSION_TOKEN", ""),
		CORSOrigins:       loadOrigins(environment),
		SessionTTL:        sessionTTL,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		RequestTimeout:    requestTimeout,
		MaxBodyBytes:      maxBodyBytes,
		Version:           getString("APP_VERSION", "dev"),
		SMTPHost:          getString("SMTP_HOST", ""),
		SMTPPort:          smtpPort,
		SMTPUser:          getString("SMTP_USER", ""),
		SMTPPassword:      getString("SMTP_PASSWORD", ""),
		SMTPFrom:          getString("SMTP_FROM", "no-reply@biess.local"),
		SMTPUseTLS:        smtpUseTLS,
		ResendAPIKey:      getString("RESEND_API_KEY", ""),
		MailOutboxKey:     getString("MAIL_OUTBOX_KEY", ""),
		OTPResendDelay:    otpDelay, OTPMaxSends: otpSends,
		EligibilitySimulationFile: getString("ELIGIBILITY_SIMULATION_FILE", ""),
		RunEmbeddedWorkers:        getString("EMBEDDED_WORKERS", "false") == "true" || getString("EMBEDDED_WORKERS", "false") == "1",
	}, nil
}

/**
 * Address devuelve la dirección de escucha del servidor HTTP.
 *
 * @returns {string} Dirección en formato ":puerto".
 */
func (c Config) Address() string {
	return fmt.Sprintf(":%d", c.Port)
}

/**
 * IsProduction indica si la aplicación corre en un entorno de producción.
 *
 * @returns {boolean} True cuando el entorno es de producción.
 */
func (c Config) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "produccion"
}

/**
 * getString lee una variable de entorno como texto.
 *
 * @param {string} key - Nombre de la variable de entorno.
 * @param {string} fallback - Valor devuelto cuando la variable no existe.
 * @returns {string} Valor de la variable o el valor por defecto.
 */
func getString(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

/**
 * getInt lee una variable de entorno como número entero.
 *
 * @param {string} key - Nombre de la variable de entorno.
 * @param {number} fallback - Valor devuelto cuando la variable no existe.
 * @returns {number} Valor de la variable o el valor por defecto.
 * @throws {Error} Si la variable existe pero no es un entero válido.
 */
func getInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s debe ser un número entero: %w", key, err)
	}
	return parsed, nil
}

/**
 * getInt64 lee una variable de entorno como entero de 64 bits.
 *
 * @param {string} key - Nombre de la variable de entorno.
 * @param {number} fallback - Valor devuelto cuando la variable no existe.
 * @returns {number} Valor de la variable o el valor por defecto.
 * @throws {Error} Si la variable existe pero no es un entero válido.
 */
func getInt64(key string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s debe ser un número entero: %w", key, err)
	}
	return parsed, nil
}

/**
 * getBool lee una variable de entorno como booleano.
 *
 * @param {string} key - Nombre de la variable de entorno.
 * @param {boolean} fallback - Valor devuelto cuando la variable no existe.
 * @returns {boolean} Valor de la variable o el valor por defecto.
 * @throws {Error} Si la variable existe pero no es "true" o "false".
 */
func getBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s debe ser true o false: %w", key, err)
	}
	return parsed, nil
}

/**
 * getDuration lee una variable de entorno como duración.
 *
 * @param {string} key - Nombre de la variable de entorno.
 * @param {number} fallback - Valor devuelto cuando la variable no existe.
 * @returns {number} Duración en nanosegundos o el valor por defecto.
 * @throws {Error} Si la variable existe pero no es una duración válida.
 */
func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s debe ser una duración válida (ej. 30s, 15m): %w", key, err)
	}
	return parsed, nil
}

/**
 * loadOrigins resuelve la lista de orígenes permitidos. Una variable definida
 * —incluso vacía— se respeta al pie de la letra: vacío significa denegar
 * todo, tal como promete el contrato documentado. Sólo cuando la variable no
 * existe se aplica el valor por defecto del entorno, que en producción es
 * denegar todo y deja la API sirviéndose desde un único origen.
 *
 * @param {string} environment - Entorno de la aplicación.
 * @returns {string[]} Orígenes permitidos; una lista vacía deniega cualquier
 * origen cruzado.
 */
func loadOrigins(environment string) []string {
	if value, ok := os.LookupEnv("CORS_ALLOWED_ORIGINS"); ok {
		return parseList(value)
	}
	if environment == "production" || environment == "produccion" {
		return []string{}
	}
	return []string{"http://localhost:5173"}
}

/**
 * parseList divide una lista separada por comas y descarta los elementos
 * vacíos.
 *
 * @param {string} value - Lista separada por comas.
 * @returns {string[]} Valores sin espacios sobrantes.
 */
func parseList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
