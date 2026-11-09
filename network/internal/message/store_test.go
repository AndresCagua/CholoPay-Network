package message

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

func TestPostgresInterNetworkMessageIsIdempotentAndBounded(t *testing.T) {
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
	store := NewStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE inter_network_message, payment_network CASCADE"); err != nil {
		t.Fatal(err)
	}
	banking, fintech, suspended := uuid.New(), uuid.New(), uuid.New()
	if _, err := db.ExecContext(ctx, "INSERT INTO payment_network(network_id,code,kind,status) VALUES ($1,'BANKING','BANKING','ACTIVE'),($2,'FINTECH','FINTECH','ACTIVE'),($3,'SUSPENDED','FINTECH','SUSPENDED')", banking, fintech, suspended); err != nil {
		t.Fatal(err)
	}
	envelope := Envelope{MessageID: uuid.New(), IdempotencyKey: "message-test-1", SenderNetworkID: banking, DestinationNetworkID: fintech, SchemaVersion: "1.0", MessageType: "TRANSFER_REQUESTED", CorrelationID: uuid.New(), CausationID: uuid.New(), Payload: json.RawMessage(`{"transfer_id":"synthetic-transfer","amount_minor":100}`)}
	accepted, err := store.Accept(ctx, envelope)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Replayed || accepted.Status != "ACCEPTED" || accepted.PayloadHash == "" {
		t.Fatalf("unexpected acceptance: %+v", accepted)
	}
	replayed, err := store.Accept(ctx, envelope)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.Envelope.MessageID != envelope.MessageID {
		t.Fatalf("unexpected replay: %+v", replayed)
	}
	conflict := envelope
	conflict.Payload = json.RawMessage(`{"transfer_id":"synthetic-transfer","amount_minor":101}`)
	if _, err := store.Accept(ctx, conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflicting replay, got %v", err)
	}
	invalid := envelope
	invalid.MessageID = uuid.New()
	invalid.IdempotencyKey = "message-test-inactive"
	invalid.SenderNetworkID = suspended
	if _, err := store.Accept(ctx, invalid); !errors.Is(err, ErrBoundary) {
		t.Fatalf("expected inactive boundary rejection, got %v", err)
	}
	body, _ := json.Marshal(envelope)
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/inter-network/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("expected HTTP replay, got %d: %s", response.Code, response.Body.String())
	}
}
