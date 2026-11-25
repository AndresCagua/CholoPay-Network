package settlement

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

type settleRequest struct {
	TransferID               uuid.UUID `json:"transfer_id"`
	SourceParticipantID      uuid.UUID `json:"source_participant_id"`
	DestinationParticipantID uuid.UUID `json:"destination_participant_id"`
	AmountMinor              int64     `json:"amount_minor"`
	IdempotencyKey           string    `json:"idempotency_key"`
	CorrelationID            uuid.UUID `json:"correlation_id"`
	DestinationAccountID     uuid.UUID `json:"destination_account_id"`
	ClearingAccountID        uuid.UUID `json:"clearing_account_id"`
}

type settleResponse struct {
	TransferID uuid.UUID `json:"transfer_id"`
	State      string    `json:"state"`
	JournalID  uuid.UUID `json:"journal_id"`
}

type operatorAccountRequest struct {
	ParticipantID       uuid.UUID `json:"participant_id"`
	OpeningBalanceMinor int64     `json:"opening_balance_minor"`
	Currency            string    `json:"currency"`
	Reason              string    `json:"reason"`
}

type operatorAccountResponse struct {
	OperationID      uuid.UUID `json:"funding_operation_id,omitempty"`
	FundingJournalID uuid.UUID `json:"funding_journal_id,omitempty"`
	ParticipantID    uuid.UUID `json:"participant_id"`
	BalanceMinor     int64     `json:"balance_minor"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	CreatedAt        string    `json:"created_at,omitempty"`
	UpdatedAt        string    `json:"updated_at,omitempty"`
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/settlements", h.create)
	mux.HandleFunc("POST /internal/v1/operator/settlement-accounts", h.createOperatorAccount)
	mux.HandleFunc("GET /internal/v1/operator/settlement-accounts/{participant_id}", h.getOperatorAccount)
	mux.HandleFunc("GET /internal/v1/operator/participants/{participant_id}/settlement-history", h.getSettlementHistory)
	mux.HandleFunc("GET /internal/v1/operator/settlement-deliveries/incidents", h.listOperatorIncidents)
	mux.HandleFunc("GET /internal/v1/operator/settlement-deliveries/{id}/actions", h.getOperatorActions)
	mux.HandleFunc("POST /internal/v1/settlement-deliveries/{id}/replay", h.replay)
}
func (h *Handler) createOperatorAccount(w http.ResponseWriter, r *http.Request) {
	if !authorizedReplayBearer(r) {
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	var request operatorAccountRequest
	if json.NewDecoder(r.Body).Decode(&request) != nil {
		http.Error(w, "solicitud de fondeo inválida", http.StatusBadRequest)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	result, err := h.store.CreateOperatorAccount(r.Context(), OperatorAccountRequest{
		ParticipantID: request.ParticipantID, OpeningBalanceMinor: request.OpeningBalanceMinor,
		Currency: request.Currency, IdempotencyKey: idempotencyKey, Reason: request.Reason,
	})
	if err == ErrInvalidFunding {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err == ErrFundingIdempotencyConflict || err == ErrAccountExists {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err == ErrParticipantUnavailable {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "no se pudo crear la cuenta de liquidación", http.StatusServiceUnavailable)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(operatorAccountResponse{OperationID: result.OperationID, FundingJournalID: result.FundingJournalID, ParticipantID: result.ParticipantID, BalanceMinor: result.BalanceMinor, Currency: result.Currency, Status: result.Status})
}

func (h *Handler) getOperatorAccount(w http.ResponseWriter, r *http.Request) {
	if !authorizedReplayBearer(r) {
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	participantID, err := uuid.Parse(r.PathValue("participant_id"))
	if err != nil {
		http.Error(w, "participant_id inválido", http.StatusBadRequest)
		return
	}
	account, err := h.store.SettlementAccount(r.Context(), participantID)
	if err == sql.ErrNoRows {
		http.Error(w, "la cuenta de liquidación no existe", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "la cuenta de liquidación no está disponible", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorAccountResponse{
		ParticipantID: account.ParticipantID, BalanceMinor: account.BalanceMinor,
		Currency: account.Currency, Status: "ACTIVE",
		CreatedAt: account.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: account.UpdatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) getSettlementHistory(w http.ResponseWriter, r *http.Request) {
	if !authorizedReplayBearer(r) {
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	participantID, err := uuid.Parse(r.PathValue("participant_id"))
	if err != nil {
		http.Error(w, "participant_id inválido", http.StatusBadRequest)
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 500 {
			http.Error(w, "limit inválido", http.StatusBadRequest)
			return
		}
	}
	entries, err := h.store.SettlementHistory(r.Context(), participantID, limit)
	if err == sql.ErrNoRows {
		http.Error(w, "la cuenta de liquidación no existe", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "el historial de liquidación no está disponible", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"participant_id": participantID, "data": entries})
}

func (h *Handler) listOperatorIncidents(w http.ResponseWriter, r *http.Request) {
	if !authorizedReplayBearer(r) {
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	incidents, err := h.store.ListOperatorIncidents(r.Context())
	if err != nil {
		http.Error(w, "las incidencias de entregas no están disponibles", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": incidents})
}

func (h *Handler) getOperatorActions(w http.ResponseWriter, r *http.Request) {
	if !authorizedReplayBearer(r) {
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	deliveryID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "identificador de entrega inválido", http.StatusBadRequest)
		return
	}
	actions, err := h.store.ListOperatorActions(r.Context(), deliveryID)
	if err != nil {
		http.Error(w, "las acciones del operador no están disponibles", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"delivery_id": deliveryID, "actions": actions})
}

func (h *Handler) replay(w http.ResponseWriter, r *http.Request) {
	if !authorizedReplayBearer(r) {
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	deliveryID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "identificador de entrega inválido", http.StatusBadRequest)
		return
	}
	delivery, err := h.store.ReplayDelivery(r.Context(), deliveryID)
	if err == sql.ErrNoRows {
		http.Error(w, "entrega no encontrada", http.StatusNotFound)
		return
	}
	if err == ErrDeliveryNotReplayable {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "falló el replay de la entrega", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"delivery_id": delivery.DeliveryID, "event_id": delivery.EventID, "state": delivery.State, "attempt_count": delivery.AttemptCount})
}

func authorizedReplayBearer(r *http.Request) bool {
	expected := strings.TrimSpace(os.Getenv("OPERATOR_CONSOLE_TOKEN"))
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	value, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || expected == "" || value == "" || strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	return len(value) == len(expected) && subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var request settleRequest
	if json.NewDecoder(r.Body).Decode(&request) != nil || request.TransferID == uuid.Nil || request.SourceParticipantID == uuid.Nil || request.DestinationParticipantID == uuid.Nil || request.AmountMinor <= 0 || request.IdempotencyKey == "" {
		http.Error(w, "solicitud de liquidación inválida", http.StatusBadRequest)
		return
	}
	if request.DestinationAccountID == uuid.Nil || request.ClearingAccountID == uuid.Nil || request.DestinationAccountID == request.ClearingAccountID {
		http.Error(w, "se requieren cuentas de destino y compensación válidas y distintas", http.StatusBadRequest)
		return
	}
	journal, err := h.store.Settle(r.Context(), request.TransferID, request.SourceParticipantID, request.DestinationParticipantID, request.AmountMinor, request.IdempotencyKey, request.CorrelationID, request.DestinationAccountID, request.ClearingAccountID)
	if err == ErrIdempotencyConflict {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(settleResponse{TransferID: journal.TransferID, State: "SETTLED", JournalID: journal.JournalID})
}
