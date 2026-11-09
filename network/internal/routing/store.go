package routing

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
)

type Network struct {
	ID     uuid.UUID
	Code   string
	Kind   string
	Status string
}
type Participant struct {
	ID        uuid.UUID
	NetworkID uuid.UUID
	Code      string
	Status    string
}
type RouteKind string

const (
	IntraNetwork RouteKind = "INTRA_NETWORK"
	InterNetwork RouteKind = "INTER_NETWORK"
)

type Route struct {
	Kind                   RouteKind
	SourceNetwork          Network
	DestinationNetwork     Network
	SourceParticipant      Participant
	DestinationParticipant Participant
}

var ErrParticipantUnavailable = errors.New("participant or network is unavailable")

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }
func (s *Store) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS payment_network (network_id UUID PRIMARY KEY, code VARCHAR(64) NOT NULL UNIQUE, kind VARCHAR(16) NOT NULL CHECK (kind IN ('BANKING','FINTECH')), status VARCHAR(16) NOT NULL CHECK (status IN ('ACTIVE','SUSPENDED')), created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS network_participant (participant_id UUID PRIMARY KEY, network_id UUID NOT NULL REFERENCES payment_network(network_id), code VARCHAR(64) NOT NULL UNIQUE, status VARCHAR(16) NOT NULL CHECK (status IN ('ACTIVE','SUSPENDED')), created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS network_participant_network ON network_participant(network_id,status)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) RegisterNetwork(ctx context.Context, network Network) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO payment_network(network_id,code,kind,status) VALUES ($1,$2,$3,$4)`, network.ID, network.Code, network.Kind, network.Status)
	return err
}
func (s *Store) RegisterParticipant(ctx context.Context, participant Participant) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO network_participant(participant_id,network_id,code,status) VALUES ($1,$2,$3,$4)`, participant.ID, participant.NetworkID, participant.Code, participant.Status)
	return err
}
func (s *Store) Route(ctx context.Context, sourceID, destinationID uuid.UUID) (Route, error) {
	var route Route
	var sourceNetwork, destinationNetwork Network
	var source, destination Participant
	if err := s.db.QueryRowContext(ctx, `SELECT p.participant_id,p.network_id,p.code,p.status,n.network_id,n.code,n.kind,n.status FROM network_participant p JOIN payment_network n ON n.network_id=p.network_id WHERE p.participant_id=$1`, sourceID).Scan(&source.ID, &source.NetworkID, &source.Code, &source.Status, &sourceNetwork.ID, &sourceNetwork.Code, &sourceNetwork.Kind, &sourceNetwork.Status); err != nil {
		return Route{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT p.participant_id,p.network_id,p.code,p.status,n.network_id,n.code,n.kind,n.status FROM network_participant p JOIN payment_network n ON n.network_id=p.network_id WHERE p.participant_id=$1`, destinationID).Scan(&destination.ID, &destination.NetworkID, &destination.Code, &destination.Status, &destinationNetwork.ID, &destinationNetwork.Code, &destinationNetwork.Kind, &destinationNetwork.Status); err != nil {
		return Route{}, err
	}
	if source.Status != "ACTIVE" || destination.Status != "ACTIVE" || sourceNetwork.Status != "ACTIVE" || destinationNetwork.Status != "ACTIVE" {
		return Route{}, ErrParticipantUnavailable
	}
	route.Kind, route.SourceNetwork, route.DestinationNetwork, route.SourceParticipant, route.DestinationParticipant = IntraNetwork, sourceNetwork, destinationNetwork, source, destination
	if sourceNetwork.ID != destinationNetwork.ID {
		route.Kind = InterNetwork
	}
	return route, nil
}
