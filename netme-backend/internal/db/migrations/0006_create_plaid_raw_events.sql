-- +goose Up
CREATE TABLE plaid_raw_events (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        REFERENCES users(id) ON DELETE CASCADE,
  event_type  TEXT        NOT NULL,
  payload     JSONB       NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_plaid_raw_events_user_id ON plaid_raw_events(user_id);
CREATE INDEX idx_plaid_raw_events_type    ON plaid_raw_events(event_type);
CREATE INDEX idx_plaid_raw_events_created ON plaid_raw_events(created_at DESC);

-- +goose Down
DROP TABLE plaid_raw_events;
