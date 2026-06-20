-- +goose Up
CREATE TABLE transactions (
  id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id               UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  account_id            UUID        NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  plaid_transaction_id  TEXT        NOT NULL UNIQUE,
  amount                NUMERIC(12,2) NOT NULL,
  currency_code         TEXT        NOT NULL DEFAULT 'USD',
  name                  TEXT        NOT NULL,
  merchant_name         TEXT,
  date                  DATE        NOT NULL,
  authorized_date       DATE,
  category              TEXT,
  category_detailed     TEXT,
  payment_channel       TEXT,
  pending               BOOLEAN     NOT NULL DEFAULT false,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_account_id ON transactions(account_id);
CREATE INDEX idx_transactions_date ON transactions(date DESC);

-- +goose Down
DROP TABLE transactions;
