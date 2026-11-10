CREATE TABLE directory_key (
  key_id UUID PRIMARY KEY,
  network_id UUID NOT NULL REFERENCES payment_network(network_id),
  participant_id UUID NOT NULL REFERENCES network_participant(participant_id),
  key_type VARCHAR(32) NOT NULL,
  key_value VARCHAR(256) NOT NULL,
  account_ref VARCHAR(128) NOT NULL,
  account_kind VARCHAR(16) NOT NULL DEFAULT 'PERSONAL' CHECK (account_kind IN ('PERSONAL','BUSINESS')),
  status VARCHAR(16) NOT NULL CHECK (status IN ('ACTIVE','SUSPENDED')),
  version BIGINT NOT NULL CHECK (version > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (network_id, key_type, key_value)
);
CREATE UNIQUE INDEX directory_key_global_active_identity ON directory_key(key_type, key_value) WHERE status='ACTIVE' AND key_type IN ('EMAIL','PHONE','DOCUMENT');
CREATE INDEX directory_key_lookup ON directory_key(network_id, key_type, key_value, status);
CREATE TABLE directory_key_operations (
  idempotency_key VARCHAR(128) PRIMARY KEY,
  request_fingerprint CHAR(64) NOT NULL,
  key_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
