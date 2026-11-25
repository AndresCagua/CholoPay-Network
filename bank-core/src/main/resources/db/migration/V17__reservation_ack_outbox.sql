ALTER TABLE inter_network_outbox DROP CONSTRAINT IF EXISTS inter_network_outbox_aggregate_id_key;
ALTER TABLE inter_network_outbox DROP CONSTRAINT IF EXISTS inter_network_outbox_event_name_check;
ALTER TABLE inter_network_outbox ADD CONSTRAINT inter_network_outbox_event_name_check CHECK (event_name IN ('TRANSFER_REQUESTED','FUNDS_RESERVED'));
CREATE UNIQUE INDEX inter_network_outbox_aggregate_event_key ON inter_network_outbox(aggregate_id, event_name);
