package settlement

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReplayHTTPRequiresOperatorBearer(t *testing.T) {
	t.Setenv("OPERATOR_CONSOLE_TOKEN", "operator-test-token")
	mux := http.NewServeMux()
	NewHandler(nil).Register(mux)
	for _, authorization := range []string{"", "wrong-token", "operator-test-token", "Basic operator-test-token", "Bearer wrong-token"} {
		req := httptest.NewRequest(http.MethodPost, "/internal/v1/settlement-deliveries/00000000-0000-4000-8000-000000000001/replay", nil)
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("Authorization %q: se esperaba 401, se obtuvo %d", authorization, response.Code)
		}
	}
}

func TestReplayHTTPRejectsMissingOperatorTokenConfiguration(t *testing.T) {
	t.Setenv("OPERATOR_CONSOLE_TOKEN", "")
	mux := http.NewServeMux()
	NewHandler(nil).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/settlement-deliveries/00000000-0000-4000-8000-000000000001/replay", nil)
	req.Header.Set("Authorization", "Bearer operator-test-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("sin token configurado: se esperaba 401, se obtuvo %d", response.Code)
	}
}

func TestSettlementHTTPReportsSettledOnlyAfterJournalCommit(t *testing.T) {
	db, store := testStore(t)
	defer db.Close()
	ctx := context.Background()
	source, destination := uuid.New(), uuid.New()
	destinationAccount, clearingAccount := uuid.New(), uuid.New()
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
	payload, _ := json.Marshal(map[string]any{"transfer_id": uuid.New(), "source_participant_id": source, "destination_participant_id": destination, "amount_minor": 200, "idempotency_key": "http-settlement-key", "correlation_id": uuid.New(), "destination_account_id": destinationAccount, "clearing_account_id": clearingAccount})
	response, err := http.Post(server.URL+"/internal/v1/settlements", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		var detail []byte
		detail, _ = io.ReadAll(response.Body)
		t.Fatalf("se esperaba 201, se obtuvo %d: %s", response.StatusCode, detail)
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
