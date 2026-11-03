package httpx

import (
	"errors"
	"net/http"
	"net/url"
)

/** ValidateQuery rechaza filtros desconocidos/duplicados y valores vacíos explícitos.
 * @param {http.Request} request - Consulta recibida.
 * @param {string[]} keys - Contrato cerrado de filtros admitidos.
 * @returns {error} Error del cliente sin repetir valores potencialmente sensibles.
 */
func ValidateQuery(request *http.Request, keys ...string) error {
	query, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return err
	}
	for key, values := range query {
		allowed := false
		for _, k := range keys {
			if key == k {
				allowed = true
				break
			}
		}
		if !allowed || len(values) != 1 || values[0] == "" {
			return errors.New("los parámetros de consulta no son válidos")
		}
	}
	return nil
}
