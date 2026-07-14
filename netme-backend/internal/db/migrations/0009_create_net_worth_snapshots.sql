-- +goose Up
CREATE TABLE net_worth_snapshots (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  assets      NUMERIC(14,2) NOT NULL DEFAULT 0,
  liabilities NUMERIC(14,2) NOT NULL DEFAULT 0,
  net_worth   NUMERIC(14,2) NOT NULL DEFAULT 0,
  recorded_at DATE        NOT NULL DEFAULT CURRENT_DATE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, recorded_at)
);

CREATE INDEX idx_nw_snapshots_user_date ON net_worth_snapshots(user_id, recorded_at DESC);

-- +goose Down
DROP TABLE net_worth_snapshots;
