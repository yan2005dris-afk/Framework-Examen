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
	role := flag.String("role", identity.RoleAnalyst, "rol: postulante, analista o administrador")
	operator := flag.String("operator", "", "identificador del operador responsable")
	reason := flag.String("reason", "", "motivo del cambio de rol")
	flag.Parse()
	if _, err := identity.NewIDCard(*idCard); err != nil || !identity.ValidRoleChange(*role, *reason) || !identity.ValidRoleChange(*role, *operator) {
		log.Fatal("Indique -cedula válida, -role (postulante, analista o administrador), -operator y -reason.")
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
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatal("No se pudo iniciar el cambio de rol.")
	}
	defer tx.Rollback(context.Background())
	queries := sqlcgen.New(tx)
	if err := queries.SetRoleChangeContext(ctx, sqlcgen.SetRoleChangeContextParams{Operator: *operator, Reason: *reason}); err != nil {
		log.Fatal("No se pudo preparar la auditoría del cambio de rol.")
	}
	rows, err := queries.UpdateApplicantRoleByCedula(ctx, sqlcgen.UpdateApplicantRoleByCedulaParams{Cedula: *idCard, Role: *role})
	if err != nil {
		log.Fatal("No se pudo actualizar el rol; compruebe que la migración 00007 esté aplicada.")
	}
	if rows != 1 {
		log.Fatal("No existe una cuenta registrada con la cédula indicada.")
	}
	if err := tx.Commit(ctx); err != nil {
		log.Fatal("No se pudo confirmar el cambio de rol y su auditoría.")
	}
	log.Println("Rol actualizado correctamente.")
}
