package submission

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
)

type queueFailure struct{}

func (queueFailure) Enqueue(context.Context, *Submission) error { return errors.New("fallo de cola") }
func (queueFailure) State(context.Context, uuid.UUID) (*DeliveryStatus, error) {
	return nil, errors.New("fallo de estado")
}

func TestQueueFailureRollsBackSealAndDoesNotReportSuccess(t *testing.T) {
	service, repo, notifier := serviceFixture(t, true)
	service.WithConfirmationQueue(queueFailure{})
	if _, err := service.Submit(context.Background(), repo.item.ID, nil); !errors.Is(err, ErrNotificationUnavailable) {
		t.Fatal("sello confirmado sin cola", err)
	}
	if repo.item.Status == StatusSubmitted || repo.item.Hash != "" || notifier.calls != 0 {
		t.Fatal("se publicó un envío sin confirmación durable")
	}
}
