package com.cholopay.bankcore.domain;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.HexFormat;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.UUID;

/** Immutable, transport-independent envelope persisted by the bank outbox. */
public record OutboxEvent(UUID eventId, String aggregateType, UUID aggregateId,
                          String eventName, String schemaVersion, int version,
                          Instant occurredAt, UUID correlationId, UUID causationId,
                          String idempotencyKey, String traceId, Payload payload,
                          String payloadHash) {
    private static final ObjectMapper JSON = new ObjectMapper().findAndRegisterModules();

    public OutboxEvent {
        if (!"TRANSFER".equals(aggregateType) || !"TRANSFER_REQUESTED".equals(eventName)
                || !"1.0".equals(schemaVersion) || version != 1) {
            throw new IllegalArgumentException("envelope TRANSFER_REQUESTED 1.0 inválido");
        }
        if (payload == null) throw new IllegalArgumentException("payload requerido");
        payloadHash = sha256(canonicalJson());
    }

    public String canonicalJson() {
        Map<String, Object> envelope = new LinkedHashMap<>();
        envelope.put("event_id", eventId);
        envelope.put("aggregate_type", aggregateType);
        envelope.put("aggregate_id", aggregateId);
        envelope.put("event_name", eventName);
        envelope.put("schema_version", schemaVersion);
        envelope.put("version", version);
        envelope.put("occurred_at", occurredAt);
        envelope.put("correlation_id", correlationId);
        envelope.put("causation_id", causationId);
        envelope.put("idempotency_key", idempotencyKey);
        envelope.put("trace_id", traceId);
        envelope.put("payload", payload);
        try { return JSON.writeValueAsString(envelope); }
        catch (JsonProcessingException e) { throw new IllegalStateException("no se pudo serializar el envelope", e); }
    }

    private static String sha256(String value) {
        try { return HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(value.getBytes(StandardCharsets.UTF_8))); }
        catch (Exception e) { throw new IllegalStateException(e); }
    }

    public record Payload(UUID transferId, UUID sourceParticipantId, UUID destinationParticipantId,
                          long amountMinor, String currency, String routeKind,
                          UUID sourceNetworkId, UUID destinationNetworkId) {
        public Payload {
            if (amountMinor <= 0 || !"USD".equals(currency) || !"INTER_NETWORK".equals(routeKind))
                throw new IllegalArgumentException("payload inter-red inválido");
            if (sourceParticipantId.equals(destinationParticipantId))
                throw new IllegalArgumentException("los participantes deben ser distintos");
        }
    }
}
