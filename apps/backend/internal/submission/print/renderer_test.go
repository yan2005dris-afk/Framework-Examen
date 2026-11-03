package print

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"
)

func printableFixture(t *testing.T) *submission.Submission {
	t.Helper()
	raw, err := os.ReadFile("../testdata/form_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var shared struct {
		Base map[string]interface{} `json:"base"`
	}
	if err = json.Unmarshal(raw, &shared); err != nil {
		t.Fatal(err)
	}
	content := shared.Base
	consent := map[string]interface{}{"version": "LEGADO-QA", "puntos": []string{"LITERAL-UNO: Declaración de veracidad guardada.", "LITERAL-DOS: Reglas del concurso guardadas.", "LITERAL-TRES: Tratamiento de datos guardado.", "LITERAL-CUATRO: Publicación del nombre guardada.", "LITERAL-CINCO: Entrega con firma manuscrita guardada."}, "aceptadoEn": "2026-10-31T15:00:00Z"}
	content["consentimiento"] = consent
	content["datoLegado"] = "MARCADOR-CONSERVADO <script>no ejecutable</script>"
	raw, err = json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := submission.CanonicalContent(raw)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)
	return &submission.Submission{ID: uuid.MustParse("019a3be0-0100-7000-8000-000000000001"), ApplicantID: uuid.MustParse("019a3be0-0100-7000-8000-000000000002"), Code: "EF12-QA-NO-OFICIAL", Status: submission.StatusSubmitted, SubmittedAt: &now, Content: canonical, Hash: submission.HashContent(canonical)}
}
func TestRendererUsesDeterministicSealedSnapshot(t *testing.T) {
	item := printableFixture(t)
	renderer := NewRenderer()
	data, err := renderer.Render(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	again, err := renderer.Render(context.Background(), item)
	if err != nil || !bytes.Equal(data, again) {
		t.Fatal("PDF no reproducible", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("no es PDF")
	}
	if path := os.Getenv("EF12_QA_PDF"); path != "" {
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	before := item.Hash
	item.Content = []byte(strings.Replace(string(item.Content), "LEGADO-QA", "CAMBIADO", 1))
	if _, err = renderer.Render(context.Background(), item); !errors.Is(err, submission.ErrInvalidSeal) || item.Hash != before {
		t.Fatal("reconstruyó sello alterado")
	}
}
func TestRendererRejectsDraftAndCancellation(t *testing.T) {
	item := printableFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewRenderer().Render(ctx, item); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	item.Status = submission.StatusDraft
	if _, err := NewRenderer().Render(context.Background(), item); !errors.Is(err, submission.ErrInvalidSeal) {
		t.Fatal(err)
	}
}

func TestLegacySignatureDoesNotBecomeMandatoryOrBreakPrinting(t *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 12, 8))); err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{encoded.Bytes(), encoded.Bytes()[:35]} {
		item := printableFixture(t)
		var content map[string]interface{}
		if err := json.Unmarshal(item.Content, &content); err != nil {
			t.Fatal(err)
		}
		consent := content["consentimiento"].(map[string]interface{})
		consent["firmaManuscritaDataUrl"] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
		raw, _ := json.Marshal(content)
		item.Content, _ = submission.CanonicalContent(raw)
		item.Hash = submission.HashContent(item.Content)
		if _, err := NewRenderer().Render(context.Background(), item); err != nil {
			t.Fatal("firma legada rompió impresión", err)
		}
	}
	if got := clean("Español 漢"); got != "Español [U+6F22]" {
		t.Fatal("glifo perdido", got)
	}
}
