package settlement

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"os"
	"strings"
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

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/settlements", h.create)
	mux.HandleFunc("POST /internal/v1/settlement-deliveries/{id}/replay", h.replay)
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
	expected := os.Getenv("OPERATOR_CONSOLE_TOKEN")
	value := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	return expected != "" && len(value) == len(expected) && subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1
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
