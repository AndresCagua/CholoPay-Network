package settlement

import (
	"context"
	"database/sql"
	"fmt"
)

func Migrate(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS settlement_account (participant_id UUID PRIMARY KEY, balance_minor BIGINT NOT NULL CHECK (balance_minor >= 0), currency CHAR(3) NOT NULL CHECK (currency = 'USD'), created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS settlement_journal (journal_id UUID PRIMARY KEY, transfer_id UUID NOT NULL UNIQUE, idempotency_key VARCHAR(160) NOT NULL UNIQUE, request_hash CHAR(64) NOT NULL, source_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), destination_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), amount_minor BIGINT NOT NULL CHECK (amount_minor > 0), currency CHAR(3) NOT NULL CHECK (currency = 'USD'), correlation_id UUID NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, CHECK (source_participant_id <> destination_participant_id))`,
		`CREATE TABLE IF NOT EXISTS settlement_entry (entry_id UUID PRIMARY KEY, journal_id UUID NOT NULL REFERENCES settlement_journal(journal_id), participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), amount_minor BIGINT NOT NULL CHECK (amount_minor > 0), direction VARCHAR(6) NOT NULL CHECK (direction IN ('DEBIT','CREDIT')), currency CHAR(3) NOT NULL CHECK (currency = 'USD'), created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS settlement_entry_journal ON settlement_entry(journal_id)`,
		`CREATE TABLE IF NOT EXISTS settlement_delivery (delivery_id UUID PRIMARY KEY, transfer_id UUID NOT NULL UNIQUE REFERENCES settlement_journal(transfer_id), receiver_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), state VARCHAR(16) NOT NULL CHECK (state IN ('PENDING','APPLIED')), attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0), last_event_id UUID, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, applied_at TIMESTAMPTZ)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS settlement_delivery_event ON settlement_delivery(last_event_id) WHERE last_event_id IS NOT NULL`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("settlement migration: %w", err)
		}
	}
	return nil
}
