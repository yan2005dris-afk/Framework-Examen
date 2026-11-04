package postgres

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

// formFixture guarda los identificadores que reutilizan los casos de la prueba.
type formFixture struct {
	submission uuid.UUID
	ecuador    uuid.UUID
	colombia   uuid.UUID
	azuay      uuid.UUID
	pichincha  uuid.UUID
	cuenca     uuid.UUID
	quito      uuid.UUID
	documents  map[string]uuid.UUID
}

// catalogID busca un valor del catálogo geográfico sembrado por las migraciones.
func catalogID(t *testing.T, pool *pgxpool.Pool, kind, parent, value string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(context.Background(), `SELECT id FROM catalogs WHERE kind=$1 AND parent_key=$2 AND value=$3`, kind, parent, value).Scan(&id); err != nil {
		t.Fatalf("catálogo %s/%s/%s: %v", kind, parent, value, err)
	}
	return id
}

// insertDocument registra un PDF sin objeto en almacenamiento: basta para las claves foráneas.
func insertDocument(t *testing.T, pool *pgxpool.Pool, submissionID uuid.UUID, section int, record string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := pool.Exec(context.Background(), `INSERT INTO documents(id,submission_id,section,record_id,name,mime,size_bytes,sha256,storage_key)
		VALUES($1,$2,$3,$4,'respaldo.pdf','application/pdf',1024,repeat('a',64),$5)`, id, submissionID, section, record, "test/"+id.String()); err != nil {
		t.Fatalf("documento sección %d: %v", section, err)
	}
	return id
}

