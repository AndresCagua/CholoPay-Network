package directory

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/directory/resolve", h.resolve)
}

type resolveRequest struct {
	NetworkID      uuid.UUID `json:"network_id"`
	KeyType        string    `json:"key_type"`
	KeyValue       string    `json:"key_value"`
	CorrelationID  uuid.UUID `json:"correlation_id"`
	CausationID    uuid.UUID `json:"causation_id"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type resolveResponse struct {
	KeyID          uuid.UUID `json:"key_id"`
	ParticipantID  uuid.UUID `json:"participant_id"`
	AccountRef     string    `json:"account_ref"`
	SchemaVersion  string    `json:"schema_version"`
	MessageType    string    `json:"message_type"`
	CorrelationID  uuid.UUID `json:"correlation_id"`
	CausationID    uuid.UUID `json:"causation_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	ResolvedAt     string    `json:"resolved_at"`
}

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	var request resolveRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	resolution, err := h.store.Resolve(r.Context(), request.NetworkID, request.KeyType, request.KeyValue, request.CorrelationID, request.CausationID, request.IdempotencyKey)
	if err != nil {
		status := http.StatusNotFound
		if errors.Is(err, ErrSuspended) || errors.Is(err, ErrAmbiguous) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resolveResponse{KeyID: resolution.Entry.KeyID, ParticipantID: resolution.Entry.ParticipantID, AccountRef: resolution.Entry.AccountRef, SchemaVersion: resolution.SchemaVersion, MessageType: resolution.MessageType, CorrelationID: resolution.CorrelationID, CausationID: resolution.CausationID, IdempotencyKey: resolution.IdempotencyKey, ResolvedAt: resolution.ResolvedAt.Format("2006-01-02T15:04:05.999999Z07:00")})
}
