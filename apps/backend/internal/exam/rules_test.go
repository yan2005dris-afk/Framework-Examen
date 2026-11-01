package exam

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"
)

func qaBank() BankInput {
	return BankInput{Reference: "QA NO OFICIAL", Questions: []Question{{ID: "q1", Text: "Pregunta uno", Correct: "a", Options: []Option{{ID: "a", Text: "Opción A"}, {ID: "b", Text: "Opción B"}}}, {ID: "q2", Text: "Pregunta dos", Correct: "b", Options: []Option{{ID: "a", Text: "Opción A"}, {ID: "b", Text: "Opción B"}}}, {ID: "q3", Text: "Pregunta tres", Correct: "a", Options: []Option{{ID: "a", Text: "Opción A"}, {ID: "b", Text: "Opción B"}}}}}
}
func qaRules() ConfigRules {
	return ConfigRules{Count: 3, DurationSeconds: 60, Maximum: 100, Threshold: 60, Rounding: "mas_cercano", AttemptPolicy: "un_intento", UniquePolicy: "orden_preguntas_unico_rechazar_agotamiento", Reference: "QA NO OFICIAL", OpensAt: time.Now().Add(-time.Hour), ClosesAt: time.Now().Add(time.Hour)}
}
func TestBankRejectsInvalidMatrix(t *testing.T) {
	for _, name := range []string{"no_reference", "no_questions", "duplicate_id", "missing_correct", "one_option", "duplicate_option", "duplicate_text"} {
		t.Run(name, func(t *testing.T) {
			b := qaBank()
			switch name {
			case "no_reference":
				b.Reference = ""
			case "no_questions":
				b.Questions = nil
			case "duplicate_id":
				b.Questions[1].ID = b.Questions[0].ID
			case "missing_correct":
				b.Questions[0].Correct = "fake"
			case "one_option":
				b.Questions[0].Options = b.Questions[0].Options[:1]
			case "duplicate_option":
				b.Questions[0].Options[1].ID = "a"
			case "duplicate_text":
				b.Questions[0].Options[1].Text = "Opción A"
			}
			if !errors.Is(ValidateBank(b), ErrInvalid) {
				t.Fatal("matriz inválida aceptada")
			}
		})
	}
}
func TestUniqueAssignmentsAreFiniteAndFrozen(t *testing.T) {
	bank := qaBank()
	used := map[string]bool{}
	if Capacity(3, 3).Int64() != 6 {
		t.Fatal("capacidad incorrecta")
	}
	for range 6 {
		q, h, e := Assign(bank.Questions, 3, used)
		if e != nil || used[h] || len(q) != 3 {
			t.Fatal("duplicó examen", e)
		}
		used[h] = true
		q[0].Options[0].Text = "MUTADO"
		if bank.Questions[0].Options[0].Text == "MUTADO" {
			t.Fatal("mutó banco")
		}
	}
	if _, _, e := Assign(bank.Questions, 3, used); !errors.Is(e, ErrExhausted) {
		t.Fatal("prometió unicidad imposible", e)
	}
}
func TestRulesAndGradeNeverAcceptClientScores(t *testing.T) {
	for _, name := range []string{"count", "time", "scale", "threshold", "rounding", "attempts", "unique", "window", "reference"} {
		t.Run(name, func(t *testing.T) {
			r := qaRules()
			switch name {
			case "count":
				r.Count = 4
			case "time":
				r.DurationSeconds = 0
			case "scale":
				r.Maximum = 0
			case "threshold":
				r.Threshold = 101
			case "rounding":
				r.Rounding = "auto"
			case "attempts":
				r.AttemptPolicy = "ilimitados"
			case "unique":
				r.UniquePolicy = "reutilizar"
			case "window":
				r.ClosesAt = r.OpensAt
			case "reference":
				r.Reference = ""
			}
			if r.Validate(3) == nil {
				t.Fatal("parámetros incompletos aceptados")
			}
		})
	}
	now := time.Now()
	a := &Attempt{ID: uuid.New(), SubmissionID: uuid.New(), Config: Config{ID: uuid.New(), Hash: strings.Repeat("a", 64), Rules: qaRules()}, Questions: qaBank().Questions, Answers: map[string]string{"q1": "a", "q2": "b"}, SnapshotHash: strings.Repeat("b", 64), Status: "active", ExpiresAt: now.Add(30 * time.Second)}
	v, count, e := Grade(a, now, "QA")
	if e != nil || count != 2 || *v.Score != 67 || !v.CanAdvance {
		t.Fatal("calificación incorrecta", v, e)
	}
	raw, _ := json.Marshal(PublicView(a, now))
	q, e := QuestionAt(a, 1)
	if e != nil {
		t.Fatal(e)
	}
	question, _ := json.Marshal(q)
	for _, payload := range []string{string(raw), string(question)} {
		for _, key := range []string{"correcta", "matriz", "hashBanco", "respuestas"} {
			if strings.Contains(payload, key) {
				t.Fatal("matriz filtrada", payload)
			}
		}
	}
	a.Status = "finished"
	if _, e = QuestionAt(a, 1); !errors.Is(e, ErrExpired) {
		t.Fatal(e)
	}
}
