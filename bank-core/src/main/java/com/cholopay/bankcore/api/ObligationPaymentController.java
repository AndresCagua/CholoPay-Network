package com.cholopay.bankcore.api;

import com.cholopay.bankcore.service.ObligationPaymentGatewayService;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/v1")
public class ObligationPaymentController {
    private final ObligationPaymentGatewayService service;
    private final String bankApiToken;

    public ObligationPaymentController(ObligationPaymentGatewayService service,
                                       @Value("${BANK_API_TOKEN:}") String bankApiToken) {
        this.service = service;
        this.bankApiToken = bankApiToken;
    }

    @PostMapping({"/obligations/{obligationId}/payment-intents", "/participants/{participantId}/obligations/{obligationId}/payment-intents"})
    public ResponseEntity<?> start(@PathVariable String obligationId,
                                   @PathVariable(value = "participantId", required = false) String participantId,
                                   @RequestHeader(value = "Authorization", required = false) String authorization,
                                   @RequestHeader(value = "Idempotency-Key", required = false) String idempotencyKey,
                                   @RequestBody(required = false) ObligationPaymentRequest request) {
        if (bankApiToken == null || bankApiToken.isBlank()) return ResponseEntity.status(503).body(error("configuration_missing"));
        if (authorization == null || !authorization.equals("Bearer " + bankApiToken)) return ResponseEntity.status(401).body(error("unauthorized"));
        if (idempotencyKey == null || idempotencyKey.isBlank()) return ResponseEntity.badRequest().body(error("idempotency_key_required"));
        if (request == null || (participantId != null && !participantId.equals(request.participantId()))) return ResponseEntity.unprocessableEntity().body(error("participant_mismatch"));
        var result = service.start(obligationId, idempotencyKey,
                new ObligationPaymentGatewayService.ObligationPaymentRequest(request.participantId(), request.sourceAccountId(),
                        request.biller(), request.amount(), request.currency(), request.service(), request.invoiceId()));
        return ResponseEntity.status(result.status()).body(result.body());
    }

    private static ErrorResponse error(String error) { return new ErrorResponse(error); }
    public record ObligationPaymentRequest(String participantId, UUID sourceAccountId, String biller,
                                           String amount, String currency, String service, String invoiceId) {}
    public record ErrorResponse(String error) {}
}
