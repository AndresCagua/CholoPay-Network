package com.cholopay.bankcore.service;

import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.UUID;

/** Application boundary for a bank-owned inter-network order. */
@Service
public class InterNetworkTransferService {
    private final JdbcInternalTransferService journal;

    public InterNetworkTransferService(JdbcInternalTransferService journal) { this.journal = journal; }

    public OrderResult order(OrderRequest request) { return journal.transferInterNetwork(request); }

    public record OrderRequest(UUID sourceAccountId, UUID destinationAccountId,
                               UUID sourceParticipantId, UUID destinationParticipantId,
                               UUID sourceNetworkId, UUID destinationNetworkId,
                               long amountMinor, String currency, String idempotencyKey,
                               UUID correlationId, UUID causationId, String traceId,
                               Instant occurredAt) {
        public OrderRequest {
            if (idempotencyKey == null || idempotencyKey.isBlank()) throw new IllegalArgumentException("idempotency_key requerido");
            if (currency == null || !currency.equals("USD")) throw new IllegalArgumentException("currency debe ser USD");
            if (occurredAt == null) occurredAt = Instant.now();
            if (causationId == null) causationId = correlationId;
            if (traceId == null || traceId.isBlank()) traceId = "synthetic-" + correlationId;
        }
    }

    public record OrderResult(UUID transferId, UUID eventId, String payloadHash, String state) {}
}
