// Package print representa exclusivamente el contenido sellado del EF-12.
package print

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-pdf/fpdf"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"image"
	_ "image/jpeg"
	"image/png"
	"sort"
	"strings"
	"time"
	"unicode"
)

const Version = "ef12-print-v1"
const maxPages = 200

type Renderer struct{}

func NewRenderer() *Renderer { return &Renderer{} }

/** Render devuelve PDF reproducible: los metadatos temporales proceden del envío y no del reloj actual. */
func (*Renderer) Render(ctx context.Context, item *submission.Submission) ([]byte, error) {
	if err := submission.ValidateSeal(item); err != nil {
		return nil, err
	}
	if len(item.Content) > 8<<20 {
		return nil, submission.ErrPrintTooLarge
	}
	var content map[string]interface{}
	d := json.NewDecoder(bytes.NewReader(item.Content))
	d.UseNumber()
	if err := d.Decode(&content); err != nil {
		return nil, submission.ErrInvalidSeal
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(17, 24, 17)
	pdf.SetAutoPageBreak(true, 22)
	pdf.SetCatalogSort(true)
	pdf.SetCreationDate(item.SubmittedAt.UTC())
	pdf.SetModificationDate(item.SubmittedAt.UTC())
	pdf.SetCompression(true)
	pdf.AddUTF8FontFromBytes("go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("go", "B", gobold.TTF)
	pdf.SetTitle("EF-12 - "+clean(item.Code), true)
	pdf.SetAuthor("Portal de postulación", true)
	pdf.SetCreator(Version, true)
	pdf.AliasNbPages("")
	pdf.SetHeaderFunc(func() {
		pdf.SetFont("go", "B", 9)
		pdf.SetXY(17, 10)
		pdf.CellFormat(176, 5, "EF-12 | Copia del formulario enviado", "", 1, "L", false, 0, "")
		pdf.SetDrawColor(180, 185, 195)
		pdf.Line(17, 18, 193, 18)
		pdf.SetY(24)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("go", "", 8)
		pdf.CellFormat(176, 5, fmt.Sprintf("%s | Página %d/{nb}", clean(item.Code), pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	pdf.SetAcceptPageBreakFunc(func() bool {
		if ctx.Err() != nil {
			pdf.SetError(ctx.Err())
			return false
		}
		if pdf.PageNo() >= maxPages {
			pdf.SetError(submission.ErrPrintTooLarge)
			return false
		}
		return true
	})
	pdf.AddPage()
	pdf.SetFont("go", "B", 16)
	pdf.MultiCell(176, 8, "Formulario de postulación", "", "L", false)
	pdf.SetFont("go", "", 9)
	period := "Legado sin convocatoria"
	if item.CompetitionID != nil {
		period = item.CompetitionID.String()
	}
	for _, line := range []string{"Código: " + item.Code, "Convocatoria de origen: " + period, "Enviado (UTC): " + item.SubmittedAt.UTC().Format(time.RFC3339Nano), "SHA-256 del contenido: " + item.Hash, "Contenido declarado por el postulante. El sello no certifica identidad civil ni constituye calificación de méritos."} {
		pdf.MultiCell(176, 5, clean(line), "", "L", false)
	}
	titles := []string{"Información personal", "Información domiciliaria y de contacto", "Información adicional", "Selección del cargo", "Formación académica", "Experiencia profesional", "Publicaciones e investigaciones", "Capacitación", "Inhabilidades"}
	for i, title := range titles {
		heading(pdf, fmt.Sprintf("%d. %s", i+1, title))
		writeValue(ctx, pdf, content[fmt.Sprintf("seccion%d", i+1)], "", 0)
	}
	heading(pdf, "Declaraciones y consentimiento del envío")
	writeValue(ctx, pdf, content["consentimiento"], "", 0)
	extra := map[string]interface{}{}
	for key, value := range content {
		known := key == "consentimiento"
		for i := 1; i <= 9; i++ {
			known = known || key == fmt.Sprintf("seccion%d", i)
		}
		if !known {
			extra[key] = value
		}
	}
	if len(extra) > 0 {
		heading(pdf, "Otros datos conservados en el envío")
		writeValue(ctx, pdf, extra, "", 0)
	}
	heading(pdf, "Entrega física con firma manuscrita")
	pdf.SetFont("go", "", 9)
	pdf.MultiCell(176, 5, "Imprima esta copia y entréguela con firma manuscrita conforme a las condiciones publicadas del concurso. No se requiere una firma digital dibujada para generar esta copia.", "", "L", false)
	pdf.Ln(12)
	pdf.CellFormat(176, 7, "Firma manuscrita: ______________________________________", "", 1, "L", false, 0, "")
	pdf.CellFormat(176, 7, "Fecha de entrega: ______________________________________", "", 1, "L", false, 0, "")
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	if out.Len() > 16<<20 {
		return nil, submission.ErrPrintTooLarge
	}
	return out.Bytes(), nil
}
func heading(pdf *fpdf.Fpdf, text string) {
	if pdf.GetY() > 255 {
		pdf.AddPage()
	}
	pdf.Ln(5)
	pdf.SetFont("go", "B", 11)
	pdf.MultiCell(176, 7, clean(text), "", "L", false)
	pdf.SetFont("go", "", 9)
}

var displayFont, _ = sfnt.Parse(goregular.TTF)

/** clean conserva texto seguro; los caracteres sin glifo se representan como código Unicode, nunca en blanco. */
func clean(s string) string {
	var out strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			out.WriteByte(' ')
			continue
		}
		if r != '\n' && r != '\t' && displayFont != nil {
			glyph, err := displayFont.GlyphIndex(nil, r)
			if err != nil || glyph == 0 {
				fmt.Fprintf(&out, "[U+%04X]", r)
				continue
			}
		}
		out.WriteRune(r)
	}
	return out.String()
}
func ensureSpace(pdf *fpdf.Fpdf, height float64) {
	if pdf.GetY()+height > 275 {
		if pdf.PageNo() >= maxPages {
			pdf.SetError(submission.ErrPrintTooLarge)
			return
		}
		pdf.AddPage()
	}
}
func label(key string) string {
	if value, ok := labels[key]; ok {
		return value
	}
	var b strings.Builder
	for i, r := range key {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte(' ')
		}
		if i == 0 {
			r = unicode.ToUpper(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}
func writeValue(ctx context.Context, pdf *fpdf.Fpdf, value interface{}, key string, depth int) {
	if ctx.Err() != nil {
		pdf.SetError(ctx.Err())
		return
	}
	if depth > 20 {
		pdf.SetError(submission.ErrPrintTooLarge)
		return
	}
	width := 176 - float64(min(depth, 6))*3
	if key == "firmaManuscritaDataUrl" {
		writeSignature(pdf, value)
		return
	}
	switch v := value.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			a, aok := questionOrder(keys[i])
			b, bok := questionOrder(keys[j])
			if aok && bok {
				return a < b
			}
			if aok != bok {
				return aok
			}
			return keys[i] < keys[j]
		})
		if len(keys) == 0 {
			pdf.MultiCell(width, 5, "Sin datos conservados.", "", "L", false)
		}
		for _, k := range keys {
			pdf.SetFont("go", "B", 9)
			ensureSpace(pdf, float64(len(pdf.SplitText(clean(label(k))+":", width)))*5+5)
			pdf.MultiCell(width, 5, clean(label(k))+":", "", "L", false)
			pdf.SetFont("go", "", 9)
			writeValue(ctx, pdf, v[k], k, depth+1)
		}
	case []interface{}:
		if len(v) == 0 {
			pdf.MultiCell(width, 5, "Sin registros.", "", "L", false)
		}
		for i, item := range v {
			ensureSpace(pdf, 15)
			pdf.SetFont("go", "B", 9)
			pdf.MultiCell(width, 5, fmt.Sprintf("Registro %d", i+1), "", "L", false)
			pdf.SetFont("go", "", 9)
			writeValue(ctx, pdf, item, key, depth+1)
			pdf.Ln(2)
		}
	case nil:
		pdf.MultiCell(width, 5, "No consta en el envío.", "", "L", false)
	case bool:
		text := "No"
		if v {
			text = "Sí"
		}
		pdf.MultiCell(width, 5, text, "", "L", false)
	default:
		text := fmt.Sprint(value)
		if text == "" {
			text = "No consta en el envío."
		}
		pdf.MultiCell(width, 5, clean(text), "", "L", false)
	}
}
func writeSignature(pdf *fpdf.Fpdf, value interface{}) {
	s, ok := value.(string)
	if !ok || s == "" {
		pdf.MultiCell(176, 5, "Sin firma dibujada en el envío; firma manuscrita al entregar.", "", "L", false)
		return
	}
	format := ""
	for _, kind := range []string{"png", "jpeg"} {
		prefix := "data:image/" + kind + ";base64,"
		if strings.HasPrefix(s, prefix) {
			format = kind
			s = strings.TrimPrefix(s, prefix)
			break
		}
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if format == "" || err != nil || len(raw) > 2<<20 {
		pdf.MultiCell(176, 5, "Firma legada conservada en un formato no visualizable.", "", "L", false)
		return
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width*config.Height > 4_000_000 {
		pdf.MultiCell(176, 5, "Firma legada no visualizable.", "", "L", false)
		return
	}
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		pdf.MultiCell(176, 5, "Firma legada no visualizable.", "", "L", false)
		return
	}
	var normalized bytes.Buffer
	if err = png.Encode(&normalized, decoded); err != nil {
		pdf.MultiCell(176, 5, "Firma legada no visualizable.", "", "L", false)
		return
	}
	ensureSpace(pdf, 30)
	options := fpdf.ImageOptions{ImageType: "png", ReadDpi: false}
	pdf.RegisterImageOptionsReader("firma", options, bytes.NewReader(normalized.Bytes()))
	pdf.ImageOptions("firma", 17, pdf.GetY(), 60, 25, false, options, 0, "")
	pdf.Ln(28)
}

func questionOrder(key string) (int, bool) {
	if key == "q7Detalles" {
		return 15, true
	}
	for n := 1; n <= 13; n++ {
		if key == fmt.Sprintf("q%d", n) {
			return n * 2, true
		}
	}
	return 0, false
}
