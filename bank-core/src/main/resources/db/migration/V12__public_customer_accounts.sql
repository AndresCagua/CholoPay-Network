-- Base sintética para el acceso público, aislada por participante y cliente.
CREATE TABLE public_customer (
  id UUID PRIMARY KEY,
  participant_id VARCHAR(120) NOT NULL,
  synthetic_reference VARCHAR(160) NOT NULL,
  api_token_hash CHAR(64) NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (participant_id, synthetic_reference)
);

ALTER TABLE ledger_account ADD COLUMN participant_id VARCHAR(120);
ALTER TABLE ledger_account ADD COLUMN customer_id UUID REFERENCES public_customer(id);
CREATE INDEX ledger_account_public_customer ON ledger_account(participant_id, customer_id);
