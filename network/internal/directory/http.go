package directory

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/directory/resolve", h.resolve)
	mux.HandleFunc("POST /internal/v1/directory/port", h.port)
	mux.HandleFunc("POST /internal/v1/directory/keys", h.create)
	mux.HandleFunc("GET /internal/v1/directory/keys/{keyID}", h.get)
	mux.HandleFunc("POST /internal/v1/directory/keys/{keyID}/suspend", h.suspend)
	mux.HandleFunc("POST /internal/v1/directory/keys/{keyID}/block", h.block)
	mux.HandleFunc("POST /internal/v1/directory/keys/{keyID}/delete", h.delete)
}

type createRequest struct {
	KeyID         uuid.UUID `json:"key_id"`
	NetworkID     uuid.UUID `json:"network_id"`
	ParticipantID uuid.UUID `json:"participant_id"`
	KeyType       string    `json:"key_type"`
	KeyValue      string    `json:"key_value"`
	AccountRef    string    `json:"account_ref"`
	AccountKind   string    `json:"account_kind"`
}

type lifecycleHTTPRequest struct {
	ParticipantID  uuid.UUID `json:"participant_id"`
	AccountRef     string    `json:"account_ref"`
	Reason         string    `json:"reason"`
	IdempotencyKey string    `json:"idempotency_key"`
}

func (h *Handler) lifecycle(w http.ResponseWriter, r *http.Request, operation string) {
	if !authorizedBearer(r, os.Getenv("DIRECTORY_PORT_TOKEN")) {
		http.Error(w, "se requiere autorización", http.StatusUnauthorized)
		return
	}
	keyID, err := uuid.Parse(r.PathValue("keyID"))
	if err != nil {
		http.Error(w, "clave no válida", http.StatusBadRequest)
		return
	}
	var request lifecycleHTTPRequest
	if json.NewDecoder(r.Body).Decode(&request) != nil || request.ParticipantID == uuid.Nil || request.IdempotencyKey == "" {
		http.Error(w, "solicitud no válida", http.StatusBadRequest)
		return
	}
	life := LifecycleRequest{KeyID: keyID, ParticipantID: request.ParticipantID, AccountRef: request.AccountRef, Reason: request.Reason, IdempotencyKey: request.IdempotencyKey}
	switch operation {
	case "SUSPEND":
		err = h.store.Suspend(r.Context(), life)
	case "BLOCK":
		err = h.store.Block(r.Context(), life)
	case "DELETE":
		err = h.store.Delete(r.Context(), life)
	}
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, ErrUnauthorized) {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
func (h *Handler) suspend(w http.ResponseWriter, r *http.Request) { h.lifecycle(w, r, "SUSPEND") }
func (h *Handler) block(w http.ResponseWriter, r *http.Request)   { h.lifecycle(w, r, "BLOCK") }
func (h *Handler) delete(w http.ResponseWriter, r *http.Request)  { h.lifecycle(w, r, "DELETE") }

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	if !authorizedBearer(r, os.Getenv("DIRECTORY_PORT_TOKEN")) {
		http.Error(w, "se requiere autorización", http.StatusUnauthorized)
		return
	}
	var request createRequest
	if json.NewDecoder(r.Body).Decode(&request) != nil || request.NetworkID == uuid.Nil || request.ParticipantID == uuid.Nil || request.KeyType == "" || request.KeyValue == "" || request.AccountRef == "" {
		http.Error(w, "solicitud no válida", http.StatusBadRequest)
		return
	}
	entry, err := h.store.Create(r.Context(), Entry{KeyID: request.KeyID, NetworkID: request.NetworkID, ParticipantID: request.ParticipantID, KeyType: request.KeyType, KeyValue: request.KeyValue, AccountRef: request.AccountRef, AccountKind: request.AccountKind, Status: "ACTIVE", Version: 1, IdempotencyKey: r.Header.Get("Idempotency-Key")})
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, ErrKeyLimit) {
			status = http.StatusUnprocessableEntity
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if !authorizedBearer(r, os.Getenv("DIRECTORY_PORT_TOKEN")) {
		http.Error(w, "se requiere autorización", http.StatusUnauthorized)
		return
	}
	keyID, err := uuid.Parse(r.PathValue("keyID"))
	if err != nil {
		http.Error(w, "clave no válida", 400)
		return
	}
	entry, err := h.store.Get(r.Context(), keyID)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entry)
}

type resolveRequest struct {
	NetworkID      uuid.UUID `json:"network_id"`
	KeyType        string    `json:"key_type"`
	KeyValue       string    `json:"key_value"`
	CorrelationID  uuid.UUID `json:"correlation_id"`
	CausationID    uuid.UUID `json:"causation_id"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type portRequest struct {
	SchemaVersion     string    `json:"schema_version"`
	CorrelationID     uuid.UUID `json:"correlation_id"`
	CausationID       uuid.UUID `json:"causation_id"`
	TraceID           string    `json:"trace_id"`
	RequestedAt       time.Time `json:"requested_at"`
	IdempotencyKey    string    `json:"idempotency_key"`
	KeyType           string    `json:"key_type"`
	KeyValue          string    `json:"key_value"`
	AccountRef        string    `json:"account_ref"`
	AccountKind       string    `json:"account_kind"`
	FromNetworkID     uuid.UUID `json:"from_network_id"`
	FromParticipantID uuid.UUID `json:"from_participant_id"`
	ToNetworkID       uuid.UUID `json:"to_network_id"`
	ToParticipantID   uuid.UUID `json:"to_participant_id"`
	NewKeyID          uuid.UUID `json:"new_key_id"`
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

func (h *Handler) port(w http.ResponseWriter, r *http.Request) {
	if !authorizedBearer(r, os.Getenv("DIRECTORY_PORT_TOKEN")) {
		http.Error(w, "se requiere autorización", http.StatusUnauthorized)
		return
	}
	var request portRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "solicitud no válida", http.StatusBadRequest)
		return
	}
	if request.SchemaVersion == "" || request.CorrelationID == uuid.Nil || request.CausationID == uuid.Nil || request.IdempotencyKey == "" || request.NewKeyID == uuid.Nil || request.RequestedAt.IsZero() {
		http.Error(w, "se requieren schema_version, correlation_id, causation_id, requested_at, idempotency_key y new_key_id", http.StatusBadRequest)
		return
	}
	err := h.store.Port(r.Context(), PortRequest{SchemaVersion: request.SchemaVersion, CorrelationID: request.CorrelationID, CausationID: request.CausationID, TraceID: request.TraceID, RequestedAt: request.RequestedAt, IdempotencyKey: request.IdempotencyKey, KeyType: request.KeyType, KeyValue: request.KeyValue, AccountRef: request.AccountRef, AccountKind: request.AccountKind, FromNetworkID: request.FromNetworkID, FromParticipantID: request.FromParticipantID, ToNetworkID: request.ToNetworkID, ToParticipantID: request.ToParticipantID, NewKeyID: request.NewKeyID})
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func authorizedBearer(r *http.Request, expected string) bool {
	if expected == "" {
		return false
	}
	value := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	return len(value) == len(expected) && subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1
}

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	var request resolveRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "solicitud no válida", http.StatusBadRequest)
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
