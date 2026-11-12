package instruction

import (
	"encoding/json"
	"github.com/cholopay/network/internal/transfer"
	"github.com/google/uuid"
	"net/http"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

type createRequest struct {
	TransferID               uuid.UUID  `json:"transfer_id"`
	IdempotencyKey           string     `json:"idempotency_key"`
	SourceParticipantID      uuid.UUID  `json:"source_participant_id"`
	DestinationParticipantID *uuid.UUID `json:"destination_participant_id"`
	DestinationAccountID     *uuid.UUID `json:"destination_account_id"`
	ClearingAccountID        *uuid.UUID `json:"clearing_account_id"`
	AmountMinor              int64      `json:"amount_minor"`
	Currency                 string     `json:"currency"`
	CorrelationID            uuid.UUID  `json:"correlation_id"`
}
type transitionRequest struct {
	State       transfer.State `json:"state"`
	CausationID uuid.UUID      `json:"causation_id"`
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/transfers", h.create)
	mux.HandleFunc("GET /internal/v1/transfers/{id}", h.get)
	mux.HandleFunc("POST /internal/v1/transfers/{id}/transitions", h.transition)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.IdempotencyKey == "" || req.AmountMinor <= 0 || req.Currency != "USD" {
		http.Error(w, "instrucción de transferencia no válida", http.StatusBadRequest)
		return
	}
	result, err := h.store.Create(r.Context(), Instruction{TransferID: req.TransferID, IdempotencyKey: req.IdempotencyKey, SourceParticipantID: req.SourceParticipantID, DestinationParticipantID: req.DestinationParticipantID, DestinationAccountID: req.DestinationAccountID, ClearingAccountID: req.ClearingAccountID, AmountMinor: req.AmountMinor, Currency: req.Currency, CorrelationID: req.CorrelationID})
	if err == ErrIdempotencyConflict {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "no se pudo persistir la instrucción de transferencia", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "identificador de transferencia no válido", http.StatusBadRequest)
		return
	}
	result, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "transferencia no encontrada", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) transition(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "identificador de transferencia no válido", http.StatusBadRequest)
		return
	}
	var req transitionRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.State == "" {
		http.Error(w, "transición no válida", http.StatusBadRequest)
		return
	}
	if err := h.store.Advance(r.Context(), id, req.State, req.CausationID); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
