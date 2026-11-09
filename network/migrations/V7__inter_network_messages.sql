CREATE TABLE inter_network_message (
  message_id UUID PRIMARY KEY,
  idempotency_key VARCHAR(128) NOT NULL UNIQUE,
  sender_network_id UUID NOT NULL REFERENCES payment_network(network_id),
  destination_network_id UUID NOT NULL REFERENCES payment_network(network_id),
  schema_version VARCHAR(32) NOT NULL,
  message_type VARCHAR(64) NOT NULL,
  correlation_id UUID NOT NULL,
  causation_id UUID NOT NULL,
  payload_hash CHAR(64) NOT NULL,
  payload JSONB NOT NULL,
  status VARCHAR(16) NOT NULL CHECK (status IN ('ACCEPTED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (sender_network_id <> destination_network_id)
);
CREATE INDEX inter_network_message_route ON inter_network_message(sender_network_id, destination_network_id, created_at);
