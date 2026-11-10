package directory

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
	ErrNotFound     = errors.New("clave de directorio no encontrada")
	ErrSuspended    = errors.New("la clave de directorio está suspendida")
	ErrAmbiguous    = errors.New("la clave de directorio es ambigua")
	ErrPortConflict = errors.New("conflicto de portabilidad de la clave de directorio")
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
	_, err = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS directory_key_operations (idempotency_key VARCHAR(128) PRIMARY KEY, request_fingerprint CHAR(64) NOT NULL, key_id UUID NOT NULL, schema_version VARCHAR(32) NOT NULL DEFAULT '1.0', correlation_id UUID, causation_id UUID, trace_id VARCHAR(128), requested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`)
	if err != nil {
		return err
	}
	for _, statement := range []string{
		`ALTER TABLE directory_key_operations ADD COLUMN IF NOT EXISTS schema_version VARCHAR(32) NOT NULL DEFAULT '1.0'`,
		`ALTER TABLE directory_key_operations ADD COLUMN IF NOT EXISTS correlation_id UUID`,
		`ALTER TABLE directory_key_operations ADD COLUMN IF NOT EXISTS causation_id UUID`,
		`ALTER TABLE directory_key_operations ADD COLUMN IF NOT EXISTS trace_id VARCHAR(128)`,
		`ALTER TABLE directory_key_operations ADD COLUMN IF NOT EXISTS requested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`,
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

var ErrKeyLimit = errors.New("se excedió el límite de claves de directorio")
var ErrKeyConflict = errors.New("la clave de directorio ya está activa")

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
	// Serializa los registros de la misma cuenta y conserva la concurrencia entre cuentas.
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
	SchemaVersion     string
	CorrelationID     uuid.UUID
	CausationID       uuid.UUID
	TraceID           string
	RequestedAt       time.Time
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

type portFingerprint struct {
	SchemaVersion     string    `json:"schema_version"`
	CorrelationID     uuid.UUID `json:"correlation_id"`
	CausationID       uuid.UUID `json:"causation_id"`
	TraceID           string    `json:"trace_id"`
	KeyType           string    `json:"key_type"`
	KeyValue          string    `json:"key_value"`
	AccountRef        string    `json:"account_ref"`
	AccountKind       string    `json:"account_kind"`
	FromNetworkID     uuid.UUID `json:"from_network_id"`
	FromParticipantID uuid.UUID `json:"from_participant_id"`
	ToNetworkID       uuid.UUID `json:"to_network_id"`
	ToParticipantID   uuid.UUID `json:"to_participant_id"`
	NewKeyID          uuid.UUID `json:"new_key_id"`
}

func fingerprintForPort(request PortRequest) (string, error) {
	payload, err := json.Marshal(portFingerprint{SchemaVersion: request.SchemaVersion, CorrelationID: request.CorrelationID, CausationID: request.CausationID, TraceID: request.TraceID, KeyType: request.KeyType, KeyValue: request.KeyValue, AccountRef: request.AccountRef, AccountKind: request.AccountKind, FromNetworkID: request.FromNetworkID, FromParticipantID: request.FromParticipantID, ToNetworkID: request.ToNetworkID, ToParticipantID: request.ToParticipantID, NewKeyID: request.NewKeyID})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func (s *Store) Port(ctx context.Context, request PortRequest) error {
	if request.SchemaVersion == "" {
		request.SchemaVersion = "1.0"
	}
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now().UTC()
	}
	if request.TraceID == "" {
		request.TraceID = request.IdempotencyKey
	}
	if request.AccountKind == "" {
		request.AccountKind = "PERSONAL"
	}
	fingerprint, err := fingerprintForPort(request)
	if err != nil {
		return err
	}
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO directory_key(key_id,network_id,participant_id,key_type,key_value,account_ref,account_kind,status,version) VALUES ($1,$2,$3,$4,$5,$6,$7,'ACTIVE',1)`, request.NewKeyID, request.ToNetworkID, request.ToParticipantID, request.KeyType, request.KeyValue, request.AccountRef, request.AccountKind); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO directory_key_operations(idempotency_key,request_fingerprint,key_id,schema_version,correlation_id,causation_id,trace_id,requested_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, request.IdempotencyKey, fingerprint, request.NewKeyID, request.SchemaVersion, request.CorrelationID, request.CausationID, request.TraceID, request.RequestedAt); err != nil {
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
