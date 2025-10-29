CREATE TABLE IF NOT EXISTS accounts (
  id TEXT PRIMARY KEY,
  currency TEXT NOT NULL,
  balance_cents BIGINT NOT NULL CHECK (balance_cents >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transfers (
  id SERIAL PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  from_account TEXT NOT NULL REFERENCES accounts(id),
  to_account   TEXT NOT NULL REFERENCES accounts(id),
  amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
  currency TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('committed','rejected')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Helpful index for lookups by account
CREATE INDEX IF NOT EXISTS idx_transfers_from ON transfers(from_account);
CREATE INDEX IF NOT EXISTS idx_transfers_to   ON transfers(to_account);
