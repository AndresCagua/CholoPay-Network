package orchestration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cholopay/network/internal/instruction"
	"github.com/cholopay/network/internal/message"
	"github.com/cholopay/network/internal/routing"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresOrchestrationRoutesAndReplaysWithoutSettlement(t *testing.T) {
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
	instructions := instruction.NewStore(db)
	routes := routing.NewStore(db)
	messages := message.NewStore(db)
	if err := instruction.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := routes.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := messages.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE inter_network_message, transfer_instruction, network_participant, payment_network CASCADE"); err != nil {
		t.Fatal(err)
	}
	banking, fintech := uuid.New(), uuid.New()
	if _, err := db.ExecContext(ctx, "INSERT INTO payment_network(network_id,code,kind,status) VALUES ($1,'BANKING-ORCH','BANKING','ACTIVE'),($2,'FINTECH-ORCH','FINTECH','ACTIVE')", banking, fintech); err != nil {
		t.Fatal(err)
	}
	costa, sierra, wallet := uuid.New(), uuid.New(), uuid.New()
	if _, err := db.ExecContext(ctx, "INSERT INTO network_participant(participant_id,network_id,code,status) VALUES ($1,$2,'COSTA-ORCH','ACTIVE'),($3,$2,'SIERRA-ORCH','ACTIVE'),($4,$5,'WALLET-ORCH','ACTIVE')", costa, banking, sierra, wallet, fintech); err != nil {
		t.Fatal(err)
	}
	service := NewService(instructions, routes, messages)
	intra, err := service.Intake(ctx, Intake{IdempotencyKey: "orch-intra", SourceParticipantID: costa, DestinationParticipantID: sierra, AmountMinor: 100, Currency: "USD", CorrelationID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if intra.Route.Kind != routing.IntraNetwork || intra.Message != nil {
		t.Fatalf("resultado intrared inesperado: %+v", intra)
	}
	interRequest := Intake{IdempotencyKey: "orch-inter", SourceParticipantID: costa, DestinationParticipantID: wallet, AmountMinor: 200, Currency: "USD", CorrelationID: uuid.New()}
	first, err := service.Intake(ctx, interRequest)
	if err != nil {
		t.Fatal(err)
	}
	if first.Route.Kind != routing.InterNetwork || first.Message == nil || first.Message.Replayed {
		t.Fatalf("primer resultado interred inesperado: %+v", first)
	}
	second, err := service.Intake(ctx, interRequest)
	if err != nil {
		t.Fatal(err)
	}
	if second.Instruction.TransferID != first.Instruction.TransferID || second.Message == nil || !second.Message.Replayed {
		t.Fatalf("repetición inesperada result: %+v", second)
	}
	var messageCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM inter_network_message").Scan(&messageCount); err != nil {
		t.Fatal(err)
	}
	if messageCount != 1 {
		t.Fatalf("se esperaba un mensaje interred, se obtuvo %d", messageCount)
	}
	var settlementCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM transfer_instruction WHERE state='SETTLED'").Scan(&settlementCount); err != nil {
		t.Fatal(err)
	}
	if settlementCount != 0 {
		t.Fatalf("la recepción no debe liquidar, se obtuvo %d", settlementCount)
	}
	body, _ := json.Marshal(map[string]any{"idempotency_key": "orch-http", "source_participant_id": costa, "destination_participant_id": wallet, "amount_minor": 300, "currency": "USD", "correlation_id": uuid.New()})
	mux := http.NewServeMux()
	NewHandler(service).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/inter-network/transfers", bytes.NewReader(body))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusAccepted {
		t.Fatalf("se esperaba HTTP aceptado, se obtuvo %d: %s", response.Code, response.Body.String())
	}
}
