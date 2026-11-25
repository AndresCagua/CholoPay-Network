package settlement

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/cholopay/network/internal/instruction"
	"github.com/google/uuid"
)

type Direction string

const (
	Debit  Direction = "DEBIT"
	Credit Direction = "CREDIT"
)

type Entry struct {
	ParticipantID uuid.UUID
	AmountMinor   int64
	Direction     Direction
	Currency      string
}
type Journal struct {
	JournalID                uuid.UUID
	TransferID               uuid.UUID
	IdempotencyKey           string
	RequestHash              string
	SourceParticipantID      uuid.UUID
	DestinationParticipantID uuid.UUID
	AmountMinor              int64
	Currency                 string
	CorrelationID            uuid.UUID
	DestinationAccountID     uuid.UUID
	ClearingAccountID        uuid.UUID
	Entries                  []Entry
}

type Store struct{ db *sql.DB }

type OperatorAccountRequest struct {
	ParticipantID       uuid.UUID
	OpeningBalanceMinor int64
	Currency            string
	IdempotencyKey      string
	Reason              string
}

type OperatorAccountResult struct {
	OperationID      uuid.UUID
	FundingJournalID uuid.UUID
	ParticipantID    uuid.UUID
	BalanceMinor     int64
	Currency         string
	Status           string
	Replayed         bool
}

type SettlementAccountView struct {
	ParticipantID uuid.UUID
	BalanceMinor  int64
	Currency      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SettlementHistoryEntry struct {
	EventID                  string    `json:"event_id"`
	EventType                string    `json:"event_type"`
	OperationID              string    `json:"operation_id,omitempty"`
	TransferID               string    `json:"transfer_id,omitempty"`
	JournalID                string    `json:"journal_id"`
	SourceParticipantID      string    `json:"source_participant_id,omitempty"`
	DestinationParticipantID string    `json:"destination_participant_id,omitempty"`
	Direction                string    `json:"direction"`
	AmountMinor              int64     `json:"amount_minor"`
	Currency                 string    `json:"currency"`
	Reason                   string    `json:"reason,omitempty"`
	OccurredAt               time.Time `json:"occurred_at"`
}

var (
	ErrAccountExists              = errors.New("la cuenta de liquidación ya existe")
	ErrFundingIdempotencyConflict = errors.New("la clave de fondeo ya se usó con una solicitud diferente")
	ErrInvalidFunding             = errors.New("la solicitud de fondeo es inválida")
	ErrParticipantUnavailable     = errors.New("el participante o la red no están disponibles")
)

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) SettlementAccount(ctx context.Context, participantID uuid.UUID) (SettlementAccountView, error) {
	var account SettlementAccountView
	err := s.db.QueryRowContext(ctx, `SELECT participant_id, balance_minor, currency, created_at, updated_at FROM settlement_account WHERE participant_id=$1`, participantID).Scan(
		&account.ParticipantID, &account.BalanceMinor, &account.Currency, &account.CreatedAt, &account.UpdatedAt,
	)
	return account, err
}

