CREATE TABLE receiver_credit_inbox (
  delivery_id UUID PRIMARY KEY,
  event_id UUID NOT NULL UNIQUE,
  transfer_id UUID NOT NULL UNIQUE,
  destination_account_id UUID NOT NULL REFERENCES ledger_account(id),
  clearing_account_id UUID NOT NULL REFERENCES ledger_account(id),
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
  payload_hash CHAR(64) NOT NULL,
  journal_id UUID NOT NULL UNIQUE REFERENCES journal(id),
  applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (destination_account_id <> clearing_account_id)
);
CREATE INDEX receiver_inbox_destination ON receiver_credit_inbox(destination_account_id, applied_at);
