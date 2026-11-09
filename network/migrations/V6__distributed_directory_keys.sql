CREATE TABLE directory_key (
  key_id UUID PRIMARY KEY,
  network_id UUID NOT NULL REFERENCES payment_network(network_id),
  participant_id UUID NOT NULL REFERENCES network_participant(participant_id),
  key_type VARCHAR(32) NOT NULL,
  key_value VARCHAR(256) NOT NULL,
  account_ref VARCHAR(128) NOT NULL,
  status VARCHAR(16) NOT NULL CHECK (status IN ('ACTIVE','SUSPENDED')),
  version BIGINT NOT NULL CHECK (version > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (network_id, key_type, key_value)
);
CREATE INDEX directory_key_lookup ON directory_key(network_id, key_type, key_value, status);
