package instruction

import (
	"context"
	"database/sql"
	"github.com/cholopay/network/internal/transfer"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"os"
	"testing"
)

func TestPostgresInstructionStore(t *testing.T) {
	url := os.Getenv("NETWORK_DB_URL")
	if url == "" {
		t.Skip("NETWORK_DB_URL es necesario para la prueba de integración de Docker")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE transfer_instruction"); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	source, destination := uuid.New(), uuid.New()
	destinationAccount, clearingAccount := uuid.New(), uuid.New()
	created, err := store.Create(ctx, Instruction{IdempotencyKey: "network-test-key", SourceParticipantID: source, DestinationParticipantID: &destination, DestinationAccountID: &destinationAccount, ClearingAccountID: &clearingAccount, AmountMinor: 500, Currency: "USD", CorrelationID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := store.Create(ctx, Instruction{IdempotencyKey: "network-test-key", SourceParticipantID: source, DestinationParticipantID: &destination, DestinationAccountID: &destinationAccount, ClearingAccountID: &clearingAccount, AmountMinor: 500, Currency: "USD", CorrelationID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if created.TransferID != replayed.TransferID || replayed.State != transfer.Created {
		t.Fatalf("repetición inesperada: %+v", replayed)
	}
	if err := store.Advance(ctx, created.TransferID, transfer.Validating, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if err := store.Advance(ctx, created.TransferID, transfer.Completed, uuid.New()); err == nil {
		t.Fatal("se esperaba una transición no válida")
	}
	if _, err := store.Create(ctx, Instruction{IdempotencyKey: "network-test-key", SourceParticipantID: source, DestinationParticipantID: &destination, DestinationAccountID: &destinationAccount, ClearingAccountID: &clearingAccount, AmountMinor: 501, Currency: "USD", CorrelationID: uuid.New()}); err != ErrIdempotencyConflict {
		t.Fatalf("se esperaba un conflicto, se obtuvo %v", err)
	}
	otherAccount := uuid.New()
	if _, err := store.Create(ctx, Instruction{IdempotencyKey: "network-test-key", SourceParticipantID: source, DestinationParticipantID: &destination, DestinationAccountID: &otherAccount, ClearingAccountID: &clearingAccount, AmountMinor: 500, Currency: "USD", CorrelationID: uuid.New()}); err != ErrIdempotencyConflict {
		t.Fatalf("se esperaba conflicto por binding de cuenta, se obtuvo %v", err)
	}
}
