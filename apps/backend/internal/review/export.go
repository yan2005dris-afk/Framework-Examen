package review

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"strconv"
	"strings"
	"time"
)

var matrixHeaders = []string{"Código", "Postulante", "Cargo / sector", "Envío (UTC)", "Formación declarada", "Experiencia por registro", "Publicaciones", "Horas capacitación"}

func matrixRow(p Applicant) []string {
	education := []string{}
	for _, e := range p.Education {
		education = append(education, e.Level+": "+e.Title)
	}
	experience := []string{}
	for _, e := range p.Experiences {
		duration := "Sin duración"
		if e.Duration != nil {
			duration = fmt.Sprintf("%d años, %d meses, %d días", e.Duration.Years, e.Duration.Months, e.Duration.Days)
		}
		experience = append(experience, e.ID+" · "+e.Type+" · "+e.Position+": "+duration)
	}
	return []string{p.Code, strings.TrimSpace(p.FirstName + " " + p.LastName), p.Sector, p.SubmittedAt.UTC().Format("2006-01-02 15:04"),
		strings.Join(education, "\n"), strings.Join(experience, "\n"), strconv.Itoa(p.Publications), strconv.Itoa(p.TrainingHours)}
}

func matrixMetadata(items []Applicant, f Filter, now time.Time) []string {
	period := "SIN-CONVOCATORIA (legado)"
	if f.CompetitionID != nil {
		period = f.CompetitionID.String()
	}
	if len(items) > 0 && f.CompetitionID != nil {
		period = items[0].Competition + " · " + period
	}
	sector := f.Sector
	if sector == "" {
		sector = "todos"
	}
	return []string{MatrixDescription(), "Convocatoria: " + period + " · Sector: " + sector + " · Estado: submitted",
		fmt.Sprintf("Consulta: %s · Expedientes: %d · Sin sumar períodos superpuestos ni calcular méritos", now.UTC().Format(time.RFC3339), len(items))}
}

func renderMatrix(ctx context.Context, items []Applicant, f Filter, now time.Time, format string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	metadata := matrixMetadata(items, f, now)
	if format == "xlsx" {
		return renderXLSX(ctx, items, metadata)
	}
	return renderPDF(ctx, items, metadata, now)
}

/** xmlText produce texto válido de OOXML; toda celda es inlineStr, nunca fórmula ejecutable. */
func xmlText(value string) string {
	value = strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' || r >= 0x20 && r <= 0xd7ff || r >= 0xe000 && r <= 0xfffd || r >= 0x10000 && r <= 0x10ffff {
			return r
		}
		return -1
	}, value)
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(value))
	return escaped.String()
}

/** renderXLSX crea un libro OOXML sin macros, fórmulas ni dependencias de Office.
 * La hoja adicional conserva cada experiencia y su hash de expediente sin celdas gigantes.
 */
