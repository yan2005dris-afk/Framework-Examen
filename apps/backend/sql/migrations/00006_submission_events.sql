-- +goose Up
CREATE TABLE submission_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    applicant_id UUID NOT NULL REFERENCES applicants(id),
    event_type VARCHAR(16) NOT NULL CHECK (event_type IN ('guardada', 'enviada', 'consultada')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX submission_events_submission_idx
    ON submission_events (submission_id, created_at, id);

-- +goose Down
DROP TABLE submission_events;
