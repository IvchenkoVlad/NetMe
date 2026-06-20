-- +goose Up
CREATE TABLE categories (
  id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name                     TEXT        NOT NULL,
  icon                     TEXT        NOT NULL DEFAULT '📦',
  color                    TEXT        NOT NULL DEFAULT '#94a3b8',
  is_income                BOOLEAN     NOT NULL DEFAULT false,
  sort_order               INT         NOT NULL DEFAULT 0,
  plaid_primary_categories TEXT[]      NOT NULL DEFAULT '{}',
  created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_categories_user_id ON categories(user_id);

CREATE TABLE budgets (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  category_id UUID        NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  month       TEXT        NOT NULL, -- 'YYYY-MM'
  amount      NUMERIC(12,2) NOT NULL DEFAULT 0,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, category_id, month)
);

CREATE INDEX idx_budgets_user_month ON budgets(user_id, month);

-- +goose Down
DROP TABLE budgets;
DROP TABLE categories;
