package exam

import (
	"crypto/rand"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"math/big"
	"strings"
	"time"
)

func valid(s string, n int) bool { return strings.TrimSpace(s) != "" && len(s) <= n }
func ValidateBank(v BankInput) error {
	if v.PreviousVersion < 0 || !valid(v.Reference, 2000) || len(v.Questions) < 1 || len(v.Questions) > 500 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, q := range v.Questions {
		if !valid(q.ID, 80) || seen[q.ID] || !valid(q.Text, 4000) || len(q.Options) < 2 || len(q.Options) > 8 {
			return ErrInvalid
		}
		seen[q.ID] = true
		ids := map[string]bool{}
		texts := map[string]bool{}
		for _, o := range q.Options {
			if !valid(o.ID, 80) || ids[o.ID] || !valid(o.Text, 2000) || texts[strings.TrimSpace(o.Text)] {
				return ErrInvalid
			}
			ids[o.ID] = true
			texts[strings.TrimSpace(o.Text)] = true
		}
		if !ids[q.Correct] {
			return ErrInvalid
		}
	}
	raw, _ := json.Marshal(v)
	if len(raw) > 2<<20 {
		return ErrInvalid
	}
	return nil
}
func (r ConfigRules) Validate(n int) error {
	if r.Count < 1 || r.Count > 100 || r.Count > n || r.DurationSeconds < 1 || r.DurationSeconds > 14400 || r.Maximum < 1 || r.Maximum > 1e9 || r.Threshold < 0 || r.Threshold > r.Maximum || (r.Rounding != "inferior" && r.Rounding != "mas_cercano") || r.AttemptPolicy != "un_intento" || r.UniquePolicy != "orden_preguntas_unico_rechazar_agotamiento" || !valid(r.Reference, 2000) || r.OpensAt.IsZero() || !r.ClosesAt.After(r.OpensAt) || r.ClosesAt.Sub(r.OpensAt) < time.Duration(r.DurationSeconds)*time.Second {
		return ErrInvalid
	}
	return nil
}

/** Capacity cuenta permutaciones sin prometer combinaciones ilimitadas. */
func Capacity(n, k int) *big.Int {
	v := big.NewInt(1)
	if k < 1 || n < k {
		return big.NewInt(0)
	}
	for i := 0; i < k; i++ {
		v.Mul(v, big.NewInt(int64(n-i)))
	}
	return v
}

/** Assign selecciona al azar y, ante colisiones repetidas, busca exhaustivamente un orden libre en el espacio finito. */
func Assign(bank []Question, k int, used map[string]bool) ([]Question, string, error) {
	if k < 1 || k > len(bank) || Capacity(len(bank), k).Cmp(big.NewInt(int64(len(used)))) <= 0 {
		return nil, "", ErrExhausted
	}
	permutation := func(indices []int) ([]Question, string, error) {
		out := make([]Question, k)
		ids := make([]string, k)
		for i, n := range indices[:k] {
			out[i] = bank[n]
			out[i].Options = append([]Option(nil), bank[n].Options...)
			ids[i] = bank[n].ID
		}
		h, err := evaluation.Digest(struct{ IDs []string }{ids})
		return out, h, err
	}
	var out []Question
	var hash string
	for tries := 0; tries < 64; tries++ {
		indices := make([]int, len(bank))
		for i := range indices {
			indices[i] = i
		}
		for i := len(indices) - 1; i > 0; i-- {
			j, e := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
			if e != nil {
				return nil, "", e
			}
			indices[i], indices[int(j.Int64())] = indices[int(j.Int64())], indices[i]
		}
		v, h, e := permutation(indices)
		if e != nil {
			return nil, "", e
		}
		if !used[h] {
			out, hash = v, h
			break
		}
	}
	if out == nil {
		indices := make([]int, k)
		taken := make([]bool, len(bank))
		var seek func(int) bool
		seek = func(pos int) bool {
			if pos == k {
				v, h, e := permutation(indices)
				if e == nil && !used[h] {
					out, hash = v, h
					return true
				}
				return false
			}
			for i := range bank {
				if !taken[i] {
					indices[pos] = i
					taken[i] = true
					if seek(pos + 1) {
						return true
					}
					taken[i] = false
				}
			}
			return false
		}
		if !seek(0) {
			return nil, "", ErrExhausted
		}
	}
	for i := range out {
		for j := len(out[i].Options) - 1; j > 0; j-- {
			v, e := rand.Int(rand.Reader, big.NewInt(int64(j+1)))
			if e != nil {
				return nil, "", e
			}
			n := int(v.Int64())
			out[i].Options[j], out[i].Options[n] = out[i].Options[n], out[i].Options[j]
		}
	}
	return out, hash, nil
}
func Grade(a *Attempt, now time.Time, reason string) (*evaluation.Result, int, error) {
	if a == nil || len(a.Questions) != a.Config.Rules.Count || a.Config.Rules.Validate(len(a.Questions)) != nil {
		return nil, 0, ErrInvalid
	}
	count := 0
	for _, q := range a.Questions {
		if a.Answers[q.ID] == q.Correct {
			count++
		}
	}
	numerator := big.NewInt(a.Config.Rules.Maximum)
	numerator.Mul(numerator, big.NewInt(int64(count)))
	denom := big.NewInt(int64(len(a.Questions)))
	if a.Config.Rules.Rounding == "mas_cercano" {
		numerator.Add(numerator, new(big.Int).Div(new(big.Int).Set(denom), big.NewInt(2)))
	}
	score := numerator.Div(numerator, denom).Int64()
	maximum := a.Config.Rules.Maximum
	id, err := uuid.NewV7()
	if err != nil {
		return nil, 0, err
	}
	return &evaluation.Result{ID: id, SubmissionID: a.SubmissionID, Phase: "oposicion_teorica", Version: 1, SourceID: a.ID, PolicyID: a.Config.ID, PolicyHash: a.Config.Hash, SnapshotHash: a.SnapshotHash, Score: &score, Maximum: &maximum, CanAdvance: score >= a.Config.Rules.Threshold, Motive: reason, PublishedAt: now, EngineVersion: "examen-v1"}, count, nil
}
func PublicView(a *Attempt, now time.Time) View {
	remaining := int64(0)
	if a.Status == "active" && a.ExpiresAt.After(now) {
		remaining = int64((a.ExpiresAt.Sub(now) + time.Second - 1) / time.Second)
	}
	return View{ID: a.ID, SubmissionID: a.SubmissionID, Version: a.Version, Status: a.Status, StartedAt: a.StartedAt, ExpiresAt: a.ExpiresAt, ServerAt: now, RemainingSeconds: remaining, Count: len(a.Questions), Answered: len(a.Answers), Result: a.Result}
}
func QuestionAt(a *Attempt, index int) (PublicQuestion, error) {
	if a.Status != "active" {
		return PublicQuestion{}, ErrExpired
	}
	if index < 1 || index > len(a.Questions) {
		return PublicQuestion{}, ErrInvalid
	}
	q := a.Questions[index-1]
	return PublicQuestion{ID: q.ID, Text: q.Text, Options: q.Options, Selected: a.Answers[q.ID], Index: index, Version: a.Version}, nil
}
