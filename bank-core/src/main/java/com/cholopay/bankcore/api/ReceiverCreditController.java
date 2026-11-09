package com.cholopay.bankcore.api;

import com.cholopay.bankcore.security.AdapterRequestAuthenticator;
import com.cholopay.bankcore.service.ReceiverCreditService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/internal/v1/receiver-credits")
public class ReceiverCreditController {
    private final ReceiverCreditService service;
    private final AdapterRequestAuthenticator authenticator;
    public ReceiverCreditController(ReceiverCreditService service, AdapterRequestAuthenticator authenticator) { this.service = service; this.authenticator = authenticator; }

    @PostMapping
    public ResponseEntity<ReceiverCreditService.Receipt> apply(
            @RequestHeader("X-CholoPay-Request-Timestamp") String timestamp,
            @RequestHeader("X-CholoPay-Request-Id") String requestId,
            @RequestHeader("X-CholoPay-Signature") String signature,
            @RequestBody ReceiverCreditRequest request) {
        if (!authenticator.verify(timestamp, requestId, canonical(request), signature)) return ResponseEntity.status(401).build();
        var receipt = service.apply(request.deliveryId(), request.eventId(), request.transferId(), request.destinationAccountId(), request.clearingAccountId(), request.amountMinor());
        return ResponseEntity.ok(receipt);
    }

    public record ReceiverCreditRequest(UUID deliveryId, UUID eventId, UUID transferId, UUID destinationAccountId, UUID clearingAccountId, long amountMinor) {}
    public static String canonical(ReceiverCreditRequest request) { return String.join(":", request.deliveryId().toString(), request.eventId().toString(), request.transferId().toString(), request.destinationAccountId().toString(), request.clearingAccountId().toString(), Long.toString(request.amountMinor()), "USD"); }
}
