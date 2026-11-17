INSERT INTO ledger_account (id, owner_ref, currency, posted_balance_minor)
VALUES
  ('00000000-0000-4000-8000-000000000001', 'synthetic-electricity-source', 'USD', 1000000),
  ('00000000-0000-4000-8000-000000000002', 'synthetic-utilities-clearing', 'USD', 0)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE payment_intent (
  id UUID PRIMARY KEY,
  obligation_id VARCHAR(160) NOT NULL,
  idempotency_key VARCHAR(160) NOT NULL UNIQUE,
  request_hash CHAR(64) NOT NULL,
  participant_id VARCHAR(120) NOT NULL,
  source_account_id UUID NOT NULL REFERENCES ledger_account(id),
  clearing_account_id UUID NOT NULL REFERENCES ledger_account(id),
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
  biller VARCHAR(64) NOT NULL,
  service VARCHAR(64) NOT NULL,
  invoice_id VARCHAR(160),
  journal_id UUID NOT NULL UNIQUE REFERENCES journal(id),
  status VARCHAR(32) NOT NULL CHECK (status IN ('PENDING_APPLICATION', 'UNKNOWN_PENDING_RECONCILIATION')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX payment_intent_obligation ON payment_intent(obligation_id, created_at);
CREATE INDEX payment_intent_status ON payment_intent(status, created_at);

CREATE TABLE bill_payment_outbox (
  event_id UUID PRIMARY KEY,
  payment_intent_id UUID NOT NULL UNIQUE REFERENCES payment_intent(id),
  event_name VARCHAR(64) NOT NULL CHECK (event_name = 'BILL_PAYMENT_REQUESTED'),
  schema_version VARCHAR(16) NOT NULL CHECK (schema_version = '1.0'),
  idempotency_key VARCHAR(160) NOT NULL UNIQUE,
  payload JSONB NOT NULL,
  payload_hash CHAR(64) NOT NULL,
  state VARCHAR(16) NOT NULL DEFAULT 'PENDING' CHECK (state = 'PENDING'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX bill_payment_outbox_pending ON bill_payment_outbox(state, created_at);
