package orchestration

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cholopay/network/internal/instruction"
	"github.com/cholopay/network/internal/message"
	"github.com/cholopay/network/internal/routing"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/inter-network/transfers", h.intake)
}

type intakeRequest struct {
	TransferID               uuid.UUID `json:"transfer_id"`
	IdempotencyKey           string    `json:"idempotency_key"`
	SourceParticipantID      uuid.UUID `json:"source_participant_id"`
	DestinationParticipantID uuid.UUID `json:"destination_participant_id"`
	AmountMinor              int64     `json:"amount_minor"`
	Currency                 string    `json:"currency"`
	CorrelationID            uuid.UUID `json:"correlation_id"`
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
	var request intakeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.IdempotencyKey == "" || request.AmountMinor <= 0 || request.Currency != "USD" || request.SourceParticipantID == uuid.Nil || request.DestinationParticipantID == uuid.Nil || request.CorrelationID == uuid.Nil {
		http.Error(w, "transferencia entre redes inválida", http.StatusBadRequest)
		return
	}
	result, err := h.service.Intake(r.Context(), Intake{TransferID: request.TransferID, IdempotencyKey: request.IdempotencyKey, SourceParticipantID: request.SourceParticipantID, DestinationParticipantID: request.DestinationParticipantID, AmountMinor: request.AmountMinor, Currency: request.Currency, CorrelationID: request.CorrelationID})
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, routing.ErrParticipantUnavailable) {
			status = http.StatusConflict
		}
		if errors.Is(err, instruction.ErrIdempotencyConflict) || errors.Is(err, message.ErrConflict) {
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
