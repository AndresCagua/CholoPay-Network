ALTER TABLE inter_network_outbox
  ADD COLUMN IF NOT EXISTS serialized_envelope TEXT,
  ADD COLUMN IF NOT EXISTS attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  ADD COLUMN IF NOT EXISTS last_attempt_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN IF NOT EXISTS lease_token UUID,
  ADD COLUMN IF NOT EXISTS lease_until TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_error TEXT;

UPDATE inter_network_outbox
SET serialized_envelope = json_build_object(
  'event_id', event_id,
  'aggregate_type', aggregate_type,
  'aggregate_id', aggregate_id,
  'event_name', event_name,
  'schema_version', schema_version,
  'version', event_version,
  'occurred_at', occurred_at,
  'correlation_id', correlation_id,
  'causation_id', causation_id,
  'idempotency_key', idempotency_key,
  'trace_id', trace_id,
  'payload', payload
)::text
WHERE serialized_envelope IS NULL;

ALTER TABLE inter_network_outbox
  ALTER COLUMN serialized_envelope SET NOT NULL,
  DROP CONSTRAINT IF EXISTS inter_network_outbox_state_check;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'inter_network_outbox_state_check'
      AND conrelid = 'inter_network_outbox'::regclass
  ) THEN
    ALTER TABLE inter_network_outbox
      ADD CONSTRAINT inter_network_outbox_state_check
        CHECK (state IN ('PENDING', 'IN_FLIGHT', 'PUBLISHED', 'DEAD_LETTER'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS inter_network_outbox_dispatch
  ON inter_network_outbox(state, next_attempt_at, created_at);
CREATE INDEX IF NOT EXISTS inter_network_outbox_lease
  ON inter_network_outbox(state, lease_until);
