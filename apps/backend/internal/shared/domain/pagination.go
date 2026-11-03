package domain

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

/**
 * Errores de la paginación compartida.
 */
var (
	ErrInvalidCursor = errors.New("el cursor de paginación no es válido")
)

/**
 * Tamaño máximo y mínimo de una página de resultados.
 */
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

/**
 * PageRequest describe la página de resultados solicitada por el cliente.
 */
type PageRequest struct {
	Page     int
	PageSize int
}

/**
 * NewPageRequest normaliza los parámetros de paginación recibidos del cliente.
 *
 * @param {number} page - Número de página empezando en 1.
 * @param {number} pageSize - Cantidad de elementos por página.
 * @returns {PageRequest} Paginación normalizada.
 */
func NewPageRequest(page int, pageSize int) PageRequest {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	return PageRequest{Page: page, PageSize: pageSize}
}

/**
 * Limit devuelve la cantidad máxima de filas a leer en la consulta.
 *
 * @returns {number} Límite para el LIMIT de SQL.
 */
func (p PageRequest) Limit() int {
	return p.PageSize
}

/**
 * Offset devuelve la cantidad de filas a omitir en la consulta.
 *
 * @returns {number} Offset para el OFFSET de SQL.
 */
func (p PageRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

/**
 * Page envuelve una página de resultados junto con sus metadatos.
 */
type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

/**
 * NewPage construye la respuesta paginada a partir de los elementos leídos.
 *
 * @param {T[]} items - Elementos de la página actual.
 * @param {number} total - Cantidad total de registros disponibles.
 * @param {PageRequest} request - Paginación solicitada.
 * @returns {Page} Página lista para serializar.
 */
func NewPage[T any](items []T, total int64, request PageRequest) Page[T] {
	if items == nil {
		items = []T{}
	}

	return Page[T]{
		Items:    items,
		Page:     request.Page,
		PageSize: request.PageSize,
		Total:    total,
	}
}

/**
 * Cursor es un token opaco de paginación por cursor. Cada contexto guarda en
 * él los valores de su clave de orden (por ejemplo, la fecha de modificación
 * y el identificador), de modo que el cliente nunca interpreta su contenido.
 */
type Cursor struct {
	values []string
}

/**
 * NewCursor crea un cursor con los valores de la clave de orden.
 *
 * @param {string[]} values - Valores ordenados de la clave de orden.
 * @returns {Cursor} Cursor listo para codificar.
 */
func NewCursor(values ...string) Cursor {
	return Cursor{values: values}
}

/**
 * Encode codifica el cursor en un token opaco seguro para URLs.
 *
 * @returns {string} Token en base64 URL-safe.
 */
func (c Cursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.Join(c.values, "|")))
}

/**
 * DecodeCursor decodifica un token de cursor recibido del cliente.
 *
 * @param {string} raw - Token recibido en la query string.
 * @returns {Cursor} Cursor con sus valores.
 * @throws {Error} ErrInvalidCursor si el token no es válido.
 */
func DecodeCursor(raw string) (Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: el token no es un base64 válido", ErrInvalidCursor)
	}

	values := strings.Split(string(decoded), "|")
	if len(values) == 0 || values[0] == "" {
		return Cursor{}, fmt.Errorf("%w: el token está vacío", ErrInvalidCursor)
	}

	return Cursor{values: values}, nil
}

/**
 * Value devuelve el valor del cursor en la posición indicada.
 *
 * @param {number} index - Posición del valor dentro de la clave de orden.
 * @returns {string} Valor de la clave de orden.
 * @returns {boolean} True cuando la posición existe.
 */
func (c Cursor) Value(index int) (string, bool) {
	if index < 0 || index >= len(c.values) {
		return "", false
	}
	return c.values[index], true
}

/**
 * CursorRequest describe la página solicitada en modo cursor.
 */
type CursorRequest struct {
	Limit  int
	Cursor *Cursor
}

/**
 * NewCursorRequest normaliza los parámetros de paginación por cursor.
 *
 * @param {number} limit - Cantidad de elementos por página.
 * @param {Cursor} cursor - Cursor de la página anterior; nil para la primera.
 * @returns {CursorRequest} Paginación normalizada.
 */
func NewCursorRequest(limit int, cursor *Cursor) CursorRequest {
	if limit < 1 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	return CursorRequest{Limit: limit, Cursor: cursor}
}

/**
 * CursorPage envuelve una página de resultados obtenida por cursor.
 */
type CursorPage[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
	Limit      int    `json:"limit"`
}

/**
 * NewCursorPage construye la respuesta paginada por cursor.
 *
 * @param {T[]} items - Elementos de la página actual.
 * @param {string} nextCursor - Cursor de la página siguiente; vacío si no hay.
 * @param {boolean} hasMore - Indica si existe una página siguiente.
 * @param {number} limit - Límite aplicado a la consulta.
 * @returns {CursorPage} Página lista para serializar.
 */
func NewCursorPage[T any](items []T, nextCursor string, hasMore bool, limit int) CursorPage[T] {
	if items == nil {
		items = []T{}
	}

	return CursorPage[T]{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Limit:      limit,
	}
}
