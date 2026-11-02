package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/testutil"
)

// newDraft prepara un borrador del mismo postulante en la convocatoria indicada.
func newDraft(t *testing.T, applicantID uuid.UUID, competitionID uuid.UUID) *submission.Submission {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	item, err := submission.NewSubmission(id, applicantID, "EF12-"+strings.ToUpper(id.String()[24:]), json.RawMessage(`{}`), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	item.CompetitionID = &competitionID
	return item
}

func TestPostgresOneSubmissionPerCompetition(t *testing.T) {
	repo, item, pool := integrationEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := repo.Create(ctx, newDraft(t, item.ApplicantID, *item.CompetitionID)); !errors.Is(err, submission.ErrSubmissionExists) {
		t.Fatalf("segunda postulación en la misma convocatoria: %v", err)
	}

	other := testutil.OpenCompetition(t, pool)
	if err := repo.Create(ctx, newDraft(t, item.ApplicantID, other)); err != nil {
		t.Fatalf("otra convocatoria debe admitirse: %v", err)
	}

	found, err := repo.FindByApplicantAndCompetition(ctx, item.ApplicantID, *item.CompetitionID)
	if err != nil || found == nil || found.ID != item.ID {
		t.Fatalf("búsqueda por convocatoria: %v %v", found, err)
	}
	missing, err := repo.FindByApplicantAndCompetition(ctx, item.ApplicantID, uuid.New())
	if err != nil || missing != nil {
		t.Fatalf("convocatoria sin postulación: %v %v", missing, err)
	}

	// El legado sin convocatoria queda fuera del índice parcial.
	var definition string
	if err := pool.QueryRow(ctx, `SELECT indexdef FROM pg_indexes WHERE indexname='submissions_one_per_competition'`).Scan(&definition); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(definition, "UNIQUE") || !strings.Contains(definition, "WHERE (competition_id IS NOT NULL)") {
		t.Fatalf("definición del índice: %s", definition)
	}
}

func TestPostgresDiscardDraftAllowsNewSubmission(t *testing.T) {
	repo, item, _ := integrationEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := repo.Delete(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, newDraft(t, item.ApplicantID, *item.CompetitionID)); err != nil {
		t.Fatalf("tras descartar el borrador debe poder crear otro: %v", err)
	}
}

func TestPostgresConcurrentCreatesKeepOneSubmission(t *testing.T) {
	repo, item, pool := integrationEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	period := testutil.OpenCompetition(t, pool)

	results := make(chan error, 4)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			results <- repo.Create(ctx, newDraft(t, item.ApplicantID, period))
		})
	}
	wg.Wait()
	close(results)
	created, duplicated := 0, 0
	for err := range results {
		switch {
		case err == nil:
			created++
		case errors.Is(err, submission.ErrSubmissionExists):
			duplicated++
		default:
			t.Fatal(err)
		}
	}
	if created != 1 || duplicated != 3 {
		t.Fatalf("creadas=%d duplicadas=%d", created, duplicated)
	}
}

func TestPostgresClosedSubmissionStillBlocksNewDraft(t *testing.T) {
	repo, item, pool := integrationEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, `UPDATE submissions SET status='closed', eligibility='{"elegible":false}'::jsonb WHERE id=$1`, item.ID); err != nil {
		t.Fatal(err)
	}
	err := repo.Create(ctx, newDraft(t, item.ApplicantID, *item.CompetitionID))
	if !errors.Is(err, submission.ErrEligibilityClosed) && !errors.Is(err, submission.ErrSubmissionExists) {
		t.Fatalf("un cierre por elegibilidad debe impedir otro borrador: %v", err)
	}
	if err := repo.Delete(ctx, item.ID); err == nil {
		t.Fatal("una postulación cerrada no debe poder eliminarse")
	}
}
