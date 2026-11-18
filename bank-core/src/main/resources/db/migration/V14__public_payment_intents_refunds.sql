-- Registra reembolsos sintéticos completos sin modificar retrospectivamente el ledger.
CREATE TABLE payment_intent_refund (
  id UUID PRIMARY KEY,
  payment_intent_id UUID NOT NULL REFERENCES payment_intent(id),
  idempotency_key VARCHAR(160) NOT NULL UNIQUE,
  request_hash CHAR(64) NOT NULL,
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
  status VARCHAR(24) NOT NULL CHECK (status = 'SUCCEEDED'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (payment_intent_id)
);
CREATE INDEX payment_intent_refund_payment ON payment_intent_refund(payment_intent_id);
