CREATE TABLE payment_network (
  network_id UUID PRIMARY KEY,
  code VARCHAR(64) NOT NULL UNIQUE,
  kind VARCHAR(16) NOT NULL CHECK (kind IN ('BANKING','FINTECH')),
  status VARCHAR(16) NOT NULL CHECK (status IN ('ACTIVE','SUSPENDED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE network_participant (
  participant_id UUID PRIMARY KEY,
  network_id UUID NOT NULL REFERENCES payment_network(network_id),
  code VARCHAR(64) NOT NULL UNIQUE,
  status VARCHAR(16) NOT NULL CHECK (status IN ('ACTIVE','SUSPENDED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX network_participant_network ON network_participant(network_id, status);
