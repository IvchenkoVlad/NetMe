-- +goose Up
CREATE TABLE accounts (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  plaid_item_id     UUID        NOT NULL REFERENCES plaid_items(id) ON DELETE CASCADE,
  plaid_account_id  TEXT        NOT NULL UNIQUE,
  name              TEXT        NOT NULL,
  official_name     TEXT,
  type              TEXT        NOT NULL,
  subtype           TEXT,
  mask              TEXT,
  current_balance   NUMERIC(12,2),
  available_balance NUMERIC(12,2),
  currency_code     TEXT        NOT NULL DEFAULT 'USD',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_accounts_plaid_item_id ON accounts(plaid_item_id);

-- +goose Down
DROP TABLE accounts;
