-- +goose Up
CREATE TABLE plaid_items (
  id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  plaid_item_id   TEXT    NOT NULL UNIQUE,
  access_token    TEXT    NOT NULL,
  institution_id  TEXT,
  institution_name TEXT,
  cursor          TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_plaid_items_user_id ON plaid_items(user_id);

-- +goose Down
DROP TABLE plaid_items;
