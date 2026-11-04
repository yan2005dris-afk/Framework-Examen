package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
)

// formTables son las tablas de 00024 en orden de dependencia.
var formTables = []string{
	"submission_sections", "submission_personal", "submission_contact", "submission_additional", "submission_relatives",
	"submission_position", "submission_degrees", "submission_experiences", "submission_publications_info", "submission_publications",
	"submission_trainings", "submission_disqualifications", "submission_removal_details", "submission_consents",
}

/** BackfillReport resume el relleno sin datos personales: sólo cantidades e identificadores. */
type BackfillReport struct {
	Drafts  int
	Sealed  int
	Skipped []uuid.UUID
}

/**
 * BackfillForms descompone el contenido de todas las postulaciones en las tablas
 * normalizadas, dentro de la transacción recibida.
 *
 * Los borradores deben leerse desde las tablas igual que desde `content`: si uno
 * no cumple, el relleno falla y lista sus UUID. Las postulaciones enviadas o
 * cerradas siguen leyéndose de su snapshot; si una no se puede descomponer se
 * omite y se informa. `submissions` no se modifica: sólo se desactivan, dentro de
 * la transacción, los triggers de ventana de las tablas nuevas, porque los
 * expedientes enviados o de convocatorias cerradas ya no admiten escrituras.
 *
 * @param {pgx.Tx} tx - Transacción del relleno; un error la deja para rollback.
 * @param {time.Time} now - Fecha de corte de las duraciones al comparar.
 * @returns {BackfillReport, error} Resumen o la lista de borradores inconsistentes.
 */
func BackfillForms(ctx context.Context, tx pgx.Tx, now time.Time) (BackfillReport, error) {
	report := BackfillReport{}
	// Sin escrituras concurrentes mientras se copia; las lecturas siguen.
	if _, err := tx.Exec(ctx, `LOCK TABLE submissions IN EXCLUSIVE MODE`); err != nil {
		return report, err
	}
	for _, table := range formTables {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s DISABLE TRIGGER %s_window`, table, table)); err != nil {
			return report, err
		}
	}
	rows, err := tx.Query(ctx, `SELECT id, status, content, updated_at FROM submissions ORDER BY id`)
	if err != nil {
		return report, err
	}
	type pending struct {
		id      uuid.UUID
		status  string
		content json.RawMessage
		updated time.Time
	}
	items := []pending{}
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.status, &p.content, &p.updated); err != nil {
			rows.Close()
			return report, err
		}
		items = append(items, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return report, err
	}
	repo := &Repository{queries: sqlcgen.New(tx)}
	failed := []string{}
	for _, p := range items {
		editable := (&submission.Submission{Status: submission.Status(p.status)}).IsEditable()
		// Desde 00027 un borrador sin content ya vive sólo en las tablas.
		if editable && p.content == nil {
			report.Drafts++
			continue
		}
		err := backfillOne(ctx, tx, repo, p.id, p.content, p.updated, editable, now)
		switch {
		case err == nil && editable:
			report.Drafts++
		case err == nil:
			report.Sealed++
		case editable:
			failed = append(failed, fmt.Sprintf("%s (%v)", p.id, err))
		default:
			report.Skipped = append(report.Skipped, p.id)
		}
	}
	if len(failed) > 0 {
		return report, fmt.Errorf("borradores que no se pueden normalizar: %s", strings.Join(failed, "; "))
	}
	for _, table := range formTables {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s ENABLE TRIGGER %s_window`, table, table)); err != nil {
			return report, err
		}
	}
	return report, nil
}

/**
 * backfillOne reescribe las tablas de una postulación dentro de un savepoint: si
 * falla, se descartan sólo sus filas. En borradores verifica la lectura.
 */
