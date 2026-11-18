package com.cholopay.bankcore.api;

import com.cholopay.bankcore.service.JdbcObligationPaymentService;
import com.cholopay.bankcore.service.ObligationPaymentGatewayService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/v1")
public class ObligationPaymentController {
    private final JdbcObligationPaymentService persistentService;
    private final ObligationPaymentGatewayService legacyService;
    private final String bankApiToken;
    private final String participantId;

    @Autowired
    public ObligationPaymentController(JdbcObligationPaymentService persistentService,
                                       ObligationPaymentGatewayService legacyService,
                                       @Value("${BANK_API_TOKEN:}") String bankApiToken,
                                       @Value("${cholopay.participant-id:}") String participantId) {
        this.persistentService = persistentService;
        this.legacyService = legacyService;
        this.bankApiToken = bankApiToken;
        this.participantId = participantId;
    }

    /** Constructor de compatibilidad para las pruebas unitarias anteriores a la persistencia; producción usa el servicio JDBC anterior. */
    public ObligationPaymentController(ObligationPaymentGatewayService service, String bankApiToken) {
        this.persistentService = null;
        this.legacyService = service;
        this.bankApiToken = bankApiToken;
        this.participantId = "";
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

    @GetMapping("/payment-intents/{paymentIntentId}")
    public ResponseEntity<?> retrieve(@PathVariable UUID paymentIntentId,
                                      @RequestHeader(value = "Authorization", required = false) String authorization) {
        if (!configured()) return problem(503, "configuration_missing", "la configuración bancaria no está disponible");
        try {
            Map<String,Object> body = persistentService.retrieve(participantId(), customerToken(authorization), paymentIntentId);
            return body == null ? problem(404, "payment_intent_not_found", "la intención de pago no está disponible") : ResponseEntity.ok(body);
        } catch (SecurityException e) { return problem(401, "unauthorized", "la autorización del cliente no es válida"); }
    }

    @PostMapping("/payment-intents/{paymentIntentId}/refunds")
    public ResponseEntity<?> refund(@PathVariable UUID paymentIntentId,
                                    @RequestHeader(value = "Authorization", required = false) String authorization,
                                    @RequestHeader(value = "Idempotency-Key", required = false) String idempotencyKey,
                                    @RequestBody(required = false) RefundRequest request) {
        if (!configured()) return problem(503, "configuration_missing", "la configuración bancaria no está disponible");
        if (idempotencyKey == null || idempotencyKey.isBlank()) return problem(400, "idempotency_key_required", "se requiere la clave de idempotencia");
        try {
            Map<String,Object> body = persistentService.refund(participantId(), customerToken(authorization), paymentIntentId, idempotencyKey, request == null ? null : request.amount());
            return body == null ? problem(404, "payment_intent_not_found", "la intención de pago no está disponible") : ResponseEntity.status(201).body(body);
        } catch (SecurityException e) { return problem(401, "unauthorized", "la autorización del cliente no es válida"); }
          catch (IllegalArgumentException e) { return problem(400, e.getMessage(), refundMessage(e.getMessage())); }
          catch (IllegalStateException e) { return problem(409, e.getMessage(), refundMessage(e.getMessage())); }
    }

    private boolean configured() { return persistentService != null && bankApiToken != null && !bankApiToken.isBlank(); }
    private String participantId() { return participantId; }
    private String customerToken(String authorization) { if (authorization == null || !authorization.startsWith("Bearer ") || authorization.length() <= 7) throw new SecurityException(); return authorization.substring(7); }
    private ResponseEntity<Problem> problem(int status, String code, String detail) {
        String title = switch (code) {
            case "configuration_missing" -> "Configuración no disponible";
            case "unauthorized" -> "Autorización inválida";
            case "payment_intent_not_found" -> "Intención de pago no disponible";
            case "idempotency_key_required" -> "Clave de idempotencia obligatoria";
            case "refund_must_be_full" -> "Reembolso completo obligatorio";
            case "refund_amount_invalid" -> "Importe de reembolso inválido";
            case "idempotency_key_conflict" -> "Conflicto de idempotencia";
            case "payment_not_refundable" -> "Intención no reembolsable";
            default -> "Solicitud inválida";
        };
        return ResponseEntity.status(status).contentType(MediaType.valueOf("application/problem+json")).body(new Problem("https://api.cholopay.dev/problems/" + code, title, status, detail, code));
    }
    private String refundMessage(String code) { return switch (code) { case "idempotency_key_required" -> "se requiere la clave de idempotencia"; case "refund_must_be_full" -> "solo se permiten reembolsos completos"; case "refund_amount_invalid" -> "el importe del reembolso no es válido"; case "idempotency_key_conflict" -> "la clave de idempotencia ya fue utilizada con otra solicitud"; case "payment_not_refundable" -> "la intención de pago no puede reembolsarse"; default -> "la operación de reembolso no es válida"; }; }
    public record Problem(String type, String title, int status, String detail, String code) {}
    public record RefundRequest(String amount) {}
    private static ErrorResponse error(String error, String message) { return new ErrorResponse(error, message); }
    public record ObligationPaymentRequest(String participantId, UUID sourceAccountId, String biller,
                                           String amount, String currency, String service, String invoiceId) {}
    public record ErrorResponse(String error, String message) {}
}