// seedForm guarda las nueve secciones completas de un borrador con respaldos válidos.
func seedForm(t *testing.T, pool *pgxpool.Pool, submissionID uuid.UUID) formFixture {
	t.Helper()
	f := formFixture{
		submission: submissionID,
		ecuador:    catalogID(t, pool, "countries", "", "Ecuador"),
		colombia:   catalogID(t, pool, "countries", "", "Colombia"),
		azuay:      catalogID(t, pool, "provinces", "EC", "Azuay"),
		pichincha:  catalogID(t, pool, "provinces", "EC", "Pichincha"),
		cuenca:     catalogID(t, pool, "cities", "EC/Azuay", "Cuenca"),
		quito:      catalogID(t, pool, "cities", "EC/Pichincha", "Quito"),
		documents: map[string]uuid.UUID{
			"cargo":       insertDocument(t, pool, submissionID, 4, ""),
			"grado-1":     insertDocument(t, pool, submissionID, 5, "grado-1"),
			"exp-1":       insertDocument(t, pool, submissionID, 6, "exp-1"),
			"pub-1":       insertDocument(t, pool, submissionID, 7, "pub-1"),
			"curso-1":     insertDocument(t, pool, submissionID, 8, "curso-1"),
			"grado-libre": insertDocument(t, pool, submissionID, 5, "grado-libre"),
		},
	}
	statements := []string{
		`INSERT INTO submission_sections(submission_id,section,saved_at) SELECT $1,s,now() FROM generate_series(1,9) s`,
		`INSERT INTO submission_personal(submission_id,first_name,last_name,gender,birth_date,nationality,marital_status,spouse_name,spouse_id_card)
			VALUES($1,'Ana','Pérez','Femenino','1985-04-12','Ecuatoriana','Casado/a','Luis Mora','1710034065')`,
		`INSERT INTO submission_additional(submission_id,has_relatives) VALUES($1,'Sí')`,
		`INSERT INTO submission_relatives(submission_id,position,record_key,name,relationship,institution,area,job_title)
			VALUES($1,0,'fam-1','Rosa Pérez','hermano/a','IESS','Riesgos','Analista'),($1,1,'','',  '', '', '', ''),($1,2,'','','','','','')`,
		`INSERT INTO submission_position(submission_id,eligible,verification_detail,has_organization,organization,document_id)
			VALUES($1,true,'Postulante afiliado al IESS · simulación','Sí','Federación Nacional',$2)`,
		`INSERT INTO submission_degrees(submission_id,position,record_key,level,institution,type,title,area,registry,registry_date,document_id)
			VALUES($1,0,'grado-1','Tercer Nivel de Grado','Universidad Central','Nacional','Economista','Economía','1005-15-123','2015-08-01',$2)`,
		`INSERT INTO submission_experiences(submission_id,position,record_key,type,job_title,institution,start_date,end_date,is_current,activities,functions,document_id)
			VALUES($1,0,'exp-1','Dirección','Gerente','Banco Pichincha','2016-01-04','2020-12-31',false,'Finanzas','Dirección financiera',$2),
			($1,1,'exp-2','General','Asesora','BIESS','2021-01-04',NULL,true,'Mercado de Capitales','',NULL)`,
		`INSERT INTO submission_publications_info(submission_id,has_publications) VALUES($1,'Sí')`,
		`INSERT INTO submission_publications(submission_id,position,record_key,title,type,medium,published_on,related,document_id)
			VALUES($1,0,'pub-1','Riesgo de crédito','Publicación','Revista','2019-05-20','Sí',$2)`,
		`INSERT INTO submission_trainings(submission_id,position,record_key,type,name,institution,start_date,end_date,hours,document_id)
			VALUES($1,0,'curso-1','Curso','Gestión de riesgos','IAEN','2022-03-01','2022-03-05',40,$2),($1,1,'curso-2','Taller','Borrador','',NULL,NULL,0,NULL)`,
		`INSERT INTO submission_disqualifications(submission_id,q1,q2,q3,q4,q5,q6,q7,q8,q9,q10,q11,q12,q13)
			VALUES($1,'Sí','No','No','No','No','No','Sí','No','No','No','No','No','')`,
		`INSERT INTO submission_removal_details(submission_id,institution,job_title,end_date,reason) VALUES($1,'Superintendencia','Vocal','2021-06-30','Reestructuración')`,
		`INSERT INTO submission_consents(submission_id,data_processing,truthfulness) VALUES($1,true,true)`,
	}
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	args := map[int][]any{
		4: {submissionID, f.documents["cargo"]},
		5: {submissionID, f.documents["grado-1"]},
		6: {submissionID, f.documents["exp-1"]},
		8: {submissionID, f.documents["pub-1"]},
		9: {submissionID, f.documents["curso-1"]},
	}
	for i, statement := range statements {
		values, ok := args[i]
		if !ok {
			values = []any{submissionID}
		}
		if _, err := tx.Exec(ctx, statement, values...); err != nil {
			t.Fatalf("sentencia %d: %v", i, err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO submission_contact(submission_id,country_id,province_id,city_id,mobile,secondary_email,main_street,number,secondary_street,area,reference,phone)
		VALUES($1,$2,$3,$4,'0991234567','ana.alterna@example.com','Av. Solano','2-45','Remigio Crespo','El Vergel','Frente al parque','(07) 282-1234')`,
		submissionID, f.ecuador, f.azuay, f.cuenca); err != nil {
		t.Fatalf("contacto: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("el formulario completo debe confirmarse: %v", err)
	}
	return f
}

// inTx ejecuta sentencias en una transacción que siempre se revierte y devuelve
// el primer error, incluido el del COMMIT, donde fallan las claves diferidas.
func inTx(t *testing.T, pool *pgxpool.Pool, commit bool, statements ...func(context.Context, *pgxpool.Pool) string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	for _, build := range statements {
		if _, err := tx.Exec(ctx, build(ctx, pool)); err != nil {
			return err
		}
	}
	if !commit {
		// SET CONSTRAINTS ALL IMMEDIATE adelanta las claves diferidas sin confirmar nada.
		_, err := tx.Exec(ctx, `SET CONSTRAINTS ALL IMMEDIATE`)
		return err
	}
	return tx.Commit(ctx)
}

// sql adapta una sentencia ya armada al formato de inTx.
func sql(statement string) func(context.Context, *pgxpool.Pool) string {
	return func(context.Context, *pgxpool.Pool) string { return statement }
}

// TestPostgresFormTablesIntegrity recorre las reglas que la base garantiza por sí misma.
func TestPostgresFormTablesIntegrity(t *testing.T) {
	_, item, pool := integrationEnvironment(t)
	f := seedForm(t, pool, item.ID)
	id := "'" + f.submission.String() + "'"
	quote := func(value uuid.UUID) string { return "'" + value.String() + "'" }

	rejected := []struct {
		name   string
		want   string
		update string
	}{
		{"datos sin sección guardada", "submission_degrees_submission_id_section_fkey",
			`DELETE FROM submission_sections WHERE submission_id=` + id + ` AND section=5; INSERT INTO submission_degrees(submission_id,position) VALUES(` + id + `,9)`},
		{"provincia de otro país", "submission_contact_province_id_country_id_fkey",
			`UPDATE submission_contact SET country_id=` + quote(f.colombia) + ` WHERE submission_id=` + id},
		{"cantón de otra provincia", "submission_contact_city_id_province_id_fkey",
			`UPDATE submission_contact SET city_id=` + quote(f.quito) + ` WHERE submission_id=` + id},
		{"provincia usada como país", "submission_contact_country_id_country_kind_fkey",
			`UPDATE submission_contact SET country_id=` + quote(f.azuay) + `, province_id=NULL, city_id=NULL WHERE submission_id=` + id},
		{"cantón sin provincia", "submission_contact_city_catalog",
			`UPDATE submission_contact SET province_id=NULL WHERE submission_id=` + id},
		{"provincia con catálogo y texto", "submission_contact_province_source",
			`UPDATE submission_contact SET province_name='Azuay' WHERE submission_id=` + id},
		{"provincia sin país", "submission_contact_province_country",
			`UPDATE submission_contact SET country_id=NULL, province_id=NULL, city_id=NULL, province_name='Antioquia' WHERE submission_id=` + id},
		// PostgreSQL trunca a 63 caracteres el nombre automático de esta clave.
		{"PDF de otra sección", "submission_degrees_document_id",
			`UPDATE submission_degrees SET document_id=` + quote(f.documents["exp-1"]) + ` WHERE submission_id=` + id},
		{"PDF de otro registro", "submission_degrees_document_id",
			`UPDATE submission_degrees SET document_id=` + quote(f.documents["grado-libre"]) + ` WHERE submission_id=` + id},
		{"detalle 1.9.7 sin respuesta Sí", "submission_removal_details_submission_id_q7_fkey",
			`UPDATE submission_disqualifications SET q7='No' WHERE submission_id=` + id},
		{"opción fuera del RF", "submission_personal_gender_check",
			`UPDATE submission_personal SET gender='Otro' WHERE submission_id=` + id},
		{"nacionalidad no ecuatoriana", "submission_personal_nationality_check",
			`UPDATE submission_personal SET nationality='Colombiana' WHERE submission_id=` + id},
		{"funciones de más de 1000 caracteres", "submission_experiences_functions_check",
			`UPDATE submission_experiences SET functions=repeat('x',1001) WHERE submission_id=` + id + ` AND position=0`},
		{"empleo actual con fecha de fin", "submission_experiences_current",
			`UPDATE submission_experiences SET end_date='2024-01-01' WHERE submission_id=` + id + ` AND position=1`},
		{"fin antes del inicio", "submission_trainings_range",
			`UPDATE submission_trainings SET end_date='2022-02-01' WHERE submission_id=` + id + ` AND position=0`},
		{"horas negativas", "submission_trainings_hours_check",
			`UPDATE submission_trainings SET hours=-1 WHERE submission_id=` + id + ` AND position=1`},
		{"identificador de registro repetido", "submission_degrees_record_key",
			`INSERT INTO submission_degrees(submission_id,position,record_key) VALUES(` + id + `,1,'grado-1')`},
		{"correo alternativo inválido", "submission_contact_secondary_email_check",
			`UPDATE submission_contact SET secondary_email='sin-arroba' WHERE submission_id=` + id},
	}
	for _, c := range rejected {
		t.Run(c.name, func(t *testing.T) {
			err := inTx(t, pool, false, sql(c.update))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("se esperaba %s: %v", c.want, err)
			}
		})
	}

	accepted := []struct {
		name   string
		update string
	}{
		// Borradores: identificadores pendientes repetidos y cero horas son válidos (Go los admite).
		{"registros sin identificador", `INSERT INTO submission_degrees(submission_id,position) VALUES(` + id + `,1),(` + id + `,2)`},
		// Otro país: provincia y ciudad escritas a mano.
		{"país extranjero con texto libre", `UPDATE submission_contact SET country_id=` + quote(f.colombia) + `, province_id=NULL, city_id=NULL, province_name='Antioquia', city_name='Medellín' WHERE submission_id=` + id},
		// Cambiar q7 a No y quitar el detalle en la misma transacción: la clave diferida lo admite.
		{"q7 a No quitando el detalle", `UPDATE submission_disqualifications SET q7='No' WHERE submission_id=` + id + `; DELETE FROM submission_removal_details WHERE submission_id=` + id},
		// Orden real de document.Service.Delete: el PDF se borra antes de limpiar la referencia.
		{"borrar PDF antes de soltar la referencia", `DELETE FROM documents WHERE id=` + quote(f.documents["grado-1"]) + `; UPDATE submission_degrees SET document_id=NULL WHERE submission_id=` + id + ` AND position=0`},
	}
	for _, c := range accepted {
		t.Run(c.name, func(t *testing.T) {
			if err := inTx(t, pool, true, sql(c.update)); err != nil {
				t.Fatalf("debe aceptarse: %v", err)
			}
		})
	}

	t.Run("quitar una sección borra sus datos", func(t *testing.T) {
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `DELETE FROM submission_sections WHERE submission_id=$1 AND section=6`, f.submission); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM submission_experiences WHERE submission_id=$1`, f.submission).Scan(&count); err != nil || count != 0 {
			t.Fatalf("experiencias tras quitar la sección: %d %v", count, err)
		}
	})

	t.Run("expediente enviado inmutable", func(t *testing.T) {
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `UPDATE submissions SET status='submitted',hash=repeat('b',64),submitted_at=now() WHERE id=$1`, f.submission); err != nil {
			t.Fatal(err)
		}
		for _, statement := range []string{
			`UPDATE submission_personal SET first_name='Otra' WHERE submission_id=` + id,
			`DELETE FROM submission_relatives WHERE submission_id=` + id,
			`INSERT INTO submission_sections(submission_id,section,saved_at) VALUES(` + id + `,6,now())`,
			`UPDATE submission_consents SET truthfulness=false WHERE submission_id=` + id,
		} {
			if err := inTx(t, pool, true, sql(statement)); err == nil || !strings.Contains(err.Error(), "El expediente no admite cambios") {
				t.Fatalf("%s: se esperaba rechazo por expediente enviado: %v", statement, err)
			}
		}
	})
}

