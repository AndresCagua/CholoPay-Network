CREATE TABLE ledger_account (
  id UUID PRIMARY KEY,
  owner_ref VARCHAR(120) NOT NULL,
  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
  restricted BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE journal (
  id UUID PRIMARY KEY,
  transaction_id UUID NOT NULL UNIQUE,
  idempotency_key VARCHAR(160) NOT NULL,
  correlation_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE journal_entry (
  id UUID PRIMARY KEY,
  journal_id UUID NOT NULL REFERENCES journal(id),
  account_id UUID NOT NULL REFERENCES ledger_account(id),
  amount_minor BIGINT NOT NULL CHECK (amount_minor <> 0),
  direction VARCHAR(6) NOT NULL CHECK (direction IN ('DEBIT','CREDIT')),
  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE balance_reservation (
  id UUID PRIMARY KEY,
  account_id UUID NOT NULL REFERENCES ledger_account(id),
  transfer_id UUID NOT NULL UNIQUE,
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  status VARCHAR(12) NOT NULL CHECK (status IN ('ACTIVE','RELEASED','CAPTURED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX journal_idempotency_key ON journal(idempotency_key);
CREATE INDEX journal_entry_account ON journal_entry(account_id, created_at);
CREATE INDEX reservation_account_status ON balance_reservation(account_id, status);
