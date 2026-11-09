package com.cholopay.bankcore.api;

import com.cholopay.bankcore.service.ReceiverCreditService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/internal/v1/receiver-credits")
public class ReceiverCreditController {
    private final ReceiverCreditService service;
    public ReceiverCreditController(ReceiverCreditService service) { this.service = service; }

    @PostMapping
    public ResponseEntity<ReceiverCreditService.Receipt> apply(@RequestBody ReceiverCreditRequest request) {
        var receipt = service.apply(request.deliveryId(), request.eventId(), request.transferId(), request.destinationAccountId(), request.clearingAccountId(), request.amountMinor());
        return ResponseEntity.ok(receipt);
    }

    public record ReceiverCreditRequest(UUID deliveryId, UUID eventId, UUID transferId, UUID destinationAccountId, UUID clearingAccountId, long amountMinor) {}
}
