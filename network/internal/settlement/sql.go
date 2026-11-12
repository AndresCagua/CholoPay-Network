package settlement

import (
	"context"
	"database/sql"
	"fmt"
)

func Migrate(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS settlement_account (participant_id UUID PRIMARY KEY, balance_minor BIGINT NOT NULL CHECK (balance_minor >= 0), currency CHAR(3) NOT NULL CHECK (currency = 'USD'), created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS settlement_journal (journal_id UUID PRIMARY KEY, transfer_id UUID NOT NULL UNIQUE, idempotency_key VARCHAR(160) NOT NULL UNIQUE, request_hash CHAR(64) NOT NULL, source_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), destination_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), destination_account_id UUID NOT NULL, clearing_account_id UUID NOT NULL, amount_minor BIGINT NOT NULL CHECK (amount_minor > 0), currency CHAR(3) NOT NULL CHECK (currency = 'USD'), correlation_id UUID NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, CHECK (source_participant_id <> destination_participant_id), CHECK (destination_account_id <> clearing_account_id))`,
		`CREATE TABLE IF NOT EXISTS settlement_entry (entry_id UUID PRIMARY KEY, journal_id UUID NOT NULL REFERENCES settlement_journal(journal_id), participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), amount_minor BIGINT NOT NULL CHECK (amount_minor > 0), direction VARCHAR(6) NOT NULL CHECK (direction IN ('DEBIT','CREDIT')), currency CHAR(3) NOT NULL CHECK (currency = 'USD'), created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS settlement_entry_journal ON settlement_entry(journal_id)`,
		`CREATE TABLE IF NOT EXISTS settlement_delivery (delivery_id UUID PRIMARY KEY, transfer_id UUID NOT NULL UNIQUE REFERENCES settlement_journal(transfer_id), receiver_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), destination_account_id UUID NOT NULL, clearing_account_id UUID NOT NULL, amount_minor BIGINT NOT NULL CHECK (amount_minor > 0), event_id UUID NOT NULL UNIQUE, state VARCHAR(16) NOT NULL CHECK (state IN ('PENDING','APPLIED','FAILED')), attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0), last_event_id UUID, next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, lease_owner UUID, lease_until TIMESTAMPTZ, failure_code VARCHAR(64), failure_detail TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, applied_at TIMESTAMPTZ, CHECK (destination_account_id <> clearing_account_id))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS settlement_delivery_event ON settlement_delivery(last_event_id) WHERE last_event_id IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS reconciliation_run (run_id UUID PRIMARY KEY, started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, completed_at TIMESTAMPTZ, difference_count INTEGER NOT NULL DEFAULT 0 CHECK (difference_count >= 0))`,
		`CREATE TABLE IF NOT EXISTS reconciliation_difference (difference_id UUID PRIMARY KEY, run_id UUID NOT NULL REFERENCES reconciliation_run(run_id), transfer_id UUID, delivery_id UUID, code VARCHAR(64) NOT NULL, detail TEXT NOT NULL, observed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS reconciliation_difference_run ON reconciliation_difference(run_id)`,
	}
	for _, q := range statements {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("settlement migration: %w", err)
		}
	}
	// Keep upgrades from the preceding delivery slice non-destructive.
	for _, q := range []string{
		`ALTER TABLE settlement_account ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE settlement_account ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE settlement_journal ADD COLUMN IF NOT EXISTS destination_account_id UUID`,
		`ALTER TABLE settlement_journal ADD COLUMN IF NOT EXISTS clearing_account_id UUID`,
		`ALTER TABLE settlement_delivery ADD COLUMN IF NOT EXISTS destination_account_id UUID`,
		`ALTER TABLE settlement_delivery ADD COLUMN IF NOT EXISTS clearing_account_id UUID`,
		`ALTER TABLE settlement_delivery ADD COLUMN IF NOT EXISTS amount_minor BIGINT`,
		`ALTER TABLE settlement_delivery ADD COLUMN IF NOT EXISTS event_id UUID`,
		`ALTER TABLE settlement_delivery ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE settlement_delivery ADD COLUMN IF NOT EXISTS lease_owner UUID`,
		`ALTER TABLE settlement_delivery ADD COLUMN IF NOT EXISTS lease_until TIMESTAMPTZ`,
		`ALTER TABLE settlement_delivery ADD COLUMN IF NOT EXISTS failure_code VARCHAR(64)`,
		`ALTER TABLE settlement_delivery ADD COLUMN IF NOT EXISTS failure_detail TEXT`,
		`UPDATE settlement_journal SET destination_account_id=destination_participant_id WHERE destination_account_id IS NULL`,
		`UPDATE settlement_journal SET clearing_account_id=source_participant_id WHERE clearing_account_id IS NULL`,
		`UPDATE settlement_delivery d SET destination_account_id=j.destination_account_id,clearing_account_id=j.clearing_account_id,amount_minor=j.amount_minor FROM settlement_journal j WHERE d.transfer_id=j.transfer_id AND (d.destination_account_id IS NULL OR d.clearing_account_id IS NULL OR d.amount_minor IS NULL)`,
		`UPDATE settlement_delivery SET event_id=md5(delivery_id::text)::uuid WHERE event_id IS NULL`,
		`ALTER TABLE settlement_journal ALTER COLUMN destination_account_id SET NOT NULL`,
		`ALTER TABLE settlement_journal ALTER COLUMN clearing_account_id SET NOT NULL`,
		`ALTER TABLE settlement_delivery ALTER COLUMN destination_account_id SET NOT NULL`,
		`ALTER TABLE settlement_delivery ALTER COLUMN clearing_account_id SET NOT NULL`,
		`ALTER TABLE settlement_delivery ALTER COLUMN amount_minor SET NOT NULL`,
		`ALTER TABLE settlement_delivery ALTER COLUMN event_id SET NOT NULL`,
		`ALTER TABLE settlement_journal DROP CONSTRAINT IF EXISTS settlement_journal_delivery_accounts_check`,
		`ALTER TABLE settlement_journal ADD CONSTRAINT settlement_journal_delivery_accounts_check CHECK (destination_account_id <> clearing_account_id)`,
		`ALTER TABLE settlement_delivery DROP CONSTRAINT IF EXISTS settlement_delivery_accounts_check`,
		`ALTER TABLE settlement_delivery ADD CONSTRAINT settlement_delivery_accounts_check CHECK (destination_account_id <> clearing_account_id)`,
		`ALTER TABLE settlement_delivery DROP CONSTRAINT IF EXISTS settlement_delivery_state_check`,
		`ALTER TABLE settlement_delivery ADD CONSTRAINT settlement_delivery_state_check CHECK (state IN ('PENDING','APPLIED','FAILED'))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS settlement_delivery_event_id ON settlement_delivery(event_id)`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("settlement upgrade: %w", err)
		}
	}
	return nil
}
