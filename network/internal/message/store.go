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

type ClaimedMessage struct {
	MessageID        uuid.UUID `json:"message_id"`
	TransferID       uuid.UUID `json:"transfer_id"`
	MessageStatus    string    `json:"message_status"`
	InstructionState string    `json:"instruction_state"`
	Replayed         bool      `json:"replayed"`
}

var (
	ErrConflict      = errors.New("la clave de idempotencia del mensaje entra en conflicto con otro sobre")
	ErrBoundary      = errors.New("el límite de red del mensaje no es válido")
	ErrNotFound      = errors.New("mensaje no encontrado")
	ErrClaimConflict = errors.New("el mensaje no puede reclamarse en su estado actual")
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
		status VARCHAR(16) NOT NULL CHECK (status IN ('ACCEPTED','PROCESSED')),
		processed_at TIMESTAMPTZ,
		processed_causation_id UUID,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CHECK (sender_network_id <> destination_network_id)
	)`)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS inter_network_message_route ON inter_network_message(sender_network_id,destination_network_id,created_at)`)
	if err != nil {
		return err
	}
	for _, q := range []string{
		`ALTER TABLE inter_network_message ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ`,
		`ALTER TABLE inter_network_message ADD COLUMN IF NOT EXISTS processed_causation_id UUID`,
		`ALTER TABLE inter_network_message DROP CONSTRAINT IF EXISTS inter_network_message_status_check`,
		`ALTER TABLE inter_network_message ADD CONSTRAINT inter_network_message_status_check CHECK (status IN ('ACCEPTED','PROCESSED'))`,
	} {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func PayloadHash(payload json.RawMessage) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (s *Store) ClaimTransferRequest(ctx context.Context, messageID, causationID uuid.UUID) (ClaimedMessage, error) {
	if messageID == uuid.Nil || causationID == uuid.Nil {
		return ClaimedMessage{}, ErrBoundary
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ClaimedMessage{}, err
	}
	defer tx.Rollback()
	var envelope Envelope
	var payloadHash, status string
	var processedCausation *uuid.UUID
	if err := tx.QueryRowContext(ctx, `SELECT message_id,idempotency_key,sender_network_id,destination_network_id,schema_version,message_type,correlation_id,causation_id,payload_hash,payload,status,processed_causation_id FROM inter_network_message WHERE message_id=$1 FOR UPDATE`, messageID).Scan(&envelope.MessageID, &envelope.IdempotencyKey, &envelope.SenderNetworkID, &envelope.DestinationNetworkID, &envelope.SchemaVersion, &envelope.MessageType, &envelope.CorrelationID, &envelope.CausationID, &payloadHash, &envelope.Payload, &status, &processedCausation); errors.Is(err, sql.ErrNoRows) {
		return ClaimedMessage{}, ErrNotFound
	} else if err != nil {
		return ClaimedMessage{}, err
	}
	var payload struct {
		TransferID uuid.UUID `json:"transfer_id"`
	}
	if envelope.MessageType != "TRANSFER_REQUESTED" || json.Unmarshal(envelope.Payload, &payload) != nil || payload.TransferID == uuid.Nil {
		return ClaimedMessage{}, ErrClaimConflict
	}
	if status == "PROCESSED" {
		if processedCausation == nil || *processedCausation != causationID {
			return ClaimedMessage{}, ErrClaimConflict
		}
		return ClaimedMessage{MessageID: messageID, TransferID: payload.TransferID, MessageStatus: status, InstructionState: "VALIDATING", Replayed: true}, nil
	}
	if status != "ACCEPTED" {
		return ClaimedMessage{}, ErrClaimConflict
	}
	var instructionState string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM transfer_instruction WHERE transfer_id=$1 FOR UPDATE`, payload.TransferID).Scan(&instructionState); err != nil {
		return ClaimedMessage{}, ErrClaimConflict
	}
	if instructionState != "CREATED" {
		return ClaimedMessage{}, ErrClaimConflict
	}
	if _, err := tx.ExecContext(ctx, `UPDATE transfer_instruction SET state='VALIDATING', causation_id=$1, updated_at=CURRENT_TIMESTAMP WHERE transfer_id=$2`, causationID, payload.TransferID); err != nil {
		return ClaimedMessage{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE inter_network_message SET status='PROCESSED', processed_at=CURRENT_TIMESTAMP, processed_causation_id=$1 WHERE message_id=$2`, causationID, messageID); err != nil {
		return ClaimedMessage{}, err
	}
	if err := tx.Commit(); err != nil {
		return ClaimedMessage{}, err
	}
	return ClaimedMessage{MessageID: messageID, TransferID: payload.TransferID, MessageStatus: "PROCESSED", InstructionState: "VALIDATING"}, nil
}

