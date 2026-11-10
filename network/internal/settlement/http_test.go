package settlement

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSettlementHTTPReportsSettledOnlyAfterJournalCommit(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, destination := uuid.New(), uuid.New()
	if err := store.CreateAccount(ctx, source, 500); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, destination, 0); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	payload, _ := json.Marshal(map[string]any{"transfer_id": uuid.New(), "source_participant_id": source, "destination_participant_id": destination, "amount_minor": 200, "idempotency_key": "http-settlement-key", "correlation_id": uuid.New()})
	response, err := http.Post(server.URL+"/internal/v1/settlements", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("se esperaba 201, se obtuvo %d", response.StatusCode)
	}
	var body struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.State != "SETTLED" {
		t.Fatalf("estado inesperado %q", body.State)
	}
}
