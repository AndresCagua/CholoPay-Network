CREATE TABLE reconciliation_run (
  run_id UUID PRIMARY KEY,
  started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMPTZ,
  difference_count INTEGER NOT NULL DEFAULT 0 CHECK (difference_count >= 0)
);

CREATE TABLE reconciliation_difference (
  difference_id UUID PRIMARY KEY,
  run_id UUID NOT NULL REFERENCES reconciliation_run(run_id),
  transfer_id UUID,
  delivery_id UUID,
  code VARCHAR(64) NOT NULL,
  detail TEXT NOT NULL,
  observed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX reconciliation_difference_run ON reconciliation_difference(run_id);
CREATE INDEX reconciliation_difference_transfer ON reconciliation_difference(transfer_id);
