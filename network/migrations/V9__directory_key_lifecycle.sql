-- Estados terminales conservan el tombstone; solo los estados activos participan en unicidad.
ALTER TABLE directory_key DROP CONSTRAINT IF EXISTS directory_key_network_id_key_type_key_value_key;
ALTER TABLE directory_key DROP CONSTRAINT IF EXISTS directory_key_status_check;
ALTER TABLE directory_key ADD CONSTRAINT directory_key_status_check CHECK (status IN ('ACTIVE','SUSPENDED','BLOCKED','DELETED'));
CREATE UNIQUE INDEX IF NOT EXISTS directory_key_network_active_identity
  ON directory_key(network_id, key_type, key_value) WHERE status = 'ACTIVE';
CREATE UNIQUE INDEX IF NOT EXISTS directory_key_global_active_identity_v9
  ON directory_key(key_type, key_value) WHERE status = 'ACTIVE' AND key_type IN ('EMAIL','PHONE','DOCUMENT');

ALTER TABLE directory_key_operations
  ADD COLUMN IF NOT EXISTS operation_type VARCHAR(32) NOT NULL DEFAULT 'PORT',
  ADD COLUMN IF NOT EXISTS participant_id UUID,
  ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'COMPLETED',
  ADD COLUMN IF NOT EXISTS schema_version VARCHAR(32) NOT NULL DEFAULT '1.0',
  ADD COLUMN IF NOT EXISTS correlation_id UUID,
  ADD COLUMN IF NOT EXISTS causation_id UUID,
  ADD COLUMN IF NOT EXISTS trace_id VARCHAR(128),
  ADD COLUMN IF NOT EXISTS requested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE TABLE IF NOT EXISTS directory_key_history (
  history_id UUID PRIMARY KEY,
  key_id UUID NOT NULL REFERENCES directory_key(key_id),
  participant_id UUID NOT NULL REFERENCES network_participant(participant_id),
  operation_type VARCHAR(32) NOT NULL,
  from_status VARCHAR(16),
  to_status VARCHAR(16) NOT NULL,
  reason VARCHAR(256) NOT NULL DEFAULT '',
  idempotency_key VARCHAR(128),
  correlation_id UUID,
  causation_id UUID,
  schema_version VARCHAR(32) NOT NULL DEFAULT '1.0',
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS directory_key_history_key_order
  ON directory_key_history(key_id, occurred_at, history_id);
DROP INDEX IF EXISTS directory_key_history_idempotency;
CREATE UNIQUE INDEX IF NOT EXISTS directory_key_history_idempotency
  ON directory_key_history(idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';
