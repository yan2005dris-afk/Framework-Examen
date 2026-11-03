package httpx

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/shared/domain"
)

/**
 * ParsePageRequest lee los parámetros de paginación por offset de la query
 * string. Los valores fuera de rango se ajustan a los límites permitidos y
 * un valor no numérico se considera un error del cliente.
 *
 * @param {http.Request} request - Petición recibida.
 * @returns {PageRequest} Paginación normalizada.
 * @throws {Error} Si algún parámetro no es un número válido.
 */
func ParsePageRequest(request *http.Request) (domain.PageRequest, error) {
	query := request.URL.Query()

	page, err := parseIntParam(query.Get("page"), 1)
	if err != nil {
		return domain.PageRequest{}, fmt.Errorf("el parámetro page no es válido: %w", err)
	}

	pageSize, err := parseIntParam(query.Get("pageSize"), domain.DefaultPageSize)
	if err != nil {
		return domain.PageRequest{}, fmt.Errorf("el parámetro pageSize no es válido: %w", err)
	}

	return domain.NewPageRequest(page, pageSize), nil
}

/**
 * ParseCursorRequest lee los parámetros de paginación por cursor de la query
 * string.
 *
 * @param {http.Request} request - Petición recibida.
 * @returns {CursorRequest} Paginación normalizada con el cursor ya decodificado.
 * @throws {Error} Si el límite no es numérico o el cursor no es válido.
 */
func ParseCursorRequest(request *http.Request) (domain.CursorRequest, error) {
	query := request.URL.Query()

	limit, err := parseIntParam(query.Get("limit"), domain.DefaultPageSize)
	if err != nil {
		return domain.CursorRequest{}, fmt.Errorf("el parámetro limit no es válido: %w", err)
	}

	var cursor *domain.Cursor
	if raw := query.Get("cursor"); raw != "" {
		decoded, err := domain.DecodeCursor(raw)
		if err != nil {
			return domain.CursorRequest{}, err
		}
		cursor = &decoded
	}

	return domain.NewCursorRequest(limit, cursor), nil
}

/**
 * parseIntParam convierte un parámetro de consulta a número, devolviendo el
 * valor por defecto cuando el parámetro no viene.
 *
 * @param {string} raw - Valor recibido del cliente.
 * @param {number} fallback - Valor usado cuando el parámetro falta.
 * @returns {number} Valor numérico de la consulta.
 * @throws {Error} Si el valor existe pero no es numérico.
 */
func parseIntParam(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("se esperaba un número entero: %w", err)
	}

	return value, nil
}
