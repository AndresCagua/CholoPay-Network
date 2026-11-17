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
    public ObligationPaymentController(JdbcObligationPaymentService persistentService,
                                       ObligationPaymentGatewayService legacyService,
                                       @Value("${BANK_API_TOKEN:}") String bankApiToken) {
        this.persistentService = persistentService;
        this.legacyService = legacyService;
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
        if (bankApiToken == null || bankApiToken.isBlank()) return ResponseEntity.status(503).body(error("configuration_missing", "la configuración bancaria no está disponible"));
        if (authorization == null || !authorization.equals("Bearer " + bankApiToken)) return ResponseEntity.status(401).body(error("unauthorized", "la autorización bancaria no es válida"));
        if (idempotencyKey == null || idempotencyKey.isBlank()) return ResponseEntity.badRequest().body(error("idempotency_key_required", "se requiere la clave de idempotencia"));
        if (request == null || (participantId != null && !participantId.equals(request.participantId()))) return ResponseEntity.unprocessableEntity().body(error("participant_mismatch", "el participante no coincide"));
        boolean persistentElectricity = "utilities".equals(request.biller()) && "electricity".equals(request.service());
        boolean legacyCompatibility = "transit".equals(request.biller())
                || ("utilities".equals(request.biller()) && request.service() != null && java.util.Set.of("water", "internet", "mobile").contains(request.service()));
        if (!persistentElectricity && !legacyCompatibility) {
            return ResponseEntity.unprocessableEntity().body(error("unsupported_obligation", "la obligación sintética no es compatible"));
        }
        var legacyRequest = new ObligationPaymentGatewayService.ObligationPaymentRequest(request.participantId(), request.sourceAccountId(), request.biller(), request.amount(), request.currency(), request.service(), request.invoiceId());
        if (persistentElectricity && persistentService != null) {
            var result = persistentService.start(obligationId, idempotencyKey,
                    new JdbcObligationPaymentService.Request(request.participantId(), request.sourceAccountId(), request.biller(), request.amount(), request.currency(), request.service(), request.invoiceId()));
            return ResponseEntity.status(result.status()).body(result.body());
        }
        if (legacyService == null) return ResponseEntity.status(503).body(error("obligation_application_unavailable", "la aplicación de obligaciones no está disponible"));
        var result = legacyService.start(obligationId, idempotencyKey, legacyRequest);
        return ResponseEntity.status(result.status()).body(result.body());
    }

    private static ErrorResponse error(String error, String message) { return new ErrorResponse(error, message); }
    public record ObligationPaymentRequest(String participantId, UUID sourceAccountId, String biller,
                                           String amount, String currency, String service, String invoiceId) {}
    public record ErrorResponse(String error, String message) {}
}
