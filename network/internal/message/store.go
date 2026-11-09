package message

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Envelope struct {
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

type AcceptedMessage struct {
	Envelope    Envelope
	PayloadHash string
	Status      string
	CreatedAt   time.Time
	Replayed    bool
}

var (
	ErrConflict = errors.New("message idempotency key conflicts with a different envelope")
	ErrBoundary = errors.New("message network boundary is invalid")
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS inter_network_message (
		message_id UUID PRIMARY KEY,
		idempotency_key VARCHAR(128) NOT NULL UNIQUE,
		sender_network_id UUID NOT NULL REFERENCES payment_network(network_id),
		destination_network_id UUID NOT NULL REFERENCES payment_network(network_id),
		schema_version VARCHAR(32) NOT NULL,
		message_type VARCHAR(64) NOT NULL,
		correlation_id UUID NOT NULL,
		causation_id UUID NOT NULL,
		payload_hash CHAR(64) NOT NULL,
		payload JSONB NOT NULL,
		status VARCHAR(16) NOT NULL CHECK (status IN ('ACCEPTED')),
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CHECK (sender_network_id <> destination_network_id)
	)`)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS inter_network_message_route ON inter_network_message(sender_network_id,destination_network_id,created_at)`)
	return err
}

func PayloadHash(payload json.RawMessage) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (s *Store) Accept(ctx context.Context, envelope Envelope) (AcceptedMessage, error) {
	if envelope.MessageID == uuid.Nil || envelope.IdempotencyKey == "" || envelope.SenderNetworkID == uuid.Nil || envelope.DestinationNetworkID == uuid.Nil || envelope.SenderNetworkID == envelope.DestinationNetworkID || envelope.SchemaVersion == "" || envelope.MessageType == "" || envelope.CorrelationID == uuid.Nil || envelope.CausationID == uuid.Nil || len(envelope.Payload) == 0 || !json.Valid(envelope.Payload) {
		return AcceptedMessage{}, ErrBoundary
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AcceptedMessage{}, err
	}
	defer tx.Rollback()
	var senderStatus, destinationStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM payment_network WHERE network_id=$1`, envelope.SenderNetworkID).Scan(&senderStatus); err != nil {
		return AcceptedMessage{}, ErrBoundary
	}
	if err := tx.QueryRowContext(ctx, `SELECT status FROM payment_network WHERE network_id=$1`, envelope.DestinationNetworkID).Scan(&destinationStatus); err != nil || senderStatus != "ACTIVE" || destinationStatus != "ACTIVE" {
		return AcceptedMessage{}, ErrBoundary
	}
	hash := PayloadHash(envelope.Payload)
	var existing Envelope
	var existingHash, status string
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT message_id,idempotency_key,sender_network_id,destination_network_id,schema_version,message_type,correlation_id,causation_id,payload_hash,payload,status,created_at FROM inter_network_message WHERE idempotency_key=$1 FOR UPDATE`, envelope.IdempotencyKey).Scan(&existing.MessageID, &existing.IdempotencyKey, &existing.SenderNetworkID, &existing.DestinationNetworkID, &existing.SchemaVersion, &existing.MessageType, &existing.CorrelationID, &existing.CausationID, &existingHash, &existing.Payload, &status, &createdAt)
	if err == nil {
		if existingHash != hash || existing.MessageID != envelope.MessageID || existing.SenderNetworkID != envelope.SenderNetworkID || existing.DestinationNetworkID != envelope.DestinationNetworkID {
			return AcceptedMessage{}, ErrConflict
		}
		return AcceptedMessage{Envelope: existing, PayloadHash: existingHash, Status: status, CreatedAt: createdAt, Replayed: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return AcceptedMessage{}, err
	}
	createdAt = time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO inter_network_message(message_id,idempotency_key,sender_network_id,destination_network_id,schema_version,message_type,correlation_id,causation_id,payload_hash,payload,status,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'ACCEPTED',$11)`, envelope.MessageID, envelope.IdempotencyKey, envelope.SenderNetworkID, envelope.DestinationNetworkID, envelope.SchemaVersion, envelope.MessageType, envelope.CorrelationID, envelope.CausationID, hash, envelope.Payload, createdAt)
	if err != nil {
		return AcceptedMessage{}, err
	}
	if err := tx.Commit(); err != nil {
		return AcceptedMessage{}, err
	}
	return AcceptedMessage{Envelope: envelope, PayloadHash: hash, Status: "ACCEPTED", CreatedAt: createdAt}, nil
}
