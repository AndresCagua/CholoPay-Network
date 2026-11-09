package orchestration

import (
	"context"
	"encoding/json"

	"github.com/cholopay/network/internal/instruction"
	"github.com/cholopay/network/internal/message"
	"github.com/cholopay/network/internal/routing"
	"github.com/google/uuid"
)

type Intake struct {
	TransferID               uuid.UUID
	IdempotencyKey           string
	SourceParticipantID      uuid.UUID
	DestinationParticipantID uuid.UUID
	AmountMinor              int64
	Currency                 string
	CorrelationID            uuid.UUID
}

type Result struct {
	Instruction instruction.Instruction
	Route       routing.Route
	Message     *message.AcceptedMessage
}

type Service struct {
	instructions *instruction.Store
	routes       *routing.Store
	messages     *message.Store
}

func NewService(instructions *instruction.Store, routes *routing.Store, messages *message.Store) *Service {
	return &Service{instructions: instructions, routes: routes, messages: messages}
}

func (s *Service) Intake(ctx context.Context, request Intake) (Result, error) {
	route, err := s.routes.Route(ctx, request.SourceParticipantID, request.DestinationParticipantID)
	if err != nil {
		return Result{}, err
	}
	created, err := s.instructions.Create(ctx, instruction.Instruction{TransferID: request.TransferID, IdempotencyKey: request.IdempotencyKey, SourceParticipantID: request.SourceParticipantID, DestinationParticipantID: &request.DestinationParticipantID, AmountMinor: request.AmountMinor, Currency: request.Currency, CorrelationID: request.CorrelationID})
	if err != nil {
		return Result{}, err
	}
	result := Result{Instruction: created, Route: route}
	if route.Kind == routing.IntraNetwork {
		return result, nil
	}
	payload, err := json.Marshal(map[string]any{
		"transfer_id":                created.TransferID,
		"source_participant_id":      request.SourceParticipantID,
		"destination_participant_id": request.DestinationParticipantID,
		"amount_minor":               request.AmountMinor,
		"currency":                   request.Currency,
		"route_kind":                 route.Kind,
		"source_network_id":          route.SourceNetwork.ID,
		"destination_network_id":     route.DestinationNetwork.ID,
	})
	if err != nil {
		return Result{}, err
	}
	accepted, err := s.messages.Accept(ctx, message.Envelope{MessageID: created.TransferID, IdempotencyKey: "transfer:" + created.IdempotencyKey, SenderNetworkID: route.SourceNetwork.ID, DestinationNetworkID: route.DestinationNetwork.ID, SchemaVersion: "1.0", MessageType: "TRANSFER_REQUESTED", CorrelationID: created.CorrelationID, CausationID: created.TransferID, Payload: payload})
	if err != nil {
		return Result{}, err
	}
	result.Message = &accepted
	return result, nil
}
