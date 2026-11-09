package instruction

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestInternalInstructionHTTP(t *testing.T) {
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
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE transfer_instruction"); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewHandler(NewStore(db)).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	source, destination := uuid.New(), uuid.New()
	payload, _ := json.Marshal(map[string]any{"idempotency_key": "http-key", "source_participant_id": source, "destination_participant_id": destination, "amount_minor": 700, "currency": "USD", "correlation_id": uuid.New()})
	response, err := http.Post(server.URL+"/internal/v1/transfers", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", response.StatusCode)
	}
}
