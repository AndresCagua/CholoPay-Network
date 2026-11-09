package settlement

import (
	"context"
	"github.com/google/uuid"
)

type Difference struct {
	DifferenceID uuid.UUID
	RunID        uuid.UUID
	TransferID   *uuid.UUID
	DeliveryID   *uuid.UUID
	Code         string
	Detail       string
}
type ReconciliationRun struct {
	RunID           uuid.UUID
	DifferenceCount int
	Differences     []Difference
}

func (s *Store) Reconcile(ctx context.Context, runID uuid.UUID) (ReconciliationRun, error) {
	if runID == uuid.Nil {
		runID = uuid.New()
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO reconciliation_run(run_id) VALUES ($1)", runID); err != nil {
		return ReconciliationRun{}, err
	}
	var differences []Difference
	add := func(transferID, deliveryID *uuid.UUID, code, detail string) error {
		difference := Difference{DifferenceID: uuid.New(), RunID: runID, TransferID: transferID, DeliveryID: deliveryID, Code: code, Detail: detail}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO reconciliation_difference(difference_id,run_id,transfer_id,delivery_id,code,detail) VALUES ($1,$2,$3,$4,$5,$6)`, difference.DifferenceID, runID, transferID, deliveryID, code, detail); err != nil {
			return err
		}
		differences = append(differences, difference)
		return nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT j.transfer_id,d.delivery_id,d.state FROM settlement_journal j LEFT JOIN settlement_delivery d ON d.transfer_id=j.transfer_id`)
	if err != nil {
		return ReconciliationRun{}, err
	}
	for rows.Next() {
		var transferID uuid.UUID
		var deliveryID *uuid.UUID
		var state *string
		if err := rows.Scan(&transferID, &deliveryID, &state); err != nil {
			rows.Close()
			return ReconciliationRun{}, err
		}
		if deliveryID == nil {
			if err := add(&transferID, nil, "SETTLED_WITHOUT_DELIVERY", "central journal has no durable receiver delivery"); err != nil {
				rows.Close()
				return ReconciliationRun{}, err
			}
		} else if *state == "PENDING" {
			if err := add(&transferID, deliveryID, "DELIVERY_PENDING", "receiver delivery has not been acknowledged"); err != nil {
				rows.Close()
				return ReconciliationRun{}, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return ReconciliationRun{}, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT d.transfer_id,d.delivery_id,i.state FROM settlement_delivery d LEFT JOIN transfer_instruction i ON i.transfer_id=d.transfer_id WHERE d.state='APPLIED' AND (i.state IS NULL OR i.state <> 'COMPLETED')`)
	if err != nil {
		return ReconciliationRun{}, err
	}
	for rows.Next() {
		var transferID uuid.UUID
		var deliveryID uuid.UUID
		var state *string
		if err := rows.Scan(&transferID, &deliveryID, &state); err != nil {
			rows.Close()
			return ReconciliationRun{}, err
		}
		if err := add(&transferID, &deliveryID, "APPLIED_WITHOUT_COMPLETED_STATE", "delivery evidence is applied but instruction is not completed"); err != nil {
			rows.Close()
			return ReconciliationRun{}, err
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return ReconciliationRun{}, err
	}
	rows.Close()

	if _, err := s.db.ExecContext(ctx, `UPDATE reconciliation_run SET completed_at=CURRENT_TIMESTAMP,difference_count=$1 WHERE run_id=$2`, len(differences), runID); err != nil {
		return ReconciliationRun{}, err
	}
	return ReconciliationRun{RunID: runID, DifferenceCount: len(differences), Differences: differences}, nil
}

func (s *Store) ReconciliationRun(ctx context.Context, runID uuid.UUID) (ReconciliationRun, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT difference_count FROM reconciliation_run WHERE run_id=$1", runID).Scan(&count); err != nil {
		return ReconciliationRun{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT difference_id,run_id,transfer_id,delivery_id,code,detail FROM reconciliation_difference WHERE run_id=$1 ORDER BY observed_at,difference_id`, runID)
	if err != nil {
		return ReconciliationRun{}, err
	}
	defer rows.Close()
	var differences []Difference
	for rows.Next() {
		var difference Difference
		if err := rows.Scan(&difference.DifferenceID, &difference.RunID, &difference.TransferID, &difference.DeliveryID, &difference.Code, &difference.Detail); err != nil {
			return ReconciliationRun{}, err
		}
		differences = append(differences, difference)
	}
	return ReconciliationRun{RunID: runID, DifferenceCount: count, Differences: differences}, rows.Err()
}
