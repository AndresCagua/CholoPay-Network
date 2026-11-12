package settlement

import (
	"context"
	"github.com/cholopay/network/internal/instruction"
	"github.com/cholopay/network/internal/transfer"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"testing"
)

func TestReconciliationDetectsAndClearsDeliveryDifferences(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, receiver, transferID := uuid.New(), uuid.New(), uuid.New()
	if err := store.CreateAccount(ctx, source, 300); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, receiver, 0); err != nil {
		t.Fatal(err)
	}
	instructions := instruction.NewStore(db)
	if _, err := instructions.Create(ctx, instruction.Instruction{TransferID: transferID, IdempotencyKey: "reconciliation-instruction", SourceParticipantID: source, DestinationParticipantID: &receiver, AmountMinor: 100, Currency: "USD", CorrelationID: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	for _, state := range []transfer.State{transfer.Validating, transfer.FundsReserved, transfer.ReadyForSettlement} {
		if err := instructions.Advance(ctx, transferID, state, uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Settle(ctx, transferID, source, receiver, 100, "reconciliation-settlement", uuid.New()); err != nil {
		t.Fatal(err)
	}
	missing, err := store.Reconcile(ctx, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(missing, "SETTLED_WITHOUT_DELIVERY") {
		t.Fatalf("se esperaba una entrega ausente, se obtuvo %+v", missing)
	}
	delivery, err := store.EnqueueDelivery(ctx, transferID, receiver)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := store.Reconcile(ctx, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(pending, "DELIVERY_PENDING") {
		t.Fatalf("se esperaba una entrega pendiente, se obtuvo %+v", pending)
	}
	if _, err := store.AcknowledgeDelivery(ctx, delivery.DeliveryID, receiver, delivery.EventID); err != nil {
		t.Fatal(err)
	}
	clean, err := store.Reconcile(ctx, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if clean.DifferenceCount != 0 {
		t.Fatalf("se esperaba una reconciliación limpia, se obtuvo %+v", clean)
	}
}

func hasCode(run ReconciliationRun, code string) bool {
	for _, difference := range run.Differences {
		if difference.Code == code {
			return true
		}
	}
	return false
}
