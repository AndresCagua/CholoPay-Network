CREATE TABLE settlement_delivery (
  delivery_id UUID PRIMARY KEY,
  transfer_id UUID NOT NULL UNIQUE REFERENCES settlement_journal(transfer_id),
  receiver_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id),
  state VARCHAR(16) NOT NULL CHECK (state IN ('PENDING','APPLIED')),
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  last_event_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  applied_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX settlement_delivery_event ON settlement_delivery(last_event_id) WHERE last_event_id IS NOT NULL;