// TestPostgresFormTablesCascadeOnDiscard comprueba que descartar un borrador no deja huérfanos.
func TestPostgresFormTablesCascadeOnDiscard(t *testing.T) {
	_, item, pool := integrationEnvironment(t)
	seedForm(t, pool, item.ID)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM submissions WHERE id=$1`, item.ID); err != nil {
		t.Fatalf("descartar borrador: %v", err)
	}
	var orphans int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM submission_sections)+(SELECT count(*) FROM submission_personal)+(SELECT count(*) FROM submission_contact)+
		(SELECT count(*) FROM submission_additional)+(SELECT count(*) FROM submission_relatives)+(SELECT count(*) FROM submission_position)+
		(SELECT count(*) FROM submission_degrees)+(SELECT count(*) FROM submission_experiences)+(SELECT count(*) FROM submission_publications_info)+
		(SELECT count(*) FROM submission_publications)+(SELECT count(*) FROM submission_trainings)+(SELECT count(*) FROM submission_disqualifications)+
		(SELECT count(*) FROM submission_removal_details)+(SELECT count(*) FROM submission_consents)+(SELECT count(*) FROM documents)`).Scan(&orphans); err != nil {
		t.Fatal(err)
	}
	if orphans != 0 {
		t.Fatalf("quedaron %d filas huérfanas", orphans)
	}
}

