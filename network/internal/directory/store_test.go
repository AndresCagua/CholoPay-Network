package directory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresDirectoryResolvesWithinOwningNetwork(t *testing.T) {
	url := os.Getenv("NETWORK_DB_URL")
	if url == "" {
		t.Skip("NETWORK_DB_URL is required for the Docker integration test")
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
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS payment_network (network_id UUID PRIMARY KEY, code VARCHAR(64) NOT NULL UNIQUE, kind VARCHAR(16) NOT NULL, status VARCHAR(16) NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS network_participant (participant_id UUID PRIMARY KEY, network_id UUID NOT NULL REFERENCES payment_network(network_id), code VARCHAR(64) NOT NULL UNIQUE, status VARCHAR(16) NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE directory_key, network_participant, payment_network CASCADE"); err != nil {
		t.Fatal(err)
	}
	banking, fintech := uuid.New(), uuid.New()
	if _, err := db.ExecContext(ctx, "INSERT INTO payment_network(network_id,code,kind,status) VALUES ($1,'BANKING','BANKING','ACTIVE'),($2,'FINTECH','FINTECH','ACTIVE')", banking, fintech); err != nil {
		t.Fatal(err)
	}
	bankParticipant, walletParticipant := uuid.New(), uuid.New()
	if _, err := db.ExecContext(ctx, "INSERT INTO network_participant(participant_id,network_id,code,status) VALUES ($1,$2,'COSTA','ACTIVE'),($3,$4,'WALLET_A','ACTIVE')", bankParticipant, banking, walletParticipant, fintech); err != nil {
		t.Fatal(err)
	}
	key := Entry{KeyID: uuid.New(), NetworkID: banking, ParticipantID: bankParticipant, KeyType: "PHONE", KeyValue: "+593990000001", AccountRef: "costa-account-1", Status: "ACTIVE", Version: 1}
	if err := store.Register(ctx, key); err != nil {
		t.Fatal(err)
	}
	resolved, err := store.Resolve(ctx, banking, key.KeyType, key.KeyValue, uuid.New(), uuid.New(), "directory-test-1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Entry.ParticipantID != bankParticipant || resolved.MessageType != "DIRECTORY_KEY_RESOLVED" || resolved.SchemaVersion != "1.0" {
		t.Fatalf("unexpected resolution: %+v", resolved)
	}
	body, _ := json.Marshal(map[string]any{"network_id": banking, "key_type": key.KeyType, "key_value": key.KeyValue, "correlation_id": uuid.New(), "causation_id": uuid.New(), "idempotency_key": "http-directory-test"})
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/directory/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("expected HTTP resolution, got %d: %s", response.Code, response.Body.String())
	}
	if _, err := store.Resolve(ctx, fintech, key.KeyType, key.KeyValue, uuid.New(), uuid.New(), "directory-test-2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected network isolation, got %v", err)
	}
	duplicate := key
	duplicate.KeyID = uuid.New()
	duplicate.NetworkID = fintech
	duplicate.ParticipantID = walletParticipant
	if err := store.Register(ctx, duplicate); !errors.Is(err, ErrKeyConflict) {
		t.Fatalf("expected global identity conflict, got %v", err)
	}
	for index := 0; index < 4; index++ {
		entry := Entry{KeyID: uuid.New(), NetworkID: banking, ParticipantID: bankParticipant, KeyType: "RANDOM", KeyValue: uuid.NewString(), AccountRef: "costa-account-1", AccountKind: "PERSONAL", Status: "ACTIVE", Version: 1}
		if err := store.Register(ctx, entry); err != nil {
			t.Fatalf("register key %d: %v", index, err)
		}
	}
	if err := store.Register(ctx, Entry{KeyID: uuid.New(), NetworkID: banking, ParticipantID: bankParticipant, KeyType: "RANDOM", KeyValue: uuid.NewString(), AccountRef: "costa-account-1", AccountKind: "PERSONAL", Status: "ACTIVE", Version: 1}); !errors.Is(err, ErrKeyLimit) {
		t.Fatalf("expected personal key limit, got %v", err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE directory_key SET status='SUSPENDED' WHERE key_id=$1", key.KeyID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Resolve(ctx, banking, key.KeyType, key.KeyValue, uuid.New(), uuid.New(), "directory-test-3"); !errors.Is(err, ErrSuspended) {
		t.Fatalf("expected suspended key, got %v", err)
	}
}