func (s *Store) SettlementHistory(ctx context.Context, participantID uuid.UUID, limit int) ([]SettlementHistoryEntry, error) {
	if participantID == uuid.Nil || limit < 1 || limit > 500 {
		return nil, ErrInvalidFunding
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM settlement_account WHERE participant_id=$1)`, participantID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, sql.ErrNoRows
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT event_id, event_type, operation_id, transfer_id, journal_id,
		       source_participant_id, destination_participant_id, direction,
		       amount_minor, currency, reason, occurred_at
		FROM (
			SELECT fe.entry_id AS event_id, 'FUNDING' AS event_type,
			       fo.operation_id, NULL::uuid AS transfer_id, fj.journal_id,
			       NULL::uuid AS source_participant_id, fe.participant_id AS destination_participant_id,
			       fe.direction, fe.amount_minor, fe.currency, fj.reason, fe.created_at AS occurred_at
			FROM settlement_funding_entry fe
			JOIN settlement_funding_journal fj ON fj.journal_id=fe.journal_id
			JOIN settlement_funding_operation fo ON fo.operation_id=fj.operation_id
			WHERE fe.participant_id=$1
			UNION ALL
			SELECT se.entry_id, 'SETTLEMENT', NULL::uuid, sj.transfer_id, sj.journal_id,
			       sj.source_participant_id, sj.destination_participant_id,
			       se.direction, se.amount_minor, se.currency, NULL::varchar, se.created_at
			FROM settlement_entry se
			JOIN settlement_journal sj ON sj.journal_id=se.journal_id
			WHERE se.participant_id=$1
		) history
		ORDER BY occurred_at DESC, event_id DESC
		LIMIT $2`, participantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []SettlementHistoryEntry{}
	for rows.Next() {
		var entry SettlementHistoryEntry
		var operationID, transferID, sourceID, destinationID, reason sql.NullString
		if err := rows.Scan(&entry.EventID, &entry.EventType, &operationID, &transferID, &entry.JournalID, &sourceID, &destinationID, &entry.Direction, &entry.AmountMinor, &entry.Currency, &reason, &entry.OccurredAt); err != nil {
			return nil, err
		}
		entry.OperationID, entry.TransferID = operationID.String, transferID.String
		entry.SourceParticipantID, entry.DestinationParticipantID, entry.Reason = sourceID.String, destinationID.String, reason.String
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) CreateOperatorAccount(ctx context.Context, request OperatorAccountRequest) (OperatorAccountResult, error) {
	if request.ParticipantID == uuid.Nil || request.OpeningBalanceMinor < 0 || request.Currency != "USD" || request.IdempotencyKey == "" || len(request.IdempotencyKey) > 160 || request.Reason == "" || len(request.Reason) > 256 {
		return OperatorAccountResult{}, ErrInvalidFunding
	}
	requestHash := fundingRequestHash(request)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OperatorAccountResult{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, request.IdempotencyKey); err != nil {
		return OperatorAccountResult{}, err
	}
	var existing OperatorAccountResult
	var existingHash string
	err = tx.QueryRowContext(ctx, `SELECT operation_id, participant_id, amount_minor, currency, request_hash FROM settlement_funding_operation WHERE idempotency_key=$1`, request.IdempotencyKey).Scan(&existing.OperationID, &existing.ParticipantID, &existing.BalanceMinor, &existing.Currency, &existingHash)
	if err == nil {
		if existingHash != requestHash {
			return OperatorAccountResult{}, ErrFundingIdempotencyConflict
		}
		existing.Status = "ACTIVE"
		_ = tx.QueryRowContext(ctx, `SELECT journal_id FROM settlement_funding_journal WHERE operation_id=$1`, existing.OperationID).Scan(&existing.FundingJournalID)
		existing.Replayed = true
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return OperatorAccountResult{}, err
	}
	var participantStatus, networkStatus string
	err = tx.QueryRowContext(ctx, `SELECT p.status, n.status FROM network_participant p JOIN payment_network n ON n.network_id=p.network_id WHERE p.participant_id=$1 FOR SHARE`, request.ParticipantID).Scan(&participantStatus, &networkStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return OperatorAccountResult{}, ErrInvalidFunding
	}
	if err != nil {
		return OperatorAccountResult{}, err
	}
	if participantStatus != "ACTIVE" || networkStatus != "ACTIVE" {
		return OperatorAccountResult{}, ErrParticipantUnavailable
	}
	var accountExists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM settlement_account WHERE participant_id=$1)`, request.ParticipantID).Scan(&accountExists)
	if err != nil {
		return OperatorAccountResult{}, err
	}
	if accountExists {
		return OperatorAccountResult{}, ErrAccountExists
	}
	operationID := uuid.New()
	if _, err = tx.ExecContext(ctx, `INSERT INTO settlement_account(participant_id,balance_minor,currency) VALUES ($1,$2,$3)`, request.ParticipantID, request.OpeningBalanceMinor, request.Currency); err != nil {
		return OperatorAccountResult{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO settlement_funding_operation(operation_id,participant_id,idempotency_key,request_hash,amount_minor,currency,reason) VALUES ($1,$2,$3,$4,$5,$6,$7)`, operationID, request.ParticipantID, request.IdempotencyKey, requestHash, request.OpeningBalanceMinor, request.Currency, request.Reason); err != nil {
		return OperatorAccountResult{}, err
	}
	var journalID uuid.UUID
	if request.OpeningBalanceMinor > 0 {
		journalID = uuid.New()
		if _, err = tx.ExecContext(ctx, `INSERT INTO settlement_funding_journal(journal_id,operation_id,participant_id,amount_minor,currency,reason) VALUES ($1,$2,$3,$4,$5,$6)`, journalID, operationID, request.ParticipantID, request.OpeningBalanceMinor, request.Currency, request.Reason); err != nil {
			return OperatorAccountResult{}, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO settlement_funding_entry(entry_id,journal_id,account_ref,participant_id,amount_minor,direction,currency) VALUES ($1,$2,'SYNTHETIC_OPERATOR_ISSUANCE',NULL,$3,'DEBIT',$4),($5,$2,'SETTLEMENT_PARTICIPANT',$6,$3,'CREDIT',$4)`, uuid.New(), journalID, request.OpeningBalanceMinor, request.Currency, uuid.New(), request.ParticipantID); err != nil {
			return OperatorAccountResult{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return OperatorAccountResult{}, err
	}
	return OperatorAccountResult{OperationID: operationID, FundingJournalID: journalID, ParticipantID: request.ParticipantID, BalanceMinor: request.OpeningBalanceMinor, Currency: request.Currency, Status: "ACTIVE"}, nil
}

func fundingRequestHash(request OperatorAccountRequest) string {
	value := fmt.Sprintf("%s|%d|%s|%s|%s", request.ParticipantID, request.OpeningBalanceMinor, request.Currency, request.IdempotencyKey, request.Reason)
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func (s *Store) CreateAccount(ctx context.Context, participantID uuid.UUID, balanceMinor int64) error {
	if balanceMinor < 0 {
		return errors.New("el saldo de liquidación no puede ser negativo")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO settlement_account(participant_id,balance_minor,currency) VALUES ($1,$2,'USD')`, participantID, balanceMinor)
	return err
}

