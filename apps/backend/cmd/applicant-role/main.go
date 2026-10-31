package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
)

/** main asigna un rol a una cuenta existente sin exponer esta operación en la API pública. */
func main() {
	idCard := flag.String("cedula", "", "cédula de la cuenta que se actualizará")
	role := flag.String("role", identity.RoleAnalyst, "rol: postulante o analista")
	flag.Parse()
	if *idCard == "" || (*role != identity.RoleApplicant && *role != identity.RoleAnalyst) {
		log.Fatal("Debe indicar -cedula y un rol válido: postulante o analista.")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL es obligatoria; aplique primero las migraciones.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatal("No se pudo conectar a PostgreSQL para asignar el rol.")
	}
	defer conn.Close(context.Background())
	rows, err := sqlcgen.New(conn).UpdateApplicantRoleByCedula(ctx, sqlcgen.UpdateApplicantRoleByCedulaParams{Cedula: *idCard, Role: *role})
	if err != nil {
		log.Fatal("No se pudo actualizar el rol; compruebe que la migración 00005 esté aplicada.")
	}
	if rows != 1 {
		log.Fatal("No existe una cuenta registrada con la cédula indicada.")
	}
	log.Println("Rol actualizado correctamente.")
}
