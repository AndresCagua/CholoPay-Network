package com.cholopay.bankcore.api;

import com.cholopay.bankcore.security.AdapterRequestAuthenticator;
import com.cholopay.bankcore.service.InterNetworkTransferService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/internal/v1/inter-network/transfers")
public class InterNetworkTransferController {
    private final InterNetworkTransferService service;
    private final AdapterRequestAuthenticator authenticator;
    public InterNetworkTransferController(InterNetworkTransferService service, AdapterRequestAuthenticator authenticator) { this.service = service; this.authenticator = authenticator; }

    @PostMapping("/reserve")
    public ResponseEntity<InterNetworkTransferService.ReservationResult> reserve(
            @RequestHeader("X-CholoPay-Request-Timestamp") String timestamp,
            @RequestHeader("X-CholoPay-Request-Id") String requestId,
            @RequestHeader("X-CholoPay-Signature") String signature,
            @RequestHeader("Idempotency-Key") String key, @RequestBody ReservationRequest request) {
        if (request.transferId() == null || request.sourceAccountId() == null || request.sourceParticipantId() == null || request.destinationParticipantId() == null || request.correlationId() == null) {
            return ResponseEntity.badRequest().build();
        }
        if (!authenticator.verify(timestamp, requestId, canonical(request, key), signature)) return ResponseEntity.status(401).build();
        return ResponseEntity.accepted().body(service.reserve(new InterNetworkTransferService.ReservationRequest(
                request.transferId(), request.sourceAccountId(), request.sourceParticipantId(),
                request.destinationParticipantId(), request.amountMinor(), request.currency(), key,
                request.correlationId(), request.causationId())));
    }

    @PostMapping
    public ResponseEntity<InterNetworkTransferService.OrderResult> order(
            @RequestHeader("Idempotency-Key") String key, @RequestBody Request request) {
        return ResponseEntity.accepted().body(service.order(new InterNetworkTransferService.OrderRequest(
                request.sourceAccountId(), request.destinationAccountId(), request.sourceParticipantId(),
                request.destinationParticipantId(), request.sourceNetworkId(), request.destinationNetworkId(),
                request.amountMinor(), request.currency(), key, request.correlationId(), request.causationId(),
                request.traceId(), request.occurredAt())));
    }

    public record ReservationRequest(UUID transferId, UUID sourceAccountId, UUID sourceParticipantId,
                                     UUID destinationParticipantId, long amountMinor, String currency,
                                     UUID correlationId, UUID causationId) {}

    static String canonical(ReservationRequest request, String idempotencyKey) {
        return String.join(":", request.transferId().toString(), request.sourceAccountId().toString(),
                request.sourceParticipantId().toString(), request.destinationParticipantId().toString(),
                Long.toString(request.amountMinor()), request.currency(), idempotencyKey);
    }

    public record Request(UUID sourceAccountId, UUID destinationAccountId, UUID sourceParticipantId,
                          UUID destinationParticipantId, UUID sourceNetworkId, UUID destinationNetworkId,
                          long amountMinor, String currency, UUID correlationId, UUID causationId,
                          String traceId, java.time.Instant occurredAt) {}
}
