CREATE TABLE settlement_account (
  participant_id UUID PRIMARY KEY,
  balance_minor BIGINT NOT NULL CHECK (balance_minor >= 0),
  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE settlement_journal (
  journal_id UUID PRIMARY KEY,
  transfer_id UUID NOT NULL UNIQUE,
  idempotency_key VARCHAR(160) NOT NULL UNIQUE,
  request_hash CHAR(64) NOT NULL,
  source_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id),
  destination_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id),
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
  correlation_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (source_participant_id <> destination_participant_id)
);

CREATE TABLE settlement_entry (
  entry_id UUID PRIMARY KEY,
  journal_id UUID NOT NULL REFERENCES settlement_journal(journal_id),
  participant_id UUID NOT NULL REFERENCES settlement_account(participant_id),
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  direction VARCHAR(6) NOT NULL CHECK (direction IN ('DEBIT','CREDIT')),
  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX settlement_transfer_request ON settlement_journal(transfer_id, request_hash);
CREATE INDEX settlement_entry_journal ON settlement_entry(journal_id);
