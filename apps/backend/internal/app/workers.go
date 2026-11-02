package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/document"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/telemetry"
)

/**
 * RunWorkers levanta las colas en segundo plano y no retorna hasta que el
 * contexto se cancela: el proceso vive mientras haya trabajo pendiente.
 *
 * Todas son seguras con varias réplicas: la cola de correo reclama con
 * lease y SKIP LOCKED, las colas de borrado son idempotentes y el cierre de
 * exámenes serializa la fila con FOR UPDATE.
 *
 * @param {context.Context} ctx - Contexto que cancela los bucles.
 */
func (a *App) RunWorkers(ctx context.Context) {
	if a.Pool == nil {
		slog.Warn("sin base de datos: no hay workers que ejecutar")
		return
	}
	if a.otpErr != nil {
		slog.Error("configuración OTP no válida: los controles de identidad quedan bloqueados", "error", a.otpErr)
	}

	go purgeSessions(ctx, a.queries)
	go a.outbox.Run(ctx, a.deliverySender, a.otpStore.Active)
	go cleanupEvaluationUploads(ctx, a.evaluation)
	go expireExams(ctx, a.exams)
	go cleanupDocuments(ctx, a.documents)

	<-ctx.Done()
}

/**
 * purgeSessions retira las sesiones vencidas cada hora.
 *
 * @param {context.Context} ctx - Contexto de los bucles.
 * @param {sqlcgen.Queries} q - Consultas tipadas.
 */
func purgeSessions(ctx context.Context, q *sqlcgen.Queries) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		run, cancel := context.WithTimeout(ctx, 10*time.Second)
		started := time.Now()
		_, err := q.PurgeExpiredSessionBatch(run)
		cancel()
		telemetry.Default.Observe("cleanup", "sessions", telemetry.Result(err), time.Since(started))
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "purga de sesiones pendiente")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

/**
 * cleanupDocuments procesa bajas al iniciar y cada minuto sin registrar
 * claves privadas.
 *
 * @param {context.Context} ctx - Contexto de los bucles.
 * @param {document.Service} service - Servicio con cola persistente e idempotente.
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

/**
 * cleanupEvaluationUploads retira los intentos de respaldo vencidos.
 *
 * @param {context.Context} ctx - Contexto de los bucles.
 * @param {evaluation.Service} service - Servicio de evaluación.
 */
func cleanupEvaluationUploads(ctx context.Context, service *evaluation.Service) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		pass, cancel := context.WithTimeout(ctx, 30*time.Second)
		start := time.Now()
		err := service.CleanupUploads(pass)
		telemetry.Default.Observe("cleanup", "uploads", telemetry.Result(err), time.Since(start))
		if err != nil && ctx.Err() == nil {
			slog.Warn("hay respaldos vencidos pendientes de retirar; se reintentará")
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

/**
 * expireExams recupera cierres vencidos tras reinicio; el plazo lo aplica
 * PostgreSQL aunque el navegador esté cerrado.
 *
 * @param {context.Context} ctx - Contexto de los bucles.
 * @param {exam.Repository} repository - Repositorio de exámenes.
 */
func expireExams(ctx context.Context, repository exam.Repository) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		run, cancel := context.WithTimeout(ctx, 20*time.Second)
		start := time.Now()
		err := repository.Expire(run)
		cancel()
		telemetry.Default.Observe("examen", "vencimiento", telemetry.Result(err), time.Since(start))
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
