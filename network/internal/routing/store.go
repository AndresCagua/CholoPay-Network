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
type ParticipantDetail struct {
	Participant
	NetworkCode   string
	NetworkKind   string
	NetworkStatus string
}
type NetworkDetail struct {
	Network
	Participants []Participant
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

var (
	ErrParticipantUnavailable = errors.New("el participante o la red no están disponibles")
	ErrInvalidNetwork         = errors.New("red inválida")
	ErrInvalidParticipant     = errors.New("participante inválido")
)

const (
	Active    = "ACTIVE"
	Suspended = "SUSPENDED"
)

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
	if network.ID == uuid.Nil || network.Code == "" || (network.Kind != "BANKING" && network.Kind != "FINTECH") {
		return ErrInvalidNetwork
	}
	if network.Status == "" {
		network.Status = Active
	}
	if network.Status != Active && network.Status != Suspended {
		return ErrInvalidNetwork
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO payment_network(network_id,code,kind,status) VALUES ($1,$2,$3,$4)`, network.ID, network.Code, network.Kind, network.Status)
	return err
}
func (s *Store) RegisterParticipant(ctx context.Context, participant Participant) error {
	if participant.ID == uuid.Nil || participant.NetworkID == uuid.Nil || participant.Code == "" {
		return ErrInvalidParticipant
	}
	if participant.Status == "" {
		participant.Status = Active
	}
	if participant.Status != Active && participant.Status != Suspended {
		return ErrInvalidParticipant
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO network_participant(participant_id,network_id,code,status) VALUES ($1,$2,$3,$4)`, participant.ID, participant.NetworkID, participant.Code, participant.Status)
	return err
}
func (s *Store) SetNetworkStatus(ctx context.Context, id uuid.UUID, status string) error {
	if id == uuid.Nil || (status != Active && status != Suspended) {
		return ErrInvalidNetwork
	}
	result, err := s.db.ExecContext(ctx, `UPDATE payment_network SET status=$2 WHERE network_id=$1`, id, status)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (s *Store) ListNetworks(ctx context.Context) ([]Network, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT network_id, code, kind, status FROM payment_network ORDER BY code, network_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	networks := []Network{}
	for rows.Next() {
		var network Network
		if err := rows.Scan(&network.ID, &network.Code, &network.Kind, &network.Status); err != nil {
			return nil, err
		}
		networks = append(networks, network)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return networks, nil
}

func (s *Store) GetNetwork(ctx context.Context, id uuid.UUID) (NetworkDetail, error) {
	var detail NetworkDetail
	if err := s.db.QueryRowContext(ctx, `SELECT network_id, code, kind, status FROM payment_network WHERE network_id=$1`, id).Scan(
		&detail.ID, &detail.Code, &detail.Kind, &detail.Status,
	); err != nil {
		return NetworkDetail{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT participant_id, network_id, code, status
		FROM network_participant WHERE network_id=$1 ORDER BY code, participant_id`, id)
	if err != nil {
		return NetworkDetail{}, err
	}
	defer rows.Close()
	detail.Participants = []Participant{}
	for rows.Next() {
		var participant Participant
		if err := rows.Scan(&participant.ID, &participant.NetworkID, &participant.Code, &participant.Status); err != nil {
			return NetworkDetail{}, err
		}
		detail.Participants = append(detail.Participants, participant)
	}
	if err := rows.Err(); err != nil {
		return NetworkDetail{}, err
	}
	return detail, nil
}

func (s *Store) ListParticipants(ctx context.Context) ([]ParticipantDetail, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.participant_id, p.network_id, p.code, p.status,
		       n.code, n.kind, n.status
		FROM network_participant p
		JOIN payment_network n ON n.network_id = p.network_id
		ORDER BY p.code, p.participant_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	participants := []ParticipantDetail{}
	for rows.Next() {
		var detail ParticipantDetail
		if err := rows.Scan(
			&detail.ID, &detail.NetworkID, &detail.Code, &detail.Status,
			&detail.NetworkCode, &detail.NetworkKind, &detail.NetworkStatus,
		); err != nil {
			return nil, err
		}
		participants = append(participants, detail)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return participants, nil
}

func (s *Store) GetParticipant(ctx context.Context, id uuid.UUID) (ParticipantDetail, error) {
	var detail ParticipantDetail
	err := s.db.QueryRowContext(ctx, `
		SELECT p.participant_id, p.network_id, p.code, p.status,
		       n.code, n.kind, n.status
		FROM network_participant p
		JOIN payment_network n ON n.network_id = p.network_id
		WHERE p.participant_id=$1`, id).Scan(
		&detail.ID, &detail.NetworkID, &detail.Code, &detail.Status,
		&detail.NetworkCode, &detail.NetworkKind, &detail.NetworkStatus,
	)
	return detail, err
}

func (s *Store) SetParticipantStatus(ctx context.Context, id uuid.UUID, status string) error {
	if id == uuid.Nil || (status != Active && status != Suspended) {
		return ErrInvalidParticipant
	}
	result, err := s.db.ExecContext(ctx, `UPDATE network_participant SET status=$2 WHERE participant_id=$1`, id, status)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
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
