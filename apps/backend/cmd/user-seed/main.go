package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed seed.sql
var seedSQL string

/** main inyecta o actualiza las cuentas de prueba para desarrollo local. */
func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres@localhost:5432/biess?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("No se pudo conectar a PostgreSQL (%s): %v", databaseURL, err)
	}
	defer conn.Close(context.Background())

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("No se pudo iniciar transacción: %v", err)
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(ctx, seedSQL); err != nil {
		log.Fatalf("Error al ejecutar semilla de usuarios: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("Error al confirmar transacción: %v", err)
	}

	fmt.Println("Semilla de usuarios inyectada exitosamente:")
	fmt.Println("  - Postulante:    1710034008 / postulante@biess.local (Password.2026*)")
	fmt.Println("  - Analista:      1710034016 / analista@biess.local   (Password.2026*)")
	fmt.Println("  - Administrador: 0910034008 / admin@biess.local      (Password.2026*)")
}
