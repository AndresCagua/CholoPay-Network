package directory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Entry struct {
	KeyID         uuid.UUID
	NetworkID     uuid.UUID
	ParticipantID uuid.UUID
	KeyType       string
	KeyValue      string
	AccountRef    string
	AccountKind   string
	Status        string
	Version       int64
}

type Resolution struct {
	Entry          Entry
	SchemaVersion  string
	MessageType    string
	CorrelationID  uuid.UUID
	CausationID    uuid.UUID
	IdempotencyKey string
	ResolvedAt     time.Time
}

var (
	ErrNotFound     = errors.New("directory key not found")
	ErrSuspended    = errors.New("directory key is suspended")
	ErrAmbiguous    = errors.New("directory key is ambiguous")
	ErrPortConflict = errors.New("directory key porting conflict")
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS directory_key (
		key_id UUID PRIMARY KEY,
		network_id UUID NOT NULL REFERENCES payment_network(network_id),
		participant_id UUID NOT NULL REFERENCES network_participant(participant_id),
		key_type VARCHAR(32) NOT NULL,
		key_value VARCHAR(256) NOT NULL,
		account_ref VARCHAR(128) NOT NULL,
		account_kind VARCHAR(16) NOT NULL DEFAULT 'PERSONAL' CHECK (account_kind IN ('PERSONAL','BUSINESS')),
		status VARCHAR(16) NOT NULL CHECK (status IN ('ACTIVE','SUSPENDED')),
		version BIGINT NOT NULL CHECK (version > 0),
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (network_id, key_type, key_value)
	)`)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `ALTER TABLE directory_key ADD COLUMN IF NOT EXISTS account_kind VARCHAR(16) NOT NULL DEFAULT 'PERSONAL'`)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS directory_key_global_active_identity ON directory_key(key_type,key_value) WHERE status='ACTIVE' AND key_type IN ('EMAIL','PHONE','DOCUMENT')`)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS directory_key_lookup ON directory_key(network_id,key_type,key_value,status)`)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS directory_key_operations (idempotency_key VARCHAR(128) PRIMARY KEY, request_fingerprint CHAR(64) NOT NULL, key_id UUID NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`)
	return err
}

var ErrKeyLimit = errors.New("directory key limit exceeded")
var ErrKeyConflict = errors.New("directory key already active")

func (s *Store) Register(ctx context.Context, entry Entry) error {
	kind := entry.AccountKind
	if kind == "" {
		kind = "PERSONAL"
	}
	limit := 5
	if kind == "BUSINESS" {
		limit = 20
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialize registrations for the same account while retaining concurrency across accounts.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, entry.AccountRef); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM directory_key WHERE account_ref=$1 AND status='ACTIVE'`, entry.AccountRef).Scan(&count); err != nil {
		return err
	}
	if count >= limit {
		return ErrKeyLimit
	}
	if entry.KeyType != "RANDOM" {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM directory_key WHERE key_type=$1 AND key_value=$2 AND status='ACTIVE')`, entry.KeyType, entry.KeyValue).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrKeyConflict
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO directory_key(key_id,network_id,participant_id,key_type,key_value,account_ref,account_kind,status,version) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, entry.KeyID, entry.NetworkID, entry.ParticipantID, entry.KeyType, entry.KeyValue, entry.AccountRef, kind, entry.Status, entry.Version); err != nil {
		return err
	}
	return tx.Commit()
}

type PortRequest struct {
	IdempotencyKey    string
	KeyType           string
	KeyValue          string
	AccountRef        string
	AccountKind       string
	FromNetworkID     uuid.UUID
	FromParticipantID uuid.UUID
	ToNetworkID       uuid.UUID
	ToParticipantID   uuid.UUID
	NewKeyID          uuid.UUID
}

func (s *Store) Port(ctx context.Context, request PortRequest) error {
	fingerprint := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", request.KeyType, request.KeyValue, request.AccountRef, request.FromNetworkID, request.FromParticipantID, request.ToNetworkID, request.ToParticipantID)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, request.KeyType+":"+request.KeyValue); err != nil {
		return err
	}
	var existingFingerprint string
	if err := tx.QueryRowContext(ctx, `SELECT request_fingerprint FROM directory_key_operations WHERE idempotency_key=$1`, request.IdempotencyKey).Scan(&existingFingerprint); err == nil {
		if existingFingerprint != fingerprint {
			return ErrPortConflict
		}
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var oldKeyID uuid.UUID
	if err := tx.QueryRowContext(ctx, `SELECT key_id FROM directory_key WHERE network_id=$1 AND participant_id=$2 AND key_type=$3 AND key_value=$4 AND account_ref=$5 AND status='ACTIVE' FOR UPDATE`, request.FromNetworkID, request.FromParticipantID, request.KeyType, request.KeyValue, request.AccountRef).Scan(&oldKeyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE directory_key SET status='SUSPENDED', version=version+1 WHERE key_id=$1`, oldKeyID); err != nil {
		return err
	}
	kind := request.AccountKind
	if kind == "" {
		kind = "PERSONAL"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO directory_key(key_id,network_id,participant_id,key_type,key_value,account_ref,account_kind,status,version) VALUES ($1,$2,$3,$4,$5,$6,$7,'ACTIVE',1)`, request.NewKeyID, request.ToNetworkID, request.ToParticipantID, request.KeyType, request.KeyValue, request.AccountRef, kind); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO directory_key_operations(idempotency_key,request_fingerprint,key_id) VALUES ($1,$2,$3)`, request.IdempotencyKey, fingerprint, request.NewKeyID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Resolve(ctx context.Context, networkID uuid.UUID, keyType, keyValue string, correlationID, causationID uuid.UUID, idempotencyKey string) (Resolution, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key_id,network_id,participant_id,key_type,key_value,account_ref,status,version FROM directory_key WHERE network_id=$1 AND key_type=$2 AND key_value=$3`, networkID, keyType, keyValue)
	if err != nil {
		return Resolution{}, err
	}
	defer rows.Close()
	var entries []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.KeyID, &entry.NetworkID, &entry.ParticipantID, &entry.KeyType, &entry.KeyValue, &entry.AccountRef, &entry.Status, &entry.Version); err != nil {
			return Resolution{}, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return Resolution{}, err
	}
	if len(entries) == 0 {
		return Resolution{}, ErrNotFound
	}
	if len(entries) != 1 {
		return Resolution{}, ErrAmbiguous
	}
	if entries[0].Status != "ACTIVE" {
		return Resolution{}, ErrSuspended
	}
	return Resolution{Entry: entries[0], SchemaVersion: "1.0", MessageType: "DIRECTORY_KEY_RESOLVED", CorrelationID: correlationID, CausationID: causationID, IdempotencyKey: idempotencyKey, ResolvedAt: time.Now().UTC()}, nil
}
