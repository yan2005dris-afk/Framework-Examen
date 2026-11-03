-- +goose Up
-- Nunca persistir OTP, destinatario ni formulario de un correo en texto claro.
CREATE TABLE mail_outbox (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    dedup_key VARCHAR(64) NOT NULL UNIQUE,
    kind VARCHAR(32) NOT NULL,
    resource_id UUID,
    ciphertext BYTEA NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending'
        CHECK(status IN ('pending','processing','smtp_accepted','failed','expired','superseded')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 5),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    lease_id UUID,
    last_error_code VARCHAR(32) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX mail_outbox_pending_idx ON mail_outbox(next_attempt_at,created_at)
    WHERE status IN ('pending','processing');
CREATE INDEX mail_outbox_resource_idx ON mail_outbox(resource_id,created_at DESC,id DESC);

-- +goose Down
DROP TABLE mail_outbox;
