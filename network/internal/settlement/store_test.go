package settlement

import (
	"context"
	"database/sql"
	"github.com/cholopay/network/internal/instruction"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"os"
	"sync"
	"testing"
)

func testStore(t *testing.T) (*sql.DB, *Store) {
	t.Helper()
	url := os.Getenv("NETWORK_DB_URL")
	if url == "" {
		t.Skip("NETWORK_DB_URL is required for the Docker integration test")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	if err := instruction.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE reconciliation_difference, reconciliation_run, settlement_delivery, settlement_entry, settlement_journal, settlement_account, transfer_instruction"); err != nil {
		t.Fatal(err)
	}
	return db, NewStore(db)
}

func TestSettlementIsBalancedAndIdempotent(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, destination := uuid.New(), uuid.New()
	if err := store.CreateAccount(ctx, source, 1000); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, destination, 0); err != nil {
		t.Fatal(err)
	}
	transferID, correlation := uuid.New(), uuid.New()
	first, err := store.Settle(ctx, transferID, source, destination, 250, "settle-key", correlation)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Settle(ctx, uuid.New(), source, destination, 250, "settle-key", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if first.TransferID != second.TransferID || !first.Balanced() {
		t.Fatalf("unexpected journals: %+v %+v", first, second)
	}
	var sourceBalance, destinationBalance int64
	if err := db.QueryRowContext(ctx, "SELECT balance_minor FROM settlement_account WHERE participant_id=$1", source).Scan(&sourceBalance); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT balance_minor FROM settlement_account WHERE participant_id=$1", destination).Scan(&destinationBalance); err != nil {
		t.Fatal(err)
	}
	if sourceBalance != 750 || destinationBalance != 250 {
		t.Fatalf("unexpected positions: %d/%d", sourceBalance, destinationBalance)
	}
}

func TestConcurrentSettlementsCannotOverdraw(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, destination := uuid.New(), uuid.New()
	if err := store.CreateAccount(ctx, source, 1000); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, destination, 0); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	successful := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := store.Settle(ctx, uuid.New(), source, destination, 100, "concurrent-"+uuid.NewString(), uuid.New())
			if err == nil {
				mu.Lock()
				successful++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if successful != 10 {
		t.Fatalf("expected 10 successful settlements, got %d", successful)
	}
	var sourceBalance, destinationBalance int64
	if err := db.QueryRowContext(ctx, "SELECT balance_minor FROM settlement_account WHERE participant_id=$1", source).Scan(&sourceBalance); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT balance_minor FROM settlement_account WHERE participant_id=$1", destination).Scan(&destinationBalance); err != nil {
		t.Fatal(err)
	}
	if sourceBalance < 0 || sourceBalance+destinationBalance != 1000 {
		t.Fatalf("conservation failure: %d/%d", sourceBalance, destinationBalance)
	}
}
