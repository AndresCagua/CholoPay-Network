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

    public ReservationResult reserve(ReservationRequest request) { return journal.reserveInterNetwork(request); }

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

    public record ReservationRequest(UUID transferId, UUID sourceAccountId, UUID sourceParticipantId,
                                     UUID destinationParticipantId, UUID sourceNetworkId, UUID destinationNetworkId,
                                     long amountMinor, String currency, String idempotencyKey, UUID correlationId, UUID causationId) {
        public ReservationRequest {
            if (transferId == null || transferId.equals(new UUID(0, 0))) throw new IllegalArgumentException("transfer_id requerido");
            if (sourceNetworkId == null || destinationNetworkId == null) throw new IllegalArgumentException("network_id requerido");
            if (idempotencyKey == null || idempotencyKey.isBlank()) throw new IllegalArgumentException("idempotency_key requerido");
            if (currency == null || !currency.equals("USD")) throw new IllegalArgumentException("currency debe ser USD");
            if (causationId == null) causationId = correlationId;
        }
    }

    public record ReservationResult(UUID transferId, UUID reservationId, String state, boolean replayed) {}
}
