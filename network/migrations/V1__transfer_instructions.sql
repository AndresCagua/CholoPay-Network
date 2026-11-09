CREATE TABLE transfer_instruction (
  transfer_id UUID PRIMARY KEY,
  idempotency_key VARCHAR(160) NOT NULL UNIQUE,
  request_hash CHAR(64) NOT NULL,
  source_participant_id UUID NOT NULL,
  destination_participant_id UUID,
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
  state VARCHAR(40) NOT NULL,
  schema_version INTEGER NOT NULL CHECK (schema_version > 0),
  correlation_id UUID NOT NULL,
  causation_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (idempotency_key, request_hash)
);
CREATE INDEX transfer_instruction_state ON transfer_instruction(state, updated_at);
