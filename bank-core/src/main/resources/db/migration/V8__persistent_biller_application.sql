ALTER TABLE payment_intent ADD COLUMN application_result JSONB;
ALTER TABLE payment_intent DROP CONSTRAINT payment_intent_status_check;
ALTER TABLE payment_intent ADD CONSTRAINT payment_intent_status_check CHECK (status IN ('PENDING_APPLICATION', 'UNKNOWN_PENDING_RECONCILIATION', 'APPLIED'));
ALTER TABLE bill_payment_outbox DROP CONSTRAINT bill_payment_outbox_state_check;
ALTER TABLE bill_payment_outbox ADD CONSTRAINT bill_payment_outbox_state_check CHECK (state IN ('PENDING', 'APPLIED'));
