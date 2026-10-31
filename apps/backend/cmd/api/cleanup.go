package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
)

/** cleanupDocuments procesa bajas al iniciar y cada minuto sin registrar claves privadas.
 * @param {Service} service - Servicio con cola persistente e idempotente.
 */
func cleanupDocuments(ctx context.Context, service *document.Service) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		pass, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := service.Cleanup(pass); err != nil && ctx.Err() == nil {
			slog.Warn("hay objetos pendientes de eliminar; se reintentará")
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
