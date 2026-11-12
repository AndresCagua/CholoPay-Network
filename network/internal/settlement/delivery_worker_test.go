package settlement

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cholopay/network/internal/instruction"
	"github.com/cholopay/network/internal/transfer"
	"github.com/google/uuid"
)

func TestWorkerEnqueuesAfterSettlementBeforeDispatch(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, receiver, transferID := uuid.New(), uuid.New(), uuid.New()
	if err := store.CreateAccount(ctx, source, 500); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, receiver, 0); err != nil {
		t.Fatal(err)
	}
	instructions := instruction.NewStore(db)
	if _, err := instructions.Create(ctx, instruction.Instruction{TransferID: transferID, IdempotencyKey: "worker-trigger", SourceParticipantID: source, DestinationParticipantID: &receiver, AmountMinor: 100, Currency: "USD", CorrelationID: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	for _, state := range []transfer.State{transfer.Validating, transfer.FundsReserved, transfer.ReadyForSettlement} {
		if err := instructions.Advance(ctx, transferID, state, uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	destinationAccount, clearingAccount := uuid.New(), uuid.New()
	if _, err := store.Settle(ctx, transferID, source, receiver, 100, "worker-settlement", uuid.New(), destinationAccount, clearingAccount); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM settlement_delivery WHERE transfer_id=$1", transferID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("settlement silently enqueued delivery: %d", count)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	dispatcher := &DeliveryDispatcher{Store: store, Client: server.Client(), ReceiverURL: server.URL, Secret: "test", Lease: time.Minute}
	if handled, err := dispatcher.DispatchOnce(ctx); err != nil || !handled {
		t.Fatalf("dispatch failed: handled=%v err=%v", handled, err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM settlement_delivery WHERE transfer_id=$1 AND state='APPLIED'", transferID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one applied delivery, got %d", count)
	}
}
