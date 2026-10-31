package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/sql/migrations"
)

/**
 * main ejecuta las migraciones pendientes sobre la base de datos configurada.
 */
func main() {
	down := flag.Bool("down", false, "revierte la última migración aplicada")
	status := flag.Bool("status", false, "muestra el estado de las migraciones")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL es obligatoria para ejecutar las migraciones")
	}

	ctx := context.Background()

	var err error
	switch {
	case *down:
		err = migrations.Down(ctx, databaseURL)
	case *status:
		err = migrations.Status(ctx, databaseURL)
	default:
		err = migrations.Run(ctx, databaseURL)
	}

	if err != nil {
		log.Fatalf("Error en las migraciones: %v", err)
	}

	log.Println("Migraciones ejecutadas correctamente.")
}
