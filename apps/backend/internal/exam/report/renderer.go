package report

import (
	"bytes"
	"context"
	"fmt"
	"github.com/go-pdf/fpdf"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam"
	"golang.org/x/image/font/gofont/goregular"
	"time"
)

type Renderer struct{ mark string }

func NewRenderer() *Renderer { return &Renderer{} }

/** WithMark agrega la advertencia explícita del entorno sintético. */
func (r *Renderer) WithMark(mark string) *Renderer { r.mark = mark; return r }

/** Render no revela la matriz correcta: el reporte conserva nota y trazabilidad con bytes reproducibles. */
func (r *Renderer) Render(ctx context.Context, a *exam.Attempt) ([]byte, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if a == nil || a.Status != "finished" || a.Result == nil || a.FinishedAt == nil || a.Result.Score == nil || a.Result.Maximum == nil {
		return nil, exam.ErrInvalid
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetCatalogSort(true)
	pdf.SetCreationDate(a.FinishedAt.UTC())
	pdf.SetModificationDate(a.FinishedAt.UTC())
	pdf.AddUTF8FontFromBytes("go", "", goregular.TTF)
	pdf.SetTitle("Reporte de evaluación teórica", true)
	pdf.SetCreator("examen-reporte-v1", true)
	pdf.AddPage()
	pdf.SetFont("go", "", 12)
	advance := "No"
	if a.Result.CanAdvance {
		advance = "Sí"
	}
	lines := []string{"Evaluación teórica — reporte de resultado", "Versión del reporte: examen-reporte-v1", "Postulación: " + a.SubmissionID.String(), "Intento: " + a.ID.String(), "Resultado definitivo: " + a.Result.ID.String(), "Hash del envío: " + a.SnapshotHash, "Banco: " + a.Config.BankID.String(), "Hash del banco: " + a.BankHash, "Reglas: " + a.Config.ID.String(), "Hash de reglas: " + a.Config.Hash, "Inicio: " + a.StartedAt.Format(time.RFC3339Nano), "Vencimiento: " + a.ExpiresAt.Format(time.RFC3339Nano), "Cierre: " + a.FinishedAt.Format(time.RFC3339Nano), "Motivo: " + a.Reason, fmt.Sprintf("Preguntas: %d | Respondidas: %d | Aciertos: %d", len(a.Questions), len(a.Answers), a.CorrectCount), fmt.Sprintf("Puntaje: %d / %d", *a.Result.Score, *a.Result.Maximum), fmt.Sprintf("Umbral aprobado: %d | Puede avanzar: %s", a.Config.Rules.Threshold, advance), "Fórmula: máximo × aciertos / cantidad de preguntas.", "Redondeo: " + a.Config.Rules.Rounding, "No es una firma electrónica ni adjudicación de cargo. La matriz de respuestas no se publica en este reporte."}
	if r.mark != "" {
		lines = append([]string{r.mark}, lines...)
	}
	for _, line := range lines {
		pdf.MultiCell(174, 7, line, "", "L", false)
	}
	var out bytes.Buffer
	if e := pdf.Output(&out); e != nil {
		return nil, e
	}
	if out.Len() > 4<<20 {
		return nil, exam.ErrInvalid
	}
	return out.Bytes(), nil
}