// TestPostgresFormChecksMatchDefinition impide que las listas cerradas de la base
// diverjan de definition.json, la fuente compartida con el frontend.
func TestPostgresFormChecksMatchDefinition(t *testing.T) {
	_, _, pool := integrationEnvironment(t)
	definition, err := submission.FormDefinition()
	if err != nil {
		t.Fatal(err)
	}
	columns := map[string]string{
		"submission_personal.gender":                    "generos",
		"submission_personal.marital_status":            "estadosCiviles",
		"submission_additional.has_relatives":           "respuestas",
		"submission_relatives.relationship":             "parentescos",
		"submission_relatives.institution":              "instituciones",
		"submission_position.has_organization":          "respuestas",
		"submission_degrees.level":                      "niveles",
		"submission_degrees.type":                       "tiposInstitucion",
		"submission_degrees.area":                       "areas",
		"submission_experiences.type":                   "tiposExperiencia",
		"submission_experiences.activities":             "areas",
		"submission_publications_info.has_publications": "respuestas",
		"submission_publications.type":                  "tiposPublicacion",
		"submission_publications.related":               "respuestas",
		"submission_trainings.type":                     "tiposEvento",
		"submissions.sector":                            "sectores",
	}
	for q := 1; q <= 13; q++ {
		columns[fmt.Sprintf("submission_disqualifications.q%d", q)] = "respuestas"
	}
	literal := regexp.MustCompile(`'((?:[^']|'')*)'`)
	for column, key := range columns {
		parts := strings.SplitN(column, ".", 2)
		var def string
		if err := pool.QueryRow(context.Background(), `SELECT pg_get_constraintdef(oid) FROM pg_constraint
			WHERE conrelid=$1::regclass AND contype='c' AND conname=$2`, parts[0], parts[0]+"_"+parts[1]+"_check").Scan(&def); err != nil {
			t.Fatalf("%s: CHECK no encontrado: %v", column, err)
		}
		got := []string{}
		for _, match := range literal.FindAllStringSubmatch(def, -1) {
			if value := strings.ReplaceAll(match[1], "''", "'"); value != "" {
				got = append(got, value)
			}
		}
		want := slices.Clone(definition.Options[key])
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s difiere de definition.json[%s]:\n base: %v\n  RF: %v", column, key, got, want)
		}
	}
}