func (s *Store) SettleReadyOnce(ctx context.Context) (bool, error) {
	var transferID, sourceID, destinationID, destinationAccountID, clearingAccountID, correlationID uuid.UUID
	var instructionKey string
	var amount int64
	err := s.db.QueryRowContext(ctx, `
		SELECT transfer_id, idempotency_key, source_participant_id, destination_participant_id,
		       COALESCE(destination_account_id, destination_participant_id),
		       COALESCE(clearing_account_id, source_participant_id), amount_minor, correlation_id
		FROM transfer_instruction
		WHERE state='READY_FOR_SETTLEMENT'
		ORDER BY updated_at, transfer_id
		LIMIT 1`).Scan(&transferID, &instructionKey, &sourceID, &destinationID, &destinationAccountID, &clearingAccountID, &amount, &correlationID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = s.Settle(ctx, transferID, sourceID, destinationID, amount, "settlement:"+instructionKey, correlationID, destinationAccountID, clearingAccountID)
	return true, err
}

func (s *Store) Settle(ctx context.Context, transferID, sourceID, destinationID uuid.UUID, amountMinor int64, idempotencyKey string, correlationID uuid.UUID, deliveryAccounts ...uuid.UUID) (Journal, error) {
	if amountMinor <= 0 {
		return Journal{}, errors.New("el importe de liquidación debe ser positivo")
	}
	if sourceID == destinationID {
		return Journal{}, errors.New("los participantes de la liquidación deben ser distintos")
	}
	destinationAccountID, clearingAccountID := destinationID, sourceID
	if len(deliveryAccounts) == 2 {
		destinationAccountID, clearingAccountID = deliveryAccounts[0], deliveryAccounts[1]
	}
	if destinationAccountID == uuid.Nil || clearingAccountID == uuid.Nil || destinationAccountID == clearingAccountID {
		return Journal{}, errors.New("las cuentas de entrega deben ser válidas y distintas")
	}
	requestHash := instruction.RequestHashWithAccounts(sourceID, &destinationID, &destinationAccountID, &clearingAccountID, amountMinor, "USD")
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Journal{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", idempotencyKey); err != nil {
		return Journal{}, err
	}
	var existing Journal
	err = tx.QueryRowContext(ctx, `SELECT journal_id,transfer_id,idempotency_key,request_hash,source_participant_id,destination_participant_id,destination_account_id,clearing_account_id,amount_minor,currency,correlation_id FROM settlement_journal WHERE idempotency_key=$1`, idempotencyKey).Scan(&existing.JournalID, &existing.TransferID, &existing.IdempotencyKey, &existing.RequestHash, &existing.SourceParticipantID, &existing.DestinationParticipantID, &existing.DestinationAccountID, &existing.ClearingAccountID, &existing.AmountMinor, &existing.Currency, &existing.CorrelationID)
	if err == nil {
		legacyHash := instruction.RequestHash(existing.SourceParticipantID, &existing.DestinationParticipantID, existing.AmountMinor, existing.Currency)
		if existing.RequestHash != requestHash && !(existing.RequestHash == legacyHash && sameJournalRequest(existing, sourceID, destinationID, destinationAccountID, clearingAccountID, amountMinor)) {
			return Journal{}, ErrIdempotencyConflict
		}
		existing.Entries, err = entries(ctx, tx, existing.JournalID)
		if err != nil {
			return Journal{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Journal{}, err
	}
	var instructionState string
	if err = tx.QueryRowContext(ctx, `SELECT state FROM transfer_instruction WHERE transfer_id=$1 FOR UPDATE`, transferID).Scan(&instructionState); err != nil {
		return Journal{}, err
	}
	if instructionState != "READY_FOR_SETTLEMENT" {
		return Journal{}, errors.New("la transferencia no está lista para liquidación")
	}

	rows, err := tx.QueryContext(ctx, `SELECT participant_id,balance_minor FROM settlement_account WHERE participant_id IN ($1,$2) ORDER BY participant_id FOR UPDATE`, sourceID, destinationID)
	if err != nil {
		return Journal{}, err
	}
	balances := map[uuid.UUID]int64{}
	for rows.Next() {
		var id uuid.UUID
		var balance int64
		if err := rows.Scan(&id, &balance); err != nil {
			rows.Close()
			return Journal{}, err
		}
		balances[id] = balance
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Journal{}, err
	}
	rows.Close()
	if len(balances) != 2 {
		return Journal{}, errors.New("no se encontró la cuenta del participante de liquidación")
	}
	if balances[sourceID] < amountMinor {
		return Journal{}, errors.New("posición de liquidación insuficiente")
	}

	journalID := uuid.New()
	if _, err = tx.ExecContext(ctx, `UPDATE settlement_account SET balance_minor=balance_minor-$1,updated_at=CURRENT_TIMESTAMP WHERE participant_id=$2`, amountMinor, sourceID); err != nil {
		return Journal{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE settlement_account SET balance_minor=balance_minor+$1,updated_at=CURRENT_TIMESTAMP WHERE participant_id=$2`, amountMinor, destinationID); err != nil {
		return Journal{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO settlement_journal(journal_id,transfer_id,idempotency_key,request_hash,source_participant_id,destination_participant_id,destination_account_id,clearing_account_id,amount_minor,currency,correlation_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'USD',$10)`, journalID, transferID, idempotencyKey, requestHash, sourceID, destinationID, destinationAccountID, clearingAccountID, amountMinor, correlationID); err != nil {
		return Journal{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO settlement_entry(entry_id,journal_id,participant_id,amount_minor,direction,currency) VALUES ($1,$2,$3,$4,'DEBIT','USD'),($5,$2,$6,$4,'CREDIT','USD')`, uuid.New(), journalID, sourceID, amountMinor, uuid.New(), destinationID); err != nil {
		return Journal{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE transfer_instruction SET state='SETTLED',updated_at=CURRENT_TIMESTAMP WHERE transfer_id=$1 AND state='READY_FOR_SETTLEMENT'`, transferID); err != nil {
		return Journal{}, err
	}
	result := Journal{JournalID: journalID, TransferID: transferID, IdempotencyKey: idempotencyKey, RequestHash: requestHash, SourceParticipantID: sourceID, DestinationParticipantID: destinationID, AmountMinor: amountMinor, Currency: "USD", CorrelationID: correlationID, DestinationAccountID: destinationAccountID, ClearingAccountID: clearingAccountID, Entries: []Entry{{sourceID, amountMinor, Debit, "USD"}, {destinationID, amountMinor, Credit, "USD"}}}
	if err = tx.Commit(); err != nil {
		return Journal{}, err
	}
	return result, nil
}

func entries(ctx context.Context, tx *sql.Tx, journalID uuid.UUID) ([]Entry, error) {
	rows, err := tx.QueryContext(ctx, `SELECT participant_id,amount_minor,direction,currency FROM settlement_entry WHERE journal_id=$1 ORDER BY entry_id`, journalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ParticipantID, &entry.AmountMinor, &entry.Direction, &entry.Currency); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (j Journal) Balanced() bool {
	if len(j.Entries) != 2 {
		return false
	}
	return j.Entries[0].AmountMinor == j.Entries[1].AmountMinor && j.Entries[0].Direction == Debit && j.Entries[1].Direction == Credit
}

func sameJournalRequest(j Journal, sourceID, destinationID, destinationAccountID, clearingAccountID uuid.UUID, amountMinor int64) bool {
	return j.SourceParticipantID == sourceID && j.DestinationParticipantID == destinationID && j.DestinationAccountID == destinationAccountID && j.ClearingAccountID == clearingAccountID && j.AmountMinor == amountMinor && j.Currency == "USD"
}

var ErrIdempotencyConflict = fmt.Errorf("la clave de idempotencia de liquidación ya se usó con una solicitud diferente")