func (s *Store) Get(ctx context.Context, messageID uuid.UUID) (AcceptedMessage, error) {
	if messageID == uuid.Nil {
		return AcceptedMessage{}, ErrNotFound
	}
	var envelope Envelope
	var accepted AcceptedMessage
	err := s.db.QueryRowContext(ctx, `SELECT message_id,idempotency_key,sender_network_id,destination_network_id,schema_version,message_type,correlation_id,causation_id,payload_hash,payload,status,created_at FROM inter_network_message WHERE message_id=$1`, messageID).Scan(
		&envelope.MessageID, &envelope.IdempotencyKey, &envelope.SenderNetworkID, &envelope.DestinationNetworkID,
		&envelope.SchemaVersion, &envelope.MessageType, &envelope.CorrelationID, &envelope.CausationID,
		&accepted.PayloadHash, &envelope.Payload, &accepted.Status, &accepted.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AcceptedMessage{}, ErrNotFound
	}
	if err != nil {
		return AcceptedMessage{}, err
	}
	accepted.Envelope = envelope
	return accepted, nil
}

func (s *Store) Accept(ctx context.Context, envelope Envelope) (AcceptedMessage, error) {
	if err := validateEnvelope(envelope); err != nil {
		return AcceptedMessage{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AcceptedMessage{}, err
	}
	defer tx.Rollback()
	accepted, err := s.acceptTx(ctx, tx, envelope)
	if err != nil {
		return AcceptedMessage{}, err
	}
	if err := tx.Commit(); err != nil {
		return AcceptedMessage{}, err
	}
	return accepted, nil
}

func validateEnvelope(envelope Envelope) error {
	if envelope.MessageID == uuid.Nil || envelope.IdempotencyKey == "" || envelope.SenderNetworkID == uuid.Nil || envelope.DestinationNetworkID == uuid.Nil || envelope.SenderNetworkID == envelope.DestinationNetworkID || envelope.SchemaVersion == "" || envelope.MessageType == "" || envelope.CorrelationID == uuid.Nil || envelope.CausationID == uuid.Nil || len(envelope.Payload) == 0 || !json.Valid(envelope.Payload) {
		return ErrBoundary
	}
	return nil
}

func (s *Store) AcceptTx(ctx context.Context, tx *sql.Tx, envelope Envelope) (AcceptedMessage, error) {
	if err := validateEnvelope(envelope); err != nil {
		return AcceptedMessage{}, err
	}
	return s.acceptTx(ctx, tx, envelope)
}

func (s *Store) acceptTx(ctx context.Context, tx *sql.Tx, envelope Envelope) (AcceptedMessage, error) {
	hash := PayloadHash(envelope.Payload)
	var existing Envelope
	var existingHash, status string
	var createdAt time.Time
	err := tx.QueryRowContext(ctx, `SELECT message_id,idempotency_key,sender_network_id,destination_network_id,schema_version,message_type,correlation_id,causation_id,payload_hash,payload,status,created_at FROM inter_network_message WHERE idempotency_key=$1 FOR UPDATE`, envelope.IdempotencyKey).Scan(&existing.MessageID, &existing.IdempotencyKey, &existing.SenderNetworkID, &existing.DestinationNetworkID, &existing.SchemaVersion, &existing.MessageType, &existing.CorrelationID, &existing.CausationID, &existingHash, &existing.Payload, &status, &createdAt)
	if err == nil {
		if existingHash != hash || existing.MessageID != envelope.MessageID || existing.SenderNetworkID != envelope.SenderNetworkID || existing.DestinationNetworkID != envelope.DestinationNetworkID || existing.SchemaVersion != envelope.SchemaVersion || existing.MessageType != envelope.MessageType || existing.CorrelationID != envelope.CorrelationID || existing.CausationID != envelope.CausationID {
			return AcceptedMessage{}, ErrConflict
		}
		return AcceptedMessage{Envelope: existing, PayloadHash: existingHash, Status: status, CreatedAt: createdAt, Replayed: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return AcceptedMessage{}, err
	}
	var senderStatus, destinationStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM payment_network WHERE network_id=$1`, envelope.SenderNetworkID).Scan(&senderStatus); err != nil {
		return AcceptedMessage{}, ErrBoundary
	}
	if err := tx.QueryRowContext(ctx, `SELECT status FROM payment_network WHERE network_id=$1`, envelope.DestinationNetworkID).Scan(&destinationStatus); err != nil || senderStatus != "ACTIVE" || destinationStatus != "ACTIVE" {
		return AcceptedMessage{}, ErrBoundary
	}
	createdAt = time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO inter_network_message(message_id,idempotency_key,sender_network_id,destination_network_id,schema_version,message_type,correlation_id,causation_id,payload_hash,payload,status,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'ACCEPTED',$11)`, envelope.MessageID, envelope.IdempotencyKey, envelope.SenderNetworkID, envelope.DestinationNetworkID, envelope.SchemaVersion, envelope.MessageType, envelope.CorrelationID, envelope.CausationID, hash, envelope.Payload, createdAt)
	if err != nil {
		return AcceptedMessage{}, err
	}
	return AcceptedMessage{Envelope: envelope, PayloadHash: hash, Status: "ACCEPTED", CreatedAt: createdAt}, nil
}
