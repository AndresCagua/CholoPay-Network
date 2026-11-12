CREATE EXTENSION IF NOT EXISTS pgcrypto;

UPDATE inter_network_outbox
SET payload = jsonb_build_object(
  'transfer_id', to_jsonb(COALESCE(payload->>'transfer_id', payload->>'transferId')),
  'source_participant_id', to_jsonb(COALESCE(payload->>'source_participant_id', payload->>'sourceParticipantId')),
  'destination_participant_id', to_jsonb(COALESCE(payload->>'destination_participant_id', payload->>'destinationParticipantId')),
  'amount_minor', to_jsonb(COALESCE(payload->>'amount_minor', payload->>'amountMinor')::bigint),
  'currency', to_jsonb(payload->>'currency'),
  'route_kind', to_jsonb(COALESCE(payload->>'route_kind', payload->>'routeKind')),
  'source_network_id', to_jsonb(COALESCE(payload->>'source_network_id', payload->>'sourceNetworkId')),
  'destination_network_id', to_jsonb(COALESCE(payload->>'destination_network_id', payload->>'destinationNetworkId'))
)
WHERE payload ? 'transferId'
   OR payload ? 'sourceParticipantId'
   OR payload ? 'destinationParticipantId'
   OR payload ? 'amountMinor'
   OR payload ? 'routeKind'
   OR payload ? 'sourceNetworkId'
   OR payload ? 'destinationNetworkId';

UPDATE inter_network_outbox
SET serialized_envelope = json_build_object(
  'event_id', event_id,
  'aggregate_type', aggregate_type,
  'aggregate_id', aggregate_id,
  'event_name', event_name,
  'schema_version', schema_version,
  'version', event_version,
  'occurred_at', occurred_at,
  'correlation_id', correlation_id,
  'causation_id', causation_id,
  'idempotency_key', idempotency_key,
  'trace_id', trace_id,
  'payload', payload
)::text,
    payload_hash = encode(digest((json_build_object(
      'event_id', event_id,
      'aggregate_type', aggregate_type,
      'aggregate_id', aggregate_id,
      'event_name', event_name,
      'schema_version', schema_version,
      'version', event_version,
      'occurred_at', occurred_at,
      'correlation_id', correlation_id,
      'causation_id', causation_id,
      'idempotency_key', idempotency_key,
      'trace_id', trace_id,
      'payload', payload
    )::text)::bytea, 'sha256'), 'hex');
