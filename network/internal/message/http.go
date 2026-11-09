package message

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/inter-network/messages", h.accept)
}

type acceptRequest struct {
	MessageID            uuid.UUID       `json:"message_id"`
	IdempotencyKey       string          `json:"idempotency_key"`
	SenderNetworkID      uuid.UUID       `json:"sender_network_id"`
	DestinationNetworkID uuid.UUID       `json:"destination_network_id"`
	SchemaVersion        string          `json:"schema_version"`
	MessageType          string          `json:"message_type"`
	CorrelationID        uuid.UUID       `json:"correlation_id"`
	CausationID          uuid.UUID       `json:"causation_id"`
	Payload              json.RawMessage `json:"payload"`
}

type acceptResponse struct {
	MessageID      uuid.UUID `json:"message_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Status         string    `json:"status"`
	PayloadHash    string    `json:"payload_hash"`
	Replayed       bool      `json:"replayed"`
	CreatedAt      string    `json:"created_at"`
}

func (h *Handler) accept(w http.ResponseWriter, r *http.Request) {
	var req acceptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid message envelope", http.StatusBadRequest)
		return
	}
	accepted, err := h.store.Accept(r.Context(), Envelope{MessageID: req.MessageID, IdempotencyKey: req.IdempotencyKey, SenderNetworkID: req.SenderNetworkID, DestinationNetworkID: req.DestinationNetworkID, SchemaVersion: req.SchemaVersion, MessageType: req.MessageType, CorrelationID: req.CorrelationID, CausationID: req.CausationID, Payload: req.Payload})
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, ErrBoundary) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, ErrConflict) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	status := http.StatusAccepted
	if accepted.Replayed {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(acceptResponse{MessageID: accepted.Envelope.MessageID, IdempotencyKey: accepted.Envelope.IdempotencyKey, Status: accepted.Status, PayloadHash: accepted.PayloadHash, Replayed: accepted.Replayed, CreatedAt: accepted.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00")})
}
