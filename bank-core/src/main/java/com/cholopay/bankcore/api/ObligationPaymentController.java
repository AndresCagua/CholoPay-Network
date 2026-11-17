package com.cholopay.bankcore.api;

import com.cholopay.bankcore.service.JdbcObligationPaymentService;
import com.cholopay.bankcore.service.ObligationPaymentGatewayService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/v1")
public class ObligationPaymentController {
    private final JdbcObligationPaymentService persistentService;
    private final ObligationPaymentGatewayService legacyService;
    private final String bankApiToken;

    @Autowired
    public ObligationPaymentController(JdbcObligationPaymentService service,
                                       @Value("${BANK_API_TOKEN:}") String bankApiToken) {
        this.persistentService = service;
        this.legacyService = null;
        this.bankApiToken = bankApiToken;
    }

    /** Compatibility constructor for the pre-persistence unit tests; production wiring uses the JDBC service above. */
    public ObligationPaymentController(ObligationPaymentGatewayService service, String bankApiToken) {
        this.persistentService = null;
        this.legacyService = service;
        this.bankApiToken = bankApiToken;
    }

    @PostMapping({"/obligations/{obligationId}/payment-intents", "/participants/{participantId}/obligations/{obligationId}/payment-intents"})
    public ResponseEntity<?> start(@PathVariable String obligationId,
                                   @PathVariable(value = "participantId", required = false) String participantId,
                                   @RequestHeader(value = "Authorization", required = false) String authorization,
                                   @RequestHeader(value = "Idempotency-Key", required = false) String idempotencyKey,
                                   @RequestBody(required = false) ObligationPaymentRequest request) {
        if (bankApiToken == null || bankApiToken.isBlank()) return ResponseEntity.status(503).body(error("configuración bancaria no disponible"));
        if (authorization == null || !authorization.equals("Bearer " + bankApiToken)) return ResponseEntity.status(401).body(error("autorización inválida"));
        if (idempotencyKey == null || idempotencyKey.isBlank()) return ResponseEntity.badRequest().body(error("se requiere la clave de idempotencia"));
        if (request == null || (participantId != null && !participantId.equals(request.participantId()))) return ResponseEntity.unprocessableEntity().body(error("el participante no coincide"));
        if (persistentService != null) {
            var result = persistentService.start(obligationId, idempotencyKey,
                    new JdbcObligationPaymentService.Request(request.participantId(), request.sourceAccountId(), request.biller(), request.amount(), request.currency(), request.service(), request.invoiceId()));
            return ResponseEntity.status(result.status()).body(result.body());
        }
        var result = legacyService.start(obligationId, idempotencyKey,
                new ObligationPaymentGatewayService.ObligationPaymentRequest(request.participantId(), request.sourceAccountId(), request.biller(), request.amount(), request.currency(), request.service(), request.invoiceId()));
        return ResponseEntity.status(result.status()).body(result.body());
    }

    private static ErrorResponse error(String error) { return new ErrorResponse(error); }
    public record ObligationPaymentRequest(String participantId, UUID sourceAccountId, String biller,
                                           String amount, String currency, String service, String invoiceId) {}
    public record ErrorResponse(String error) {}
}
