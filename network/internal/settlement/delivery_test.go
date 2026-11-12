package settlement

import (
	"context"
	"github.com/cholopay/network/internal/instruction"
	"github.com/cholopay/network/internal/transfer"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"testing"
)

func TestDeliveryIsAtLeastOnceAndAcknowledgementIsIdempotent(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, receiver := uuid.New(), uuid.New()
	transferID := uuid.New()
	if err := store.CreateAccount(ctx, source, 1000); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, receiver, 0); err != nil {
		t.Fatal(err)
	}
	instructions := instruction.NewStore(db)
	if _, err := instructions.Create(ctx, instruction.Instruction{TransferID: transferID, IdempotencyKey: "delivery-instruction", SourceParticipantID: source, DestinationParticipantID: &receiver, AmountMinor: 250, Currency: "USD", CorrelationID: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	for _, state := range []transfer.State{transfer.Validating, transfer.FundsReserved, transfer.ReadyForSettlement} {
		if err := instructions.Advance(ctx, transferID, state, uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Settle(ctx, transferID, source, receiver, 250, "delivery-settlement", uuid.New()); err != nil {
		t.Fatal(err)
	}
	first, err := store.EnqueueDelivery(ctx, transferID, receiver)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.EnqueueDelivery(ctx, transferID, receiver)
	if err != nil {
		t.Fatal(err)
	}
	if first.DeliveryID != replay.DeliveryID || first.State != "PENDING" {
		t.Fatalf("repetición de entrega inesperada: %+v %+v", first, replay)
	}
	eventID := first.EventID
	applied, err := store.AcknowledgeDelivery(ctx, first.DeliveryID, receiver, eventID)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := store.AcknowledgeDelivery(ctx, first.DeliveryID, receiver, eventID)
	if err != nil {
		t.Fatal(err)
	}
	if applied.State != "APPLIED" || duplicate.State != "APPLIED" {
		t.Fatalf("confirmación inesperada: %+v %+v", applied, duplicate)
	}
	if _, err := store.AcknowledgeDelivery(ctx, first.DeliveryID, receiver, uuid.New()); err != ErrDeliveryConflict {
		t.Fatalf("se esperaba el rechazo del evento conflictivo, se obtuvo %v", err)
	}
	if _, err := instructions.Get(ctx, transferID); err != nil {
		t.Fatal(err)
	}
}

func TestDeliveryRejectsWrongReceiver(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, receiver, wrong := uuid.New(), uuid.New(), uuid.New()
	transferID := uuid.New()
	if err := store.CreateAccount(ctx, source, 100); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, receiver, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, wrong, 0); err != nil {
		t.Fatal(err)
	}
	instructions := instruction.NewStore(db)
	if _, err := instructions.Create(ctx, instruction.Instruction{TransferID: transferID, IdempotencyKey: "wrong-receiver-instruction", SourceParticipantID: source, DestinationParticipantID: &receiver, AmountMinor: 50, Currency: "USD", CorrelationID: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	for _, state := range []transfer.State{transfer.Validating, transfer.FundsReserved, transfer.ReadyForSettlement} {
		if err := instructions.Advance(ctx, transferID, state, uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Settle(ctx, transferID, source, receiver, 50, "wrong-receiver-settlement", uuid.New()); err != nil {
		t.Fatal(err)
	}
	delivery, err := store.EnqueueDelivery(ctx, transferID, receiver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcknowledgeDelivery(ctx, delivery.DeliveryID, wrong, uuid.New()); err != ErrDeliveryConflict {
		t.Fatalf("se esperaba el rechazo del receptor incorrecto, se obtuvo %v", err)
	}
}
