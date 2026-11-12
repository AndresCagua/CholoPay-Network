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

    @PostMapping
    public ResponseEntity<InterNetworkTransferService.OrderResult> order(
            @RequestHeader("Idempotency-Key") String key, @RequestBody Request request) {
        return ResponseEntity.accepted().body(service.order(new InterNetworkTransferService.OrderRequest(
                request.sourceAccountId(), request.destinationAccountId(), request.sourceParticipantId(),
                request.destinationParticipantId(), request.sourceNetworkId(), request.destinationNetworkId(),
                request.amountMinor(), request.currency(), key, request.correlationId(), request.causationId(),
                request.traceId(), request.occurredAt())));
    }

    public record Request(UUID sourceAccountId, UUID destinationAccountId, UUID sourceParticipantId,
                          UUID destinationParticipantId, UUID sourceNetworkId, UUID destinationNetworkId,
                          long amountMinor, String currency, UUID correlationId, UUID causationId,
                          String traceId, java.time.Instant occurredAt) {}
}
