package settlement

import (
	"context"
	"errors"
	"github.com/google/uuid"
)

type Delivery struct {
	DeliveryID            uuid.UUID
	TransferID            uuid.UUID
	ReceiverParticipantID uuid.UUID
	State                 string
	AttemptCount          int
	LastEventID           *uuid.UUID
}

var ErrDeliveryConflict = errors.New("receiver acknowledgement conflicts with existing delivery evidence")

func (s *Store) EnqueueDelivery(ctx context.Context, transferID, receiverID uuid.UUID) (Delivery, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Delivery{}, err
	}
	defer tx.Rollback()
	var delivery Delivery
	err = tx.QueryRowContext(ctx, `INSERT INTO settlement_delivery(delivery_id,transfer_id,receiver_participant_id,state) VALUES ($1,$2,$3,'PENDING') ON CONFLICT (transfer_id) DO UPDATE SET updated_at=CURRENT_TIMESTAMP RETURNING delivery_id,transfer_id,receiver_participant_id,state,attempt_count,last_event_id`, uuid.New(), transferID, receiverID).Scan(&delivery.DeliveryID, &delivery.TransferID, &delivery.ReceiverParticipantID, &delivery.State, &delivery.AttemptCount, &delivery.LastEventID)
	if err != nil {
		return Delivery{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE transfer_instruction SET state='CREDIT_PENDING',updated_at=CURRENT_TIMESTAMP WHERE transfer_id=$1 AND state='SETTLED'`, transferID); err != nil {
		return Delivery{}, err
	}
	if err = tx.Commit(); err != nil {
		return Delivery{}, err
	}
	return delivery, nil
}

func (s *Store) AcknowledgeDelivery(ctx context.Context, deliveryID, receiverID, eventID uuid.UUID) (Delivery, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Delivery{}, err
	}
	defer tx.Rollback()
	var delivery Delivery
	err = tx.QueryRowContext(ctx, `SELECT delivery_id,transfer_id,receiver_participant_id,state,attempt_count,last_event_id FROM settlement_delivery WHERE delivery_id=$1 FOR UPDATE`, deliveryID).Scan(&delivery.DeliveryID, &delivery.TransferID, &delivery.ReceiverParticipantID, &delivery.State, &delivery.AttemptCount, &delivery.LastEventID)
	if err != nil {
		return Delivery{}, err
	}
	if delivery.ReceiverParticipantID != receiverID {
		return Delivery{}, ErrDeliveryConflict
	}
	if delivery.State == "APPLIED" {
		if delivery.LastEventID == nil || *delivery.LastEventID != eventID {
			return Delivery{}, ErrDeliveryConflict
		}
		return delivery, nil
	}
	if _, err = tx.ExecContext(ctx, `UPDATE settlement_delivery SET state='APPLIED',last_event_id=$1,updated_at=CURRENT_TIMESTAMP,applied_at=CURRENT_TIMESTAMP WHERE delivery_id=$2`, eventID, deliveryID); err != nil {
		return Delivery{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE transfer_instruction SET state='COMPLETED',updated_at=CURRENT_TIMESTAMP WHERE transfer_id=$1 AND state='CREDIT_PENDING'`, delivery.TransferID); err != nil {
		return Delivery{}, err
	}
	delivery.State, delivery.LastEventID = "APPLIED", &eventID
	if err = tx.Commit(); err != nil {
		return Delivery{}, err
	}
	return delivery, nil
}
