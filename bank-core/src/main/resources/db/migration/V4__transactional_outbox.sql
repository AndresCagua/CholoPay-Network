CREATE TABLE inter_network_outbox (
  event_id UUID PRIMARY KEY,
  aggregate_type VARCHAR(64) NOT NULL CHECK (aggregate_type = 'TRANSFER'),
  aggregate_id UUID NOT NULL UNIQUE,
  event_name VARCHAR(64) NOT NULL CHECK (event_name = 'TRANSFER_REQUESTED'),
  schema_version VARCHAR(16) NOT NULL CHECK (schema_version = '1.0'),
  event_version INTEGER NOT NULL CHECK (event_version = 1),
  occurred_at TIMESTAMPTZ NOT NULL,
  correlation_id UUID NOT NULL,
  causation_id UUID NOT NULL,
  idempotency_key VARCHAR(160) NOT NULL UNIQUE,
  trace_id VARCHAR(160) NOT NULL,
  payload JSONB NOT NULL,
  serialized_envelope TEXT NOT NULL,
  payload_hash CHAR(64) NOT NULL,
  state VARCHAR(32) NOT NULL DEFAULT 'PENDING' CHECK (state IN ('PENDING', 'IN_FLIGHT', 'PUBLISHED', 'DEAD_LETTER')),
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  last_attempt_at TIMESTAMPTZ,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  lease_token UUID,
  lease_until TIMESTAMPTZ,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX inter_network_outbox_dispatch ON inter_network_outbox(state, next_attempt_at, created_at);
CREATE INDEX inter_network_outbox_lease ON inter_network_outbox(state, lease_until);
