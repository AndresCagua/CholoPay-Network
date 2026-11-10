package directory

import (
	"context"
	"database/sql"
	"errors"
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
	ErrNotFound  = errors.New("directory key not found")
	ErrSuspended = errors.New("directory key is suspended")
	ErrAmbiguous = errors.New("directory key is ambiguous")
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
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM directory_key WHERE account_ref=$1 AND status='ACTIVE'`, entry.AccountRef).Scan(&count); err != nil {
		return err
	}
	if count >= limit {
		return ErrKeyLimit
	}
	if entry.KeyType != "RANDOM" {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM directory_key WHERE key_type=$1 AND key_value=$2 AND status='ACTIVE')`, entry.KeyType, entry.KeyValue).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrKeyConflict
		}
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO directory_key(key_id,network_id,participant_id,key_type,key_value,account_ref,account_kind,status,version) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, entry.KeyID, entry.NetworkID, entry.ParticipantID, entry.KeyType, entry.KeyValue, entry.AccountRef, kind, entry.Status, entry.Version)
	if err != nil {
		return err
	}
	return nil
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