func backfillOne(ctx context.Context, tx pgx.Tx, repo *Repository, id uuid.UUID, content json.RawMessage, updated time.Time, verify bool, now time.Time) error {
	// Un contenido ajeno al contrato no tiene secciones que escribir, pero no está normalizado.
	if _, err := submission.DecodeForm(content); err != nil {
		return err
	}
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		return err
	}
	defer savepoint.Rollback(ctx)
	scoped := &Repository{queries: repo.queries.WithTx(savepoint)}
	if _, err := savepoint.Exec(ctx, `DELETE FROM submission_sections WHERE submission_id = $1`, id); err != nil {
		return err
	}
	if _, err := savepoint.Exec(ctx, `DELETE FROM submission_consents WHERE submission_id = $1`, id); err != nil {
		return err
	}
	item := &submission.Submission{ID: id, Content: content, UpdatedAt: updated}
	if err := scoped.writeForm(ctx, item, nil); err != nil {
		return err
	}
	// Comprueba las claves diferidas de esta postulación antes de seguir con la siguiente.
	if _, err := savepoint.Exec(ctx, `SET CONSTRAINTS ALL IMMEDIATE`); err != nil {
		return err
	}
	if _, err := savepoint.Exec(ctx, `SET CONSTRAINTS ALL DEFERRED`); err != nil {
		return err
	}
	if verify {
		want, err := scoped.expectedForm(ctx, id, content, now)
		if err != nil {
			return err
		}
		got, err := scoped.loadForm(ctx, id, now)
		if err != nil {
			return err
		}
		if canonicalText(want) != canonicalText(got) {
			return fmt.Errorf("la lectura desde las tablas no coincide con content")
		}
	}
	return savepoint.Commit(ctx)
}

/**
 * expectedForm es lo que la lectura desde las tablas debe devolver para un
 * contenido: el formulario con sus datos derivados tomados de la fuente, como
 * hace loadForm, y los respaldos que no pertenecen al registro descartados.
 */
func (r *Repository) expectedForm(ctx context.Context, id uuid.UUID, content json.RawMessage, now time.Time) (*submission.Form, error) {
	form, err := submission.DecodeForm(content)
	if err != nil {
		return nil, err
	}
	who, err := r.queries.GetFormIdentity(ctx, id)
	if err != nil {
		return nil, err
	}
	documents, err := r.queries.ListFormDocuments(ctx, id)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]formDocument, len(documents))
	byID := make(map[uuid.UUID]*submission.DocumentReference, len(documents))
	for _, d := range documents {
		keys[d.ID.String()] = formDocument{Section: d.Section, Record: d.RecordID}
		byID[d.ID] = submission.NewDocumentReference(d.ID, d.Name, d.SizeBytes, d.Sha256, d.CreatedAt)
	}
	resolve := func(section int32, record string, ref *submission.DocumentReference) *submission.DocumentReference {
		if id := documentField(keys, section, record, ref); id.Valid {
			return byID[uuid.UUID(id.Bytes)]
		}
		return nil
	}
	rows := formToRows(form)
	if rows.Position != nil {
		rows.Position.Document = resolve(4, "", rows.Position.Document)
	}
	for i := range rows.Degrees {
		rows.Degrees[i].Document = resolve(5, rows.Degrees[i].ID, rows.Degrees[i].Document)
	}
	for i := range rows.Experiences {
		rows.Experiences[i].Document = resolve(6, rows.Experiences[i].ID, rows.Experiences[i].Document)
	}
	for i := range rows.Publications {
		rows.Publications[i].Document = resolve(7, rows.Publications[i].ID, rows.Publications[i].Document)
	}
	for i := range rows.Trainings {
		rows.Trainings[i].Document = resolve(8, rows.Trainings[i].ID, rows.Trainings[i].Document)
	}
	expected := rowsToForm(rows, formIdentity{IDCard: who.Cedula, Email: who.Correo, Sector: who.Sector})
	submission.NormalizeDurations(expected, now)
	return expected, nil
}

/** canonicalText serializa un formulario con las reglas del sello para compararlo. */
func canonicalText(form *submission.Form) string {
	raw, err := json.Marshal(form)
	if err != nil {
		return ""
	}
	canonical, err := submission.CanonicalContent(raw)
	if err != nil {
		return ""
	}
	return string(canonical)
}
