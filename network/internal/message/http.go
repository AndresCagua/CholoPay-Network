package message

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/inter-network/messages", h.accept)
	mux.HandleFunc("GET /internal/v1/inter-network/messages/{message_id}", h.get)
	mux.HandleFunc("POST /internal/v1/inter-network/messages/{message_id}/claim", h.claim)
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

func (h *Handler) claim(w http.ResponseWriter, r *http.Request) {
	messageID, err := uuid.Parse(r.PathValue("message_id"))
	if err != nil {
		http.Error(w, "message_id inválido", http.StatusBadRequest)
		return
	}
	var request struct {
		CausationID uuid.UUID `json:"causation_id"`
	}
	if json.NewDecoder(r.Body).Decode(&request) != nil || request.CausationID == uuid.Nil {
		http.Error(w, "causation_id inválido", http.StatusBadRequest)
		return
	}
	claimed, err := h.store.ClaimTransferRequest(r.Context(), messageID, request.CausationID)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "mensaje no encontrado", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrBoundary) {
		http.Error(w, "límite de mensaje inválido", http.StatusBadRequest)
		return
	}
	if errors.Is(err, ErrClaimConflict) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "no se pudo reclamar el mensaje", http.StatusServiceUnavailable)
		return
	}
	status := http.StatusAccepted
	if claimed.Replayed {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(claimed)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	messageID, err := uuid.Parse(r.PathValue("message_id"))
	if err != nil {
		http.Error(w, "message_id inválido", http.StatusBadRequest)
		return
	}
	accepted, err := h.store.Get(r.Context(), messageID)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "mensaje no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "el mensaje no está disponible", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(acceptedResponse{Envelope: accepted.Envelope, PayloadHash: accepted.PayloadHash, Status: accepted.Status, CreatedAt: accepted.CreatedAt.UTC().Format(time.RFC3339Nano)})
}

type acceptedResponse struct {
	Envelope
	PayloadHash string `json:"payload_hash"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func (h *Handler) accept(w http.ResponseWriter, r *http.Request) {
	var req acceptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "sobre de mensaje inválido", http.StatusBadRequest)
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
