package document

import "time"

/** DownloadResponse contiene una capacidad temporal; no debe cachearse ni registrarse. */
type DownloadResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}
