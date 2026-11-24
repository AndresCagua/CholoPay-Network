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

func TestReplayDeliveryPreservesIdentityAttemptsAndIsIdempotent(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, receiver, transferID := uuid.New(), uuid.New(), uuid.New()
	if err := store.CreateAccount(ctx, source, 1000); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, receiver, 0); err != nil {
		t.Fatal(err)
	}
	instructions := instruction.NewStore(db)
	if _, err := instructions.Create(ctx, instruction.Instruction{TransferID: transferID, IdempotencyKey: "replay-instruction", SourceParticipantID: source, DestinationParticipantID: &receiver, AmountMinor: 250, Currency: "USD", CorrelationID: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	for _, state := range []transfer.State{transfer.Validating, transfer.FundsReserved, transfer.ReadyForSettlement} {
		if err := instructions.Advance(ctx, transferID, state, uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Settle(ctx, transferID, source, receiver, 250, "replay-settlement", uuid.New()); err != nil {
		t.Fatal(err)
	}
	delivery, err := store.EnqueueDelivery(ctx, transferID, receiver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE settlement_delivery SET state='FAILED',failure_code='DELIVERY_DLQ',failure_detail='outage',attempt_count=5 WHERE delivery_id=$1`, delivery.DeliveryID); err != nil {
		t.Fatal(err)
	}
	replayed, err := store.ReplayDelivery(ctx, delivery.DeliveryID)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.State != "PENDING" || replayed.DeliveryID != delivery.DeliveryID || replayed.EventID != delivery.EventID || replayed.AttemptCount != 5 {
		t.Fatalf("replay alteró el contrato: %+v original=%+v", replayed, delivery)
	}
	repeated, err := store.ReplayDelivery(ctx, delivery.DeliveryID)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.DeliveryID != delivery.DeliveryID || repeated.EventID != delivery.EventID || repeated.AttemptCount != 5 || repeated.State != "PENDING" {
		t.Fatalf("replay no idempotente: %+v", repeated)
	}
	var instructionState string
	if err := db.QueryRowContext(ctx, `SELECT state FROM transfer_instruction WHERE transfer_id=$1`, transferID).Scan(&instructionState); err != nil {
		t.Fatal(err)
	}
	if instructionState != "CREDIT_PENDING" {
		t.Fatalf("estado de transferencia inesperado: %s", instructionState)
	}
}

func TestIndependentDeliveriesApplyInReverseArrivalOrderIdempotently(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, receiver := uuid.New(), uuid.New()
	if err := store.CreateAccount(ctx, source, 1000); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, receiver, 0); err != nil {
		t.Fatal(err)
	}
	instructions := instruction.NewStore(db)
	transfers := []struct {
		id     uuid.UUID
		amount int64
		key    string
	}{
		{uuid.New(), 300, "reverse-second"},
		{uuid.New(), 200, "reverse-first"},
	}
	for _, item := range transfers {
		if _, err := instructions.Create(ctx, instruction.Instruction{TransferID: item.id, IdempotencyKey: item.key, SourceParticipantID: source, DestinationParticipantID: &receiver, AmountMinor: item.amount, Currency: "USD", CorrelationID: uuid.New()}); err != nil {
			t.Fatal(err)
		}
		for _, state := range []transfer.State{transfer.Validating, transfer.FundsReserved, transfer.ReadyForSettlement} {
			if err := instructions.Advance(ctx, item.id, state, uuid.New()); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := store.Settle(ctx, item.id, source, receiver, item.amount, item.key+"-settlement", uuid.New()); err != nil {
			t.Fatal(err)
		}
	}

	// Apply the independent deliveries in reverse arrival order; no global order is asserted.
	second, err := store.EnqueueDelivery(ctx, transfers[0].id, receiver)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.EnqueueDelivery(ctx, transfers[1].id, receiver)
	if err != nil {
		t.Fatal(err)
	}
	for _, delivery := range []Delivery{second, first} {
		if _, err := store.AcknowledgeDelivery(ctx, delivery.DeliveryID, receiver, delivery.EventID); err != nil {
			t.Fatal(err)
		}
		if _, err := store.AcknowledgeDelivery(ctx, delivery.DeliveryID, receiver, delivery.EventID); err != nil {
			t.Fatal(err)
		}
	}

	var sourceBalance, receiverBalance int64
	if err := db.QueryRowContext(ctx, "SELECT balance_minor FROM settlement_account WHERE participant_id=$1", source).Scan(&sourceBalance); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT balance_minor FROM settlement_account WHERE participant_id=$1", receiver).Scan(&receiverBalance); err != nil {
		t.Fatal(err)
	}
	if sourceBalance != 500 || receiverBalance != 500 {
		t.Fatalf("posiciones inesperadas: %d/%d", sourceBalance, receiverBalance)
	}
	var deliveries, applied, debits, credits int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE state='APPLIED') FROM settlement_delivery`).Scan(&deliveries, &applied); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE direction='DEBIT'), COUNT(*) FILTER (WHERE direction='CREDIT') FROM settlement_entry`).Scan(&debits, &credits); err != nil {
		t.Fatal(err)
	}
	if deliveries != 2 || applied != 2 || debits != 2 || credits != 2 {
		t.Fatalf("evidencia inesperada: deliveries=%d applied=%d debit=%d credit=%d", deliveries, applied, debits, credits)
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
