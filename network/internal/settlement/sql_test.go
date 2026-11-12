package settlement

import (
	"context"
	"testing"
	"github.com/google/uuid"
)

func TestMigrateBackfillsLegacyDeliveryIdentities(t *testing.T) {
	db, _ := testStore(t)
	defer db.Close()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `DROP TABLE settlement_delivery, settlement_entry, settlement_journal, settlement_account CASCADE`); err != nil {
		t.Fatal(err)
	}
	legacy := []string{
		`CREATE TABLE settlement_account (participant_id UUID PRIMARY KEY, balance_minor BIGINT NOT NULL, currency CHAR(3) NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE settlement_journal (journal_id UUID PRIMARY KEY, transfer_id UUID NOT NULL UNIQUE, idempotency_key VARCHAR(160) NOT NULL UNIQUE, request_hash CHAR(64) NOT NULL, source_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), destination_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), amount_minor BIGINT NOT NULL, currency CHAR(3) NOT NULL, correlation_id UUID NOT NULL)`,
		`CREATE TABLE settlement_entry (entry_id UUID PRIMARY KEY, journal_id UUID NOT NULL REFERENCES settlement_journal(journal_id), participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), amount_minor BIGINT NOT NULL, direction VARCHAR(6) NOT NULL, currency CHAR(3) NOT NULL)`,
		`CREATE TABLE settlement_delivery (delivery_id UUID PRIMARY KEY, transfer_id UUID NOT NULL UNIQUE REFERENCES settlement_journal(transfer_id), receiver_participant_id UUID NOT NULL REFERENCES settlement_account(participant_id), state VARCHAR(16) NOT NULL, attempt_count INTEGER NOT NULL DEFAULT 0, last_event_id UUID, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, applied_at TIMESTAMPTZ)`,
	}
	for _, statement := range legacy {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	source, destination, transferID, journalID, deliveryID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	if _, err := db.ExecContext(ctx, `INSERT INTO settlement_account(participant_id,balance_minor,currency) VALUES ($1,0,'USD'),($2,0,'USD')`, source, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO settlement_journal(journal_id,transfer_id,idempotency_key,request_hash,source_participant_id,destination_participant_id,amount_minor,currency,correlation_id) VALUES ($1,$2,'legacy-key',repeat('a',64),$3,$4,75,'USD',$5)`, journalID, transferID, source, destination, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO settlement_delivery(delivery_id,transfer_id,receiver_participant_id,state) VALUES ($1,$2,$3,'PENDING')`, deliveryID, transferID, destination); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var destinationAccount, clearingAccount, eventID uuid.UUID
	var amount int64
	if err := db.QueryRowContext(ctx, `SELECT destination_account_id,clearing_account_id,amount_minor,event_id FROM settlement_delivery WHERE delivery_id=$1`, deliveryID).Scan(&destinationAccount, &clearingAccount, &amount, &eventID); err != nil {
		t.Fatal(err)
	}
	if destinationAccount != destination || clearingAccount != source || amount != 75 || eventID == uuid.Nil {
		t.Fatalf("legacy delivery no fue backfilleada: amount=%d event=%s", amount, eventID)
	}
}
