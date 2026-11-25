package instruction

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/cholopay/network/internal/transfer"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	store      *Store
	hmacSecret string
}

func NewHandler(store *Store, secret ...string) *Handler {
	configured := ""
	if len(secret) > 0 {
		configured = secret[0]
	}
	return &Handler{store: store, hmacSecret: configured}
}

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
	mux.HandleFunc("POST /internal/v1/transfers/{id}/funds-reserved", h.fundsReserved)
	mux.HandleFunc("POST /internal/v1/transfers/{id}/ready-for-settlement", h.readyForSettlement)
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

func (h *Handler) fundsReserved(w http.ResponseWriter, r *http.Request) {
	body, readErr := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if readErr != nil || !h.validSignature(r, body) {
		http.Error(w, "solicitud HMAC inválida", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "identificador de transferencia no válido", http.StatusBadRequest)
		return
	}
	var req struct {
		CausationID uuid.UUID `json:"causation_id"`
	}
	if json.Unmarshal(body, &req) != nil || req.CausationID == uuid.Nil {
		http.Error(w, "causation_id inválido", http.StatusBadRequest)
		return
	}
	if err := h.store.MarkFundsReserved(r.Context(), id, req.CausationID); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) readyForSettlement(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "identificador de transferencia no válido", http.StatusBadRequest)
		return
	}
	var req struct {
		CausationID uuid.UUID `json:"causation_id"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.CausationID == uuid.Nil {
		http.Error(w, "causation_id inválido", http.StatusBadRequest)
		return
	}
	if err := h.store.MarkReadyForSettlement(r.Context(), id, req.CausationID); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) validSignature(r *http.Request, body []byte) bool {
	if h.hmacSecret == "" {
		return false
	}
	ts, err := strconv.ParseInt(r.Header.Get("X-CholoPay-Timestamp"), 10, 64)
	if err != nil || time.Since(time.Unix(ts, 0)) > 5*time.Minute || time.Until(time.Unix(ts, 0)) > 5*time.Minute {
		return false
	}
	requestID, err := uuid.Parse(r.Header.Get("X-CholoPay-Request-Id"))
	if err != nil {
		return false
	}
	provided, err := hex.DecodeString(r.Header.Get("X-CholoPay-Signature"))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(h.hmacSecret))
	_, _ = mac.Write([]byte(strconv.FormatInt(ts, 10) + "." + requestID.String() + "." + string(body)))
	return hmac.Equal(provided, mac.Sum(nil))
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
