package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/app"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/config"
)

func TestAppNewWithoutDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := config.Config{
		Environment: "test",
		Port:        8080,
		Version:     "test-version",
		DatabaseURL: "", // Sin base de datos
	}

	application, err := app.New(ctx, cfg, app.ModeAPI)
	if err != nil {
		t.Fatalf("se esperaba que New sin BD no fallara, error: %v", err)
	}
	defer application.Close()

	if application.Router == nil {
		t.Fatal("el router no debería ser nil")
	}

	handler := application.Handler()
	if handler == nil {
		t.Fatal("el handler no debería ser nil")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200 en /healthz, obtenido: %d", rec.Code)
	}
}

func TestAppNewWorkerWithoutDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := config.Config{
		Environment: "test",
		DatabaseURL: "",
	}

	application, err := app.New(ctx, cfg, app.ModeWorker)
	if err != nil {
		t.Fatalf("se esperaba que New en ModeWorker sin BD no fallara, error: %v", err)
	}
	defer application.Close()

	// RunWorkers sin base de datos debe retornar de inmediato sin bloquear
	application.RunWorkers(ctx)
}

func TestAppNewDatabaseRequiredFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := config.Config{
		Environment:      "test",
		DatabaseURL:      "postgres://invalid:invalid@127.0.0.1:54329/invalid?sslmode=disable",
		DatabaseRequired: true,
	}

	_, err := app.New(ctx, cfg, app.ModeAPI)
	if err == nil {
		t.Fatal("se esperaba un error cuando DatabaseRequired=true y la BD no responde")
	}
}

func TestAppCloseIdempotent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := config.Config{Environment: "test"}
	application, err := app.New(ctx, cfg, app.ModeAPI)
	if err != nil {
		t.Fatalf("error inesperado en New: %v", err)
	}

	// Cerrar múltiples veces no debe causar pánico
	application.Close()
	application.Close()
}
