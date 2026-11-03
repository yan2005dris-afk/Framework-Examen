package report

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReportDeterministicAndCancelled(t *testing.T) {
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	score, maximum := int64(70), int64(100)
	a := &exam.Attempt{ID: uuid.MustParse("00000000-0000-7000-8000-000000000001"), SubmissionID: uuid.MustParse("00000000-0000-7000-8000-000000000002"), Status: "finished", StartedAt: at.Add(-time.Hour), ExpiresAt: at, FinishedAt: &at, SnapshotHash: "HASH-QA-NO-OFICIAL", BankHash: "BANCO-QA", Config: exam.Config{ID: uuid.MustParse("00000000-0000-7000-8000-000000000003"), Hash: "REGLAS-QA", Rules: exam.ConfigRules{Threshold: 60, Rounding: "inferior"}}, Questions: []exam.Question{{ID: "q1"}}, Answers: map[string]string{"q1": "a"}, CorrectCount: 1, Reason: "QA SINTÉTICO — NO OFICIAL", Result: &evaluation.Result{ID: uuid.MustParse("00000000-0000-7000-8000-000000000004"), Score: &score, Maximum: &maximum, CanAdvance: true}}
	a.Questions = make([]exam.Question, 10)
	a.Answers = map[string]string{}
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("q%d", i)
		a.Questions[i] = exam.Question{ID: key}
		if i < 7 {
			a.Answers[key] = "a"
		}
	}
	a.CorrectCount = 7
	r := NewRenderer()
	data, e := r.Render(context.Background(), a)
	if e != nil {
		t.Fatal(e)
	}
	again, e := r.Render(context.Background(), a)
	if e != nil || !bytes.Equal(data, again) || !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("PDF no reproducible", e)
	}
	path := filepath.Join(t.TempDir(), "reporte.pdf")
	if e = os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	if qa := os.Getenv("EXAM_QA_PDF_PATH"); qa != "" {
		if e = os.WriteFile(qa, data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = r.Render(ctx, a); e == nil {
		t.Fatal("ignoró cancelación")
	}
}
