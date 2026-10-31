package catalog

import (
	"net/url"
	"strings"
)

/** geographyFilter exige parámetros únicos, sin aceptar filtros desconocidos. */
func geographyFilter(values url.Values, withProvince bool) (string, string, error) {
	for key, entries := range values {
		if len(entries) != 1 || key != "country" && (key != "province" || !withProvince) {
			return "", "", ErrInvalidFilter
		}
	}
	country, province := strings.TrimSpace(values.Get("country")), strings.TrimSpace(values.Get("province"))
	if country == "" || len(country) > 160 || withProvince && (province == "" || len(province) > 160) {
		return "", "", ErrInvalidFilter
	}
	return country, province, nil
}
