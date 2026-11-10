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

func TestPortFingerprintCoversAllOwnershipFields(t *testing.T) {
	base := PortRequest{SchemaVersion: "1.0", KeyType: "PHONE", KeyValue: "+593990000001", AccountRef: "account-1", AccountKind: "PERSONAL", FromNetworkID: uuid.New(), FromParticipantID: uuid.New(), ToNetworkID: uuid.New(), ToParticipantID: uuid.New(), NewKeyID: uuid.New()}
	first, err := fingerprintForPort(base)
	if err != nil {
		t.Fatal(err)
	}
	variants := []PortRequest{
		func() PortRequest { value := base; value.NewKeyID = uuid.New(); return value }(),
		func() PortRequest { value := base; value.ToParticipantID = uuid.New(); return value }(),
		func() PortRequest { value := base; value.AccountRef = "account-2"; return value }(),
	}
	for index, variant := range variants {
		fingerprint, err := fingerprintForPort(variant)
		if err != nil {
			t.Fatal(err)
		}
		if fingerprint == first || len(fingerprint) != 64 {
			t.Fatalf("variant %d did not produce a distinct SHA-256 fingerprint: %q", index, fingerprint)
		}
	}
}

func TestPortRequiresAuthorization(t *testing.T) {
	t.Setenv("DIRECTORY_PORT_TOKEN", "expected-token")
	mux := http.NewServeMux()
	NewHandler(nil).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/directory/port", bytes.NewReader([]byte(`{}`)))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("se esperaba una solicitud de portabilidad no autorizada, se obtuvo %d", response.Code)
	}
}

func TestPostgresDirectoryResolvesWithinOwningNetwork(t *testing.T) {
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
	if _, err := db.ExecContext(ctx, "TRUNCATE directory_key, directory_key_operations, network_participant, payment_network CASCADE"); err != nil {
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
		t.Fatalf("resolución inesperada: %+v", resolved)
	}
	body, _ := json.Marshal(map[string]any{"network_id": banking, "key_type": key.KeyType, "key_value": key.KeyValue, "correlation_id": uuid.New(), "causation_id": uuid.New(), "idempotency_key": "http-directory-test"})
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/directory/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("se esperaba una resolución HTTP, se obtuvo %d: %s", response.Code, response.Body.String())
	}
	if _, err := store.Resolve(ctx, fintech, key.KeyType, key.KeyValue, uuid.New(), uuid.New(), "directory-test-2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("se esperaba aislamiento de red, se obtuvo %v", err)
	}
	duplicate := key
	duplicate.KeyID = uuid.New()
	duplicate.NetworkID = fintech
	duplicate.ParticipantID = walletParticipant
	if err := store.Register(ctx, duplicate); !errors.Is(err, ErrKeyConflict) {
		t.Fatalf("se esperaba un conflicto de identidad global, se obtuvo %v", err)
	}
	for index := 0; index < 4; index++ {
		entry := Entry{KeyID: uuid.New(), NetworkID: banking, ParticipantID: bankParticipant, KeyType: "RANDOM", KeyValue: uuid.NewString(), AccountRef: "costa-account-1", AccountKind: "PERSONAL", Status: "ACTIVE", Version: 1}
		if err := store.Register(ctx, entry); err != nil {
			t.Fatalf("register key %d: %v", index, err)
		}
	}
	if err := store.Register(ctx, Entry{KeyID: uuid.New(), NetworkID: banking, ParticipantID: bankParticipant, KeyType: "RANDOM", KeyValue: uuid.NewString(), AccountRef: "costa-account-1", AccountKind: "PERSONAL", Status: "ACTIVE", Version: 1}); !errors.Is(err, ErrKeyLimit) {
		t.Fatalf("se esperaba el límite de claves personales, se obtuvo %v", err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE directory_key SET status='SUSPENDED' WHERE key_id=$1", key.KeyID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Resolve(ctx, banking, key.KeyType, key.KeyValue, uuid.New(), uuid.New(), "directory-test-3"); !errors.Is(err, ErrSuspended) {
		t.Fatalf("se esperaba una clave suspendida, se obtuvo %v", err)
	}
	portValue := "+593990000002"
	portKey := Entry{KeyID: uuid.New(), NetworkID: banking, ParticipantID: bankParticipant, KeyType: "PHONE", KeyValue: portValue, AccountRef: "costa-account-2", AccountKind: "PERSONAL", Status: "ACTIVE", Version: 1}
	if err := store.Register(ctx, portKey); err != nil {
		t.Fatal(err)
	}
	portRequest := PortRequest{IdempotencyKey: "port-001", KeyType: portKey.KeyType, KeyValue: portValue, AccountRef: portKey.AccountRef, AccountKind: "PERSONAL", FromNetworkID: banking, FromParticipantID: bankParticipant, ToNetworkID: fintech, ToParticipantID: walletParticipant, NewKeyID: uuid.New()}
	if err := store.Port(ctx, portRequest); err != nil {
		t.Fatal(err)
	}
	if err := store.Port(ctx, portRequest); err != nil {
		t.Fatalf("se esperaba una repetición idempotente de portabilidad, se obtuvo %v", err)
	}
	if _, err := store.Resolve(ctx, banking, portKey.KeyType, portValue, uuid.New(), uuid.New(), "directory-port-old"); !errors.Is(err, ErrSuspended) {
		t.Fatalf("se esperaba que el propietario anterior estuviera suspendido, se obtuvo %v", err)
	}
	if resolved, err := store.Resolve(ctx, fintech, portKey.KeyType, portValue, uuid.New(), uuid.New(), "directory-port-new"); err != nil || resolved.Entry.ParticipantID != walletParticipant {
		t.Fatalf("se esperaba la resolución del nuevo propietario, se obtuvo %+v, %v", resolved, err)
	}
}
