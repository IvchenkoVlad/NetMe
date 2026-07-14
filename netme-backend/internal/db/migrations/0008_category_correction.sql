-- +goose Up

ALTER TABLE transactions
  ADD COLUMN category_id UUID REFERENCES categories(id) ON DELETE SET NULL;

CREATE INDEX idx_transactions_category_id ON transactions(category_id);

CREATE TABLE category_rules (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  normalized_merchant TEXT        NOT NULL,
  category_id         UUID        NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, normalized_merchant)
);

CREATE INDEX idx_category_rules_user_id ON category_rules(user_id);

-- +goose Down

DROP TABLE category_rules;
ALTER TABLE transactions DROP COLUMN category_id;
