package settlement

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"time"
)

type Delivery struct {
	DeliveryID            uuid.UUID
	TransferID            uuid.UUID
	ReceiverParticipantID uuid.UUID
	DestinationAccountID  uuid.UUID
	ClearingAccountID     uuid.UUID
	AmountMinor           int64
	EventID               uuid.UUID
	State                 string
	AttemptCount          int
	LastEventID           *uuid.UUID
	LeaseOwner            *uuid.UUID
}

var ErrDeliveryConflict = errors.New("la confirmación del receptor entra en conflicto con la evidencia de entrega existente")

// EnqueueEligibleDeliveries is the explicit post-settlement trigger owned by the worker.
func (s *Store) EnqueueEligibleDeliveries(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT j.transfer_id,j.destination_participant_id,j.destination_account_id,j.clearing_account_id FROM settlement_journal j JOIN transfer_instruction i ON i.transfer_id=j.transfer_id LEFT JOIN settlement_delivery d ON d.transfer_id=j.transfer_id WHERE i.state='SETTLED' AND d.delivery_id IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var transferID, receiverID, destinationAccountID, clearingAccountID uuid.UUID
		if err := rows.Scan(&transferID, &receiverID, &destinationAccountID, &clearingAccountID); err != nil {
			return err
		}
		if _, err := s.EnqueueDelivery(ctx, transferID, receiverID, destinationAccountID, clearingAccountID); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Store) EnqueueDelivery(ctx context.Context, transferID, receiverID uuid.UUID, accounts ...uuid.UUID) (Delivery, error) {
	dest, clear := receiverID, uuid.New()
	if len(accounts) == 2 {
		dest, clear = accounts[0], accounts[1]
	}
	if dest == clear {
		return Delivery{}, errors.New("las cuentas de entrega deben ser distintas")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Delivery{}, err
	}
	defer tx.Rollback()
	var d Delivery
	q := `INSERT INTO settlement_delivery(delivery_id,transfer_id,receiver_participant_id,destination_account_id,clearing_account_id,amount_minor,event_id,state) SELECT $1,j.transfer_id,$2,j.destination_account_id,j.clearing_account_id,j.amount_minor,$3,'PENDING' FROM settlement_journal j WHERE j.transfer_id=$4 ON CONFLICT (transfer_id) DO UPDATE SET updated_at=CURRENT_TIMESTAMP RETURNING delivery_id,transfer_id,receiver_participant_id,destination_account_id,clearing_account_id,amount_minor,event_id,state,attempt_count,last_event_id,lease_owner`
	err = tx.QueryRowContext(ctx, q, uuid.New(), receiverID, uuid.New(), transferID).Scan(&d.DeliveryID, &d.TransferID, &d.ReceiverParticipantID, &d.DestinationAccountID, &d.ClearingAccountID, &d.AmountMinor, &d.EventID, &d.State, &d.AttemptCount, &d.LastEventID, &d.LeaseOwner)
	if err != nil {
		return Delivery{}, err
	}
	if len(accounts) == 2 {
		if _, err = tx.ExecContext(ctx, `UPDATE settlement_delivery SET destination_account_id=$1,clearing_account_id=$2 WHERE delivery_id=$3 AND attempt_count=0`, dest, clear, d.DeliveryID); err != nil {
			return Delivery{}, err
		}
		d.DestinationAccountID, d.ClearingAccountID = dest, clear
	}
	if _, err = tx.ExecContext(ctx, `UPDATE transfer_instruction SET state='CREDIT_PENDING',updated_at=CURRENT_TIMESTAMP WHERE transfer_id=$1 AND state='SETTLED'`, transferID); err != nil {
		return Delivery{}, err
	}
	if err = tx.Commit(); err != nil {
		return Delivery{}, err
	}
	return d, nil
}
func (s *Store) AcknowledgeDelivery(ctx context.Context, deliveryID, receiverID, eventID uuid.UUID) (Delivery, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Delivery{}, err
	}
	defer tx.Rollback()
	var d Delivery
	err = tx.QueryRowContext(ctx, `SELECT delivery_id,transfer_id,receiver_participant_id,destination_account_id,clearing_account_id,amount_minor,event_id,state,attempt_count,last_event_id,lease_owner FROM settlement_delivery WHERE delivery_id=$1 FOR UPDATE`, deliveryID).Scan(&d.DeliveryID, &d.TransferID, &d.ReceiverParticipantID, &d.DestinationAccountID, &d.ClearingAccountID, &d.AmountMinor, &d.EventID, &d.State, &d.AttemptCount, &d.LastEventID, &d.LeaseOwner)
	if err != nil {
		return Delivery{}, err
	}
	if d.ReceiverParticipantID != receiverID || d.EventID != eventID {
		return Delivery{}, ErrDeliveryConflict
	}
	if d.State == "APPLIED" {
		return d, nil
	}
	if _, err = tx.ExecContext(ctx, `UPDATE settlement_delivery SET state='APPLIED',last_event_id=$1,updated_at=CURRENT_TIMESTAMP,applied_at=CURRENT_TIMESTAMP,lease_owner=NULL,lease_until=NULL WHERE delivery_id=$2`, eventID, deliveryID); err != nil {
		return Delivery{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE transfer_instruction SET state='COMPLETED',updated_at=CURRENT_TIMESTAMP WHERE transfer_id=$1 AND state='CREDIT_PENDING'`, d.TransferID); err != nil {
		return Delivery{}, err
	}
	d.State = "APPLIED"
	d.LastEventID = &eventID
	if err = tx.Commit(); err != nil {
		return Delivery{}, err
	}
	return d, nil
}
func (s *Store) ClaimDelivery(ctx context.Context, owner uuid.UUID, lease time.Duration) (Delivery, error) {
	seconds, err := durationSeconds(lease)
	if err != nil {
		return Delivery{}, err
	}
	var d Delivery
	err = s.db.QueryRowContext(ctx, `WITH candidate AS (SELECT delivery_id FROM settlement_delivery WHERE state='PENDING' AND next_attempt_at<=CURRENT_TIMESTAMP AND (lease_until IS NULL OR lease_until<CURRENT_TIMESTAMP) ORDER BY next_attempt_at,created_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE settlement_delivery d SET lease_owner=$1,lease_until=CURRENT_TIMESTAMP+($2 * INTERVAL '1 second'),attempt_count=attempt_count+1,updated_at=CURRENT_TIMESTAMP FROM candidate WHERE d.delivery_id=candidate.delivery_id RETURNING d.delivery_id,d.transfer_id,d.receiver_participant_id,d.destination_account_id,d.clearing_account_id,d.amount_minor,d.event_id,d.state,d.attempt_count,d.last_event_id,d.lease_owner`, owner, seconds).Scan(&d.DeliveryID, &d.TransferID, &d.ReceiverParticipantID, &d.DestinationAccountID, &d.ClearingAccountID, &d.AmountMinor, &d.EventID, &d.State, &d.AttemptCount, &d.LastEventID, &d.LeaseOwner)
	return d, err
}

func durationSeconds(value time.Duration) (int64, error) {
	if value <= 0 || value%time.Second != 0 {
		return 0, errors.New("la duración debe expresarse en segundos enteros positivos")
	}
	return int64(value / time.Second), nil
}
