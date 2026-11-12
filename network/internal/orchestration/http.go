package orchestration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/cholopay/network/internal/instruction"
	"github.com/cholopay/network/internal/message"
	"github.com/cholopay/network/internal/routing"
	"github.com/google/uuid"
)

type Handler struct {
	service    *Service
	hmacSecret string
}

// NewHandler keeps construction small for callers; an empty secret deliberately rejects every request.
func NewHandler(service *Service, secret ...string) *Handler {
	configured := ""
	if len(secret) > 0 {
		configured = secret[0]
	}
	return &Handler{service: service, hmacSecret: configured}
}
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/inter-network/transfers", h.intake)
}

type intakeRequest struct {
	TransferID                    uuid.UUID `json:"transfer_id"`
	TransferIDCamel               uuid.UUID `json:"transferId"`
	IdempotencyKey                string    `json:"idempotency_key"`
	SourceParticipantID           uuid.UUID `json:"source_participant_id"`
	SourceParticipantIDCamel      uuid.UUID `json:"sourceParticipantId"`
	DestinationParticipantID      uuid.UUID `json:"destination_participant_id"`
	DestinationParticipantIDCamel uuid.UUID `json:"destinationParticipantId"`
	AmountMinor                   int64     `json:"amount_minor"`
	Currency                      string    `json:"currency"`
	CorrelationID                 uuid.UUID `json:"correlation_id"`
}

func (r *intakeRequest) normalize() {
	if r.TransferID == uuid.Nil {
		r.TransferID = r.TransferIDCamel
	}
	if r.SourceParticipantID == uuid.Nil {
		r.SourceParticipantID = r.SourceParticipantIDCamel
	}
	if r.DestinationParticipantID == uuid.Nil {
		r.DestinationParticipantID = r.DestinationParticipantIDCamel
	}
}

type intakeEnvelope struct {
	Payload json.RawMessage `json:"payload"`
}
type intakeResponse struct {
	TransferID           uuid.UUID         `json:"transfer_id"`
	State                string            `json:"state"`
	RouteKind            routing.RouteKind `json:"route_kind"`
	SourceNetworkID      uuid.UUID         `json:"source_network_id"`
	DestinationNetworkID uuid.UUID         `json:"destination_network_id"`
	MessageAccepted      bool              `json:"message_accepted"`
	MessageReplayed      bool              `json:"message_replayed"`
	MessageID            *uuid.UUID        `json:"message_id,omitempty"`
}

func (h *Handler) intake(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil || !h.validSignature(r, body) {
		http.Error(w, "solicitud HMAC inválida", http.StatusUnauthorized)
		return
	}
	var request intakeRequest
	var envelope intakeEnvelope
	if json.Unmarshal(body, &envelope) == nil && len(envelope.Payload) > 0 {
		if err := json.Unmarshal(envelope.Payload, &request); err != nil {
			http.Error(w, "transferencia entre redes inválida", http.StatusBadRequest)
			return
		}
	} else if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "transferencia entre redes inválida", http.StatusBadRequest)
		return
	}
	request.normalize()
	if err := validate(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := h.service.Intake(r.Context(), Intake{TransferID: request.TransferID, IdempotencyKey: request.IdempotencyKey, SourceParticipantID: request.SourceParticipantID, DestinationParticipantID: request.DestinationParticipantID, AmountMinor: request.AmountMinor, Currency: request.Currency, CorrelationID: request.CorrelationID})
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, routing.ErrParticipantUnavailable) || errors.Is(err, instruction.ErrIdempotencyConflict) || errors.Is(err, message.ErrConflict) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	response := intakeResponse{TransferID: result.Instruction.TransferID, State: string(result.Instruction.State), RouteKind: result.Route.Kind, SourceNetworkID: result.Route.SourceNetwork.ID, DestinationNetworkID: result.Route.DestinationNetwork.ID}
	if result.Message != nil {
		response.MessageAccepted = true
		response.MessageReplayed = result.Message.Replayed
		response.MessageID = &result.Message.Envelope.MessageID
	}
	status := http.StatusAccepted
	if result.Message != nil && result.Message.Replayed {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func validate(request intakeRequest) error {
	if request.IdempotencyKey == "" || request.AmountMinor <= 0 || request.Currency != "USD" || request.SourceParticipantID == uuid.Nil || request.DestinationParticipantID == uuid.Nil || request.CorrelationID == uuid.Nil {
		return errors.New("transferencia entre redes inválida")
	}
	return nil
}
func (h *Handler) validSignature(r *http.Request, body []byte) bool {
	if h.hmacSecret == "" {
		return false
	}
	ts, err := strconv.ParseInt(r.Header.Get("X-CholoPay-Timestamp"), 10, 64)
	if err != nil || time.Since(time.Unix(ts, 0)) > 5*time.Minute || time.Until(time.Unix(ts, 0)) > 5*time.Minute {
		return false
	}
	requestID := r.Header.Get("X-CholoPay-Request-Id")
	provided, err := hex.DecodeString(r.Header.Get("X-CholoPay-Signature"))
	if err != nil || requestID == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(h.hmacSecret))
	_, _ = mac.Write([]byte(strconv.FormatInt(ts, 10) + "." + requestID + "." + string(body)))
	return hmac.Equal(provided, mac.Sum(nil))
}
