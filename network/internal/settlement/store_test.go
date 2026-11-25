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
		t.Skip("NETWORK_DB_URL es necesario para la prueba de integración de Docker")
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
	if _, err := db.ExecContext(ctx, "TRUNCATE reconciliation_difference, reconciliation_run, settlement_delivery, settlement_entry, settlement_journal, settlement_account, transfer_instruction CASCADE"); err != nil {
		t.Fatal(err)
	}
	return db, NewStore(db)
}

func readyInstruction(t *testing.T, db *sql.DB, transferID, source, destination uuid.UUID, amount int64, correlation uuid.UUID, deliveryAccounts ...uuid.UUID) {
	t.Helper()
	var destinationAccount, clearingAccount any
	if len(deliveryAccounts) == 2 {
		destinationAccount, clearingAccount = deliveryAccounts[0], deliveryAccounts[1]
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO transfer_instruction (transfer_id,idempotency_key,request_hash,source_participant_id,destination_participant_id,destination_account_id,clearing_account_id,amount_minor,currency,state,schema_version,correlation_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'USD','READY_FOR_SETTLEMENT',1,$9)`, transferID, "test-"+transferID.String(), instruction.RequestHashWithAccounts(source, &destination, deliveryUUID(deliveryAccounts, 0), deliveryUUID(deliveryAccounts, 1), amount, "USD"), source, destination, destinationAccount, clearingAccount, amount, correlation)
	if err != nil {
		t.Fatal(err)
	}
}

func deliveryUUID(values []uuid.UUID, index int) *uuid.UUID {
	if len(values) != 2 {
		return nil
	}
	value := values[index]
	return &value
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
	destinationAccount, clearingAccount := uuid.New(), uuid.New()
	readyInstruction(t, db, transferID, source, destination, 250, correlation, destinationAccount, clearingAccount)
	first, err := store.Settle(ctx, transferID, source, destination, 250, "settle-key", correlation, destinationAccount, clearingAccount)
	if err != nil {
		t.Fatal(err)
	}
	legacyHash := instruction.RequestHash(source, &destination, 250, "USD")
	if _, err := db.ExecContext(ctx, `UPDATE settlement_journal SET request_hash=$1 WHERE transfer_id=$2`, legacyHash, transferID); err != nil {
		t.Fatal(err)
	}
	second, err := store.Settle(ctx, uuid.New(), source, destination, 250, "settle-key", uuid.New(), destinationAccount, clearingAccount)
	if err != nil {
		t.Fatal(err)
	}
	if first.TransferID != second.TransferID || !first.Balanced() {
		t.Fatalf("diarios inesperados: %+v %+v", first, second)
	}
	if _, err := store.Settle(ctx, uuid.New(), source, destination, 250, "settle-key", uuid.New(), uuid.New(), clearingAccount); err != ErrIdempotencyConflict {
		t.Fatalf("se esperaba conflicto por binding de cuenta, se obtuvo %v", err)
	}
	var sourceBalance, destinationBalance int64
	if err := db.QueryRowContext(ctx, "SELECT balance_minor FROM settlement_account WHERE participant_id=$1", source).Scan(&sourceBalance); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT balance_minor FROM settlement_account WHERE participant_id=$1", destination).Scan(&destinationBalance); err != nil {
		t.Fatal(err)
	}
	if sourceBalance != 750 || destinationBalance != 250 {
		t.Fatalf("posiciones inesperadas: %d/%d", sourceBalance, destinationBalance)
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
			transferID, correlation := uuid.New(), uuid.New()
			readyInstruction(t, db, transferID, source, destination, 100, correlation)
			_, err := store.Settle(ctx, transferID, source, destination, 100, "concurrent-"+uuid.NewString(), correlation)
			if err == nil {
				mu.Lock()
				successful++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if successful != 10 {
		t.Fatalf("se esperaban 10 liquidaciones exitosas, se obtuvieron %d", successful)
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
