package instruction

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cholopay/network/internal/transfer"
	"github.com/google/uuid"
)

type Instruction struct {
	TransferID               uuid.UUID
	IdempotencyKey           string
	RequestHash              string
	SourceParticipantID      uuid.UUID
	DestinationParticipantID *uuid.UUID
	AmountMinor              int64
	Currency                 string
	State                    transfer.State
	SchemaVersion            int
	CorrelationID            uuid.UUID
	CausationID              *uuid.UUID
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) BeginTx(ctx context.Context) (*sql.Tx, error) { return s.db.BeginTx(ctx, nil) }

func RequestHash(source uuid.UUID, destination *uuid.UUID, amount int64, currency string) string {
	destinationValue := ""
	if destination != nil {
		destinationValue = destination.String()
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%s", source, destinationValue, amount, currency)))
	return hex.EncodeToString(sum[:])
}

func (s *Store) Create(ctx context.Context, instruction Instruction) (Instruction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Instruction{}, err
	}
	defer tx.Rollback()
	existing, err := s.CreateTx(ctx, tx, instruction)
	if err != nil {
		return Instruction{}, err
	}
	if err := tx.Commit(); err != nil {
		return Instruction{}, err
	}
	return existing, nil
}

func (s *Store) CreateTx(ctx context.Context, tx *sql.Tx, instruction Instruction) (Instruction, error) {
	if instruction.TransferID == uuid.Nil {
		instruction.TransferID = uuid.New()
	}
	if instruction.RequestHash == "" {
		instruction.RequestHash = RequestHash(instruction.SourceParticipantID, instruction.DestinationParticipantID, instruction.AmountMinor, instruction.Currency)
	}
	if instruction.SchemaVersion == 0 {
		instruction.SchemaVersion = 1
	}
	if instruction.State == "" {
		instruction.State = transfer.Created
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO transfer_instruction
		(transfer_id, idempotency_key, request_hash, source_participant_id, destination_participant_id, amount_minor, currency, state, schema_version, correlation_id, causation_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (idempotency_key) DO NOTHING`, instruction.TransferID, instruction.IdempotencyKey, instruction.RequestHash, instruction.SourceParticipantID, instruction.DestinationParticipantID, instruction.AmountMinor, instruction.Currency, instruction.State, instruction.SchemaVersion, instruction.CorrelationID, instruction.CausationID)
	if err != nil {
		return Instruction{}, err
	}
	var existing Instruction
	err = tx.QueryRowContext(ctx, `SELECT transfer_id,idempotency_key,request_hash,source_participant_id,destination_participant_id,amount_minor,currency,state,schema_version,correlation_id,causation_id FROM transfer_instruction WHERE idempotency_key=$1 FOR UPDATE`, instruction.IdempotencyKey).Scan(&existing.TransferID, &existing.IdempotencyKey, &existing.RequestHash, &existing.SourceParticipantID, &existing.DestinationParticipantID, &existing.AmountMinor, &existing.Currency, &existing.State, &existing.SchemaVersion, &existing.CorrelationID, &existing.CausationID)
	if err != nil {
		return Instruction{}, err
	}
	if existing.RequestHash != instruction.RequestHash {
		return Instruction{}, ErrIdempotencyConflict
	}
	return existing, nil
}

func (s *Store) Advance(ctx context.Context, id uuid.UUID, next transfer.State, causation uuid.UUID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current transfer.State
	if err = tx.QueryRowContext(ctx, `SELECT state FROM transfer_instruction WHERE transfer_id=$1 FOR UPDATE`, id).Scan(&current); err != nil {
		return err
	}
	if err = transfer.Advance(current, next); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE transfer_instruction SET state=$1, causation_id=$2, updated_at=CURRENT_TIMESTAMP WHERE transfer_id=$3`, next, causation, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Get(ctx context.Context, id uuid.UUID) (Instruction, error) {
	var result Instruction
	err := s.db.QueryRowContext(ctx, `SELECT transfer_id,idempotency_key,request_hash,source_participant_id,destination_participant_id,amount_minor,currency,state,schema_version,correlation_id,causation_id FROM transfer_instruction WHERE transfer_id=$1`, id).Scan(&result.TransferID, &result.IdempotencyKey, &result.RequestHash, &result.SourceParticipantID, &result.DestinationParticipantID, &result.AmountMinor, &result.Currency, &result.State, &result.SchemaVersion, &result.CorrelationID, &result.CausationID)
	return result, err
}

var ErrIdempotencyConflict = errors.New("idempotency key was already used with a different request")
