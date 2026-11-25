package com.cholopay.bankcore.api;

import com.cholopay.bankcore.service.InterNetworkTransferService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/internal/v1/inter-network/transfers")
public class InterNetworkTransferController {
    private final InterNetworkTransferService service;
    public InterNetworkTransferController(InterNetworkTransferService service) { this.service = service; }

    @PostMapping("/reserve")
    public ResponseEntity<InterNetworkTransferService.ReservationResult> reserve(
            @RequestHeader("Idempotency-Key") String key, @RequestBody ReservationRequest request) {
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

    public record Request(UUID sourceAccountId, UUID destinationAccountId, UUID sourceParticipantId,
                          UUID destinationParticipantId, UUID sourceNetworkId, UUID destinationNetworkId,
                          long amountMinor, String currency, UUID correlationId, UUID causationId,
                          String traceId, java.time.Instant occurredAt) {}
}
