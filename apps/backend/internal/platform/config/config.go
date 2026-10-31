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
	Environment      string
	Port             int
	PublicURL        string
	DatabaseURL      string
	DatabaseRequired bool
	StorageDir       string
	CORSOrigins      []string
	SessionTTL       time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	RequestTimeout   time.Duration
	MaxBodyBytes     int64
	Version          string
}

/**
 * Load construye la configuración leyendo variables de entorno y aplicando
 * valores por defecto aptos para el entorno local de desarrollo.
 *
 * @returns {Config} Configuración con los valores por defecto ya aplicados.
 * @throws {Error} Si una variable existe pero no se puede convertir al tipo esperado.
 */
func Load() (Config, error) {
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

	return Config{
		Environment:      getString("APP_ENV", "development"),
		Port:             port,
		PublicURL:        getString("APP_URL", "http://localhost:5173"),
		DatabaseURL:      getString("DATABASE_URL", ""),
		DatabaseRequired: databaseRequired,
		StorageDir:       getString("STORAGE_DIR", "./data/uploads"),
		CORSOrigins:      getList("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		SessionTTL:       sessionTTL,
		ReadTimeout:      readTimeout,
		WriteTimeout:     writeTimeout,
		IdleTimeout:      idleTimeout,
		RequestTimeout:   requestTimeout,
		MaxBodyBytes:     maxBodyBytes,
		Version:          getString("APP_VERSION", "dev"),
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
 * getList lee una variable de entorno como lista separada por comas.
 *
 * @param {string} key - Nombre de la variable de entorno.
 * @param {string[]} fallback - Valor devuelto cuando la variable no existe.
 * @returns {string[]} Lista de valores sin espacios sobrantes.
 */
func getList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
