package settlement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) CreateAccount(ctx context.Context, participantID uuid.UUID, balanceMinor int64) error {
	if balanceMinor < 0 {
		return errors.New("el saldo de liquidación no puede ser negativo")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO settlement_account(participant_id,balance_minor,currency) VALUES ($1,$2,'USD')`, participantID, balanceMinor)
	return err
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
		if existing.RequestHash != requestHash && !sameJournalRequest(existing, sourceID, destinationID, destinationAccountID, clearingAccountID, amountMinor) {
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
	if _, err = tx.ExecContext(ctx, `UPDATE transfer_instruction SET state='SETTLED',updated_at=CURRENT_TIMESTAMP WHERE transfer_id=$1 AND state IN ('READY_FOR_SETTLEMENT','FUNDS_RESERVED')`, transferID); err != nil {
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