func renderXLSX(ctx context.Context, items []Applicant, metadata []string) ([]byte, error) {
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	files := []struct{ name, body string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Postulantes" sheetId="1" r:id="rId1"/><sheet name="Experiencias" sheetId="2" r:id="rId2"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`},
		{"xl/styles.xml", `<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font><sz val="10"/><name val="Calibri"/></font></fonts><fills count="1"><fill><patternFill patternType="none"/></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs><cellXfs count="1"><xf fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment vertical="top" wrapText="1"/></xf></cellXfs></styleSheet>`},
	}
	mainRows := [][]string{}
	for _, line := range metadata {
		mainRows = append(mainRows, []string{line})
	}
	mainRows = append(mainRows, []string{}, matrixHeaders)
	detailRows := [][]string{{MatrixDescription()}, {"Código", "ID experiencia", "Tipo", "Cargo", "Años", "Meses", "Días", "SHA-256 expediente"}}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		values := matrixRow(item)
		// Los detalles están normalizados en la segunda hoja; sin límite oculto de 32767 caracteres.
		values[5] = fmt.Sprintf("%d registros; ver hoja Experiencias", len(item.Experiences))
		mainRows = append(mainRows, values)
		for _, e := range item.Experiences {
			years, months, days := "", "", ""
			if e.Duration != nil {
				years = strconv.Itoa(e.Duration.Years)
				months = strconv.Itoa(e.Duration.Months)
				days = strconv.Itoa(e.Duration.Days)
			}
			detailRows = append(detailRows, []string{item.Code, e.ID, e.Type, e.Position, years, months, days, item.Hash})
		}
	}
	for _, sheet := range []struct {
		name string
		rows [][]string
	}{{"xl/worksheets/sheet1.xml", mainRows}, {"xl/worksheets/sheet2.xml", detailRows}} {
		var body strings.Builder
		body.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><cols><col min="1" max="8" width="28" customWidth="1"/></cols><sheetData>`)
		for i, row := range sheet.rows {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			fmt.Fprintf(&body, `<row r="%d">`, i+1)
			for j, value := range row {
				// Excel limita cada celda a 32767 unidades UTF-16. Fallar es preferible a truncar evidencia.
				units := 0
				for _, r := range value {
					units++
					if r > 0xffff {
						units++
					}
				}
				if units > 32767 {
					return nil, ErrExportTooLarge
				}
				fmt.Fprintf(&body, `<c r="%c%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, rune('A'+j), i+1, xmlText(value))
			}
			body.WriteString(`</row>`)
		}
		body.WriteString(`</sheetData></worksheet>`)
		files = append(files, struct{ name, body string }{sheet.name, body.String()})
	}
	for _, file := range files {
		entry, err := writer.Create(file.name)
		if err != nil {
			return nil, err
		}
		if _, err = entry.Write([]byte(file.body)); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

/** renderPDF repite encabezados y divide filas extensas sin perder texto entre páginas. */
func renderPDF(ctx context.Context, items []Applicant, metadata []string, now time.Time) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(false, 12)
	pdf.SetCellMargin(0)
	pdf.SetCreationDate(now)
	pdf.SetModificationDate(now)
	pdf.AddUTF8FontFromBytes("go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("go", "B", gobold.TTF)
	widths := []float64{28, 40, 23, 25, 38, 72, 24, 27}
	const lineHeight = 3.5
	addPage := func() {
		pdf.AddPage()
		pdf.SetFont("go", "", 8)
		for _, line := range metadata {
			pdf.MultiCell(277, 4, line, "", "L", false)
		}
		pdf.Ln(2)
		pdf.SetFont("go", "B", 8)
		x, y := 10.0, pdf.GetY()
		headerHeight := 10.0
		for i, header := range matrixHeaders {
			headerHeight = max(headerHeight, float64(len(pdf.SplitText(header, widths[i]-2)))*4+2)
		}
		for i, header := range matrixHeaders {
			pdf.SetFillColor(237, 241, 245)
			pdf.Rect(x, y, widths[i], headerHeight, "DF")
			pdf.SetXY(x+1, y+1)
			pdf.MultiCell(widths[i]-2, 4, header, "", "L", false)
			x += widths[i]
		}
		pdf.SetXY(10, y+headerHeight)
		pdf.SetFont("go", "", 7)
	}
	pdf.SetFooterFunc(func() {
		pdf.SetXY(10, 201)
		pdf.SetFont("go", "", 7)
		pdf.CellFormat(277, 4, fmt.Sprintf("Uso interno · %s · Página %d", MatrixVersion, pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	addPage()
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		values := matrixRow(item)
		lines := make([][]string, len(values))
		remaining := 1
		for i, value := range values {
			lines[i] = pdf.SplitText(value, widths[i]-2)
			if len(lines[i]) > remaining {
				remaining = len(lines[i])
			}
		}
		offset := 0
		for remaining > 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			available := int((197 - pdf.GetY() - 2) / lineHeight)
			if available < 1 {
				addPage()
				continue
			}
			count := min(available, remaining)
			height := float64(count)*lineHeight + 2
			x, y := 10.0, pdf.GetY()
			for i, cell := range lines {
				end := min(offset+count, len(cell))
				text := ""
				if offset < len(cell) {
					text = strings.Join(cell[offset:end], "\n")
				}
				if offset > 0 && i < 4 {
					// Repetir sólo las líneas que caben en el fragmento de continuación.
					value := values[i]
					if i == 0 {
						value += " (cont.)"
					}
					codeLines := pdf.SplitText(value, widths[i]-2)
					text = strings.Join(codeLines[:min(count, len(codeLines))], "\n")
				}
				pdf.Rect(x, y, widths[i], height, "D")
				pdf.SetXY(x+1, y+1)
				pdf.MultiCell(widths[i]-2, lineHeight, text, "", "L", false)
				x += widths[i]
			}
			pdf.SetXY(10, y+height)
			remaining -= count
			offset += count
			if remaining > 0 {
				addPage()
			}
		}
	}
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
