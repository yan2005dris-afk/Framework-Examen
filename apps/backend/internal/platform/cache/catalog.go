package cache

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// CatalogTTL es el tiempo de vida de los catálogos geográficos en caché.
const CatalogTTL = 24 * time.Hour

/**
 * CatalogStore gestiona la caché de catálogos geográficos (países, provincias
 * y cantones). Reduce las consultas a PostgreSQL para datos que cambian muy
 * poco: el TTL es de 24 horas.
 *
 * Claves:
 *   - catalog:countries
 *   - catalog:provinces:{pais}
 *   - catalog:cities:{pais}:{provincia}
 */
type CatalogStore struct {
	cache Cache
}

/**
 * NewCatalogStore crea un CatalogStore que usa el Cache provisto.
 *
 * @param {Cache} c - Adaptador de caché (Redis u otro).
 * @returns {*CatalogStore} Store listo para usar.
 */
func NewCatalogStore(c Cache) *CatalogStore {
	return &CatalogStore{cache: c}
}

/**
 * GetOrSet devuelve el valor cacheado para la clave dada.
 * Si la clave no existe, llama a fetch para obtenerlo, lo guarda con el TTL
 * de catálogos y lo devuelve.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} key - Clave del catálogo.
 * @param {func} fetch - Función que carga el valor cuando no está en caché.
 * @returns {string} Valor en JSON del catálogo.
 * @throws {error} Error de Redis o de la función fetch.
 */
func (s *CatalogStore) GetOrSet(ctx context.Context, key string, fetch func() (string, error)) (string, error) {
	val, err := s.cache.Get(ctx, key)
	if err == nil {
		return val, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return "", fmt.Errorf("catalog: Get(%q): %w", key, err)
	}

	// Cache miss: cargar desde la fuente.
	fresh, err := fetch()
	if err != nil {
		return "", fmt.Errorf("catalog: fetch(%q): %w", key, err)
	}

	// Guardar en caché; si falla no es un error fatal: la petición igual responde.
	if setErr := s.cache.Set(ctx, key, fresh, CatalogTTL); setErr != nil {
		// Solo registramos el fallo; no bloqueamos la respuesta.
		_ = setErr
	}

	return fresh, nil
}

/**
 * CountriesKey devuelve la clave de caché para la lista de países.
 *
 * @returns {string} Clave de caché.
 */
func CountriesKey() string {
	return "catalog:countries"
}

/**
 * ProvincesKey devuelve la clave de caché para las provincias de un país.
 *
 * @param {string} countryCode - Código ISO del país.
 * @returns {string} Clave de caché.
 */
func ProvincesKey(countryCode string) string {
	return fmt.Sprintf("catalog:provinces:%s", countryCode)
}

/**
 * CitiesKey devuelve la clave de caché para los cantones de una provincia.
 *
 * @param {string} countryCode - Código ISO del país.
 * @param {string} provinceCode - Código de la provincia.
 * @returns {string} Clave de caché.
 */
func CitiesKey(countryCode, provinceCode string) string {
	return fmt.Sprintf("catalog:cities:%s:%s", countryCode, provinceCode)
}
