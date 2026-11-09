package com.cholopay.bankcore.api;

import com.cholopay.bankcore.domain.Journal;
import com.cholopay.bankcore.service.JdbcInternalTransferService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/v1/transfers/internal")
public class TransferController {
    private final JdbcInternalTransferService service;
    public TransferController(JdbcInternalTransferService service) { this.service = service; }

    @PostMapping
    public ResponseEntity<TransferResponse> transfer(@RequestHeader("Idempotency-Key") String key,
                                                       @RequestBody TransferRequest request) {
        Journal journal = service.transfer(request.sourceAccountId(), request.destinationAccountId(),
                request.amountMinor(), key, request.correlationId());
        return ResponseEntity.accepted().body(new TransferResponse(journal.transactionId(), "COMPLETED"));
    }

    public record TransferRequest(UUID sourceAccountId, UUID destinationAccountId, long amountMinor, UUID correlationId) {}
    public record TransferResponse(UUID transactionId, String state) {}
}
