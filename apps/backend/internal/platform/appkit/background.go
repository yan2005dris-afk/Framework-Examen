package appkit

import (
	"context"
	"log/slog"
	"time"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/telemetry"
)

/**
 * cleanupEvaluationUploads conserva los intentos fallidos hasta poder borrar sus objetos, sin tocar evidencias confirmadas.
 */
func cleanupEvaluationUploads(ctx context.Context, service *evaluation.Service) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		start := time.Now()
		run, cancel := context.WithTimeout(ctx, 20*time.Second)
		err := service.CleanupUploads(run)
		cancel()
		telemetry.Default.Observe("cleanup", "uploads", telemetry.Result(err), time.Since(start))
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "no se pudieron retirar intentos de respaldo vencidos")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

/**
 * expireExams recupera cierres vencidos tras reinicio; el plazo lo aplica PostgreSQL aunque el navegador esté cerrado.
 */
func expireExams(ctx context.Context, r exam.Repository) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		run, cancel := context.WithTimeout(ctx, 20*time.Second)
		started := time.Now()
		err := r.Expire(run)
		cancel()
		telemetry.Default.Observe("cleanup", "expire_exams", telemetry.Result(err), time.Since(started))
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "no se pudieron confirmar cierres de evaluación pendientes")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

/**
 * cleanupDocuments retira los documentos vencidos que ya no protegen una evidencia.
 */
func cleanupDocuments(ctx context.Context, service *document.Service) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		pass, cancel := context.WithTimeout(ctx, 30*time.Second)
		start := time.Now()
		err := service.Cleanup(pass)
		telemetry.Default.Observe("cleanup", "objects", telemetry.Result(err), time.Since(start))
		if err != nil && ctx.Err() == nil {
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
