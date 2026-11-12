package instruction

import (
	"context"
	"database/sql"
	"fmt"
)

func Migrate(ctx context.Context, db *sql.DB) error {
	const schema = `CREATE TABLE IF NOT EXISTS transfer_instruction (
	  transfer_id UUID PRIMARY KEY,
	  idempotency_key VARCHAR(160) NOT NULL UNIQUE,
	  request_hash CHAR(64) NOT NULL,
	  source_participant_id UUID NOT NULL,
	  destination_participant_id UUID,
	  destination_account_id UUID,
	  clearing_account_id UUID,
	  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
	  currency CHAR(3) NOT NULL CHECK (currency = 'USD'),
	  state VARCHAR(40) NOT NULL,
	  schema_version INTEGER NOT NULL CHECK (schema_version > 0),
	  correlation_id UUID NOT NULL,
	  causation_id UUID,
	  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("create transfer_instruction: %w", err)
	}
	for _, statement := range []string{
		`ALTER TABLE transfer_instruction ADD COLUMN IF NOT EXISTS destination_account_id UUID`,
		`ALTER TABLE transfer_instruction ADD COLUMN IF NOT EXISTS clearing_account_id UUID`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	_, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS transfer_instruction_state ON transfer_instruction(state, updated_at)")
	return err
}
