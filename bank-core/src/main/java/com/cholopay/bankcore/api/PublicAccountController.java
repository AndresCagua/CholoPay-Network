package com.cholopay.bankcore.api;

import com.cholopay.bankcore.service.JdbcPublicLedgerService;
import com.cholopay.bankcore.service.PublicAccountService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/v1")
public class PublicAccountController {
    private final PublicAccountService service;
    private final JdbcPublicLedgerService ledger;
    private final String participantId;
    private final String bankApiToken;

    @Autowired
    public PublicAccountController(PublicAccountService service, JdbcPublicLedgerService ledger,
                                   @Value("${cholopay.participant-id:}") String participantId,
                                   @Value("${BANK_API_TOKEN:}") String bankApiToken) {
        this.service = service; this.ledger = ledger;
        this.participantId = participantId;
        this.bankApiToken = bankApiToken;
    }

    public PublicAccountController(PublicAccountService service,
                                   @Value("${cholopay.participant-id:}") String participantId,
                                   @Value("${BANK_API_TOKEN:}") String bankApiToken) {
        this(service, null, participantId, bankApiToken);
    }

    /* Los endpoints T2 usan Problem Details; los endpoints legacy conservan ErrorResponse. */
    @GetMapping("/accounts/{accountId}/movements")
    public ResponseEntity<?> movements(@PathVariable UUID accountId, @RequestParam(defaultValue = "25") int limit,
                                       @RequestParam(required = false) String cursor,
                                       @RequestHeader(value = "Authorization", required = false) String authorization) {
        return ledgerRead(accountId, limit, cursor, authorization, true);
    }

    @GetMapping("/accounts/{accountId}/journals")
    public ResponseEntity<?> journals(@PathVariable UUID accountId, @RequestParam(defaultValue = "25") int limit,
                                      @RequestParam(required = false) String cursor,
                                      @RequestHeader(value = "Authorization", required = false) String authorization) {
        return ledgerRead(accountId, limit, cursor, authorization, false);
    }

    private ResponseEntity<?> ledgerRead(UUID accountId, int limit, String cursor, String authorization, boolean movements) {
        if (!configured()) return problem(503, "configuration_missing", "la configuración bancaria no está disponible");
        if (ledger == null) return problem(503, "configuration_missing", "la consulta del ledger no está disponible");
        if (limit < 1 || limit > 100) return problem(400, "invalid_limit", "el límite debe estar entre 1 y 100");
        try {
            String token = customerToken(authorization);
            return ResponseEntity.ok().contentType(MediaType.APPLICATION_JSON)
                    .body(movements ? ledger.movements(participantId, token, accountId, limit, cursor) : ledger.journals(participantId, token, accountId, limit, cursor));
        } catch (SecurityException e) { return problem(401, "unauthorized", "la autorización del cliente no es válida"); }
          catch (JdbcPublicLedgerService.AccountNotAccessibleException e) { return problem(404, "account_not_found", "la cuenta no está disponible"); }
          catch (JdbcPublicLedgerService.InvalidCursorException e) { return problem(400, "invalid_cursor", "el cursor no es válido o no corresponde a esta cuenta y recurso"); }
    }

    private ResponseEntity<Problem> problem(int status, String code, String detail) {
        String title = switch (code) {
            case "configuration_missing" -> "Configuración no disponible";
            case "invalid_limit" -> "Límite inválido";
            case "unauthorized" -> "Autorización inválida";
            case "account_not_found" -> "Cuenta no disponible";
            case "invalid_cursor" -> "Cursor inválido";
            default -> "Solicitud inválida";
        };
        return ResponseEntity.status(status).contentType(MediaType.valueOf("application/problem+json"))
                .body(new Problem("https://api.cholopay.dev/problems/" + code, title, status, detail, code));
    }

    public record Problem(String type, String title, int status, String detail, String code) {}

    /** La alta es una operación del banco; el token emitido limita las lecturas al cliente creado. */
    @PostMapping("/customers")
    public ResponseEntity<?> onboard(@RequestHeader(value = "Authorization", required = false) String authorization,
                                     @RequestBody(required = false) CustomerRequest request) {
        if (!configured()) return ResponseEntity.status(503).body(error("configuration_missing", "la configuración bancaria no está disponible"));
        if (!authorized(authorization)) return ResponseEntity.status(401).body(error("unauthorized", "la autorización bancaria no es válida"));
        try {
            if (request == null) throw new IllegalArgumentException("la referencia sintética del cliente es obligatoria");
            var customer = service.onboard(participantId, request.syntheticReference());
            return ResponseEntity.status(201).body(Map.of("customer_id", customer.customerId(), "synthetic_reference", customer.syntheticReference(),
                    "access_token", customer.accessToken(), "accounts", customer.accounts()));
        } catch (IllegalArgumentException invalid) {
            return ResponseEntity.unprocessableEntity().body(error("invalid_customer", invalid.getMessage()));
        }
    }

    @GetMapping("/accounts")
    public ResponseEntity<?> accounts(@RequestHeader(value = "Authorization", required = false) String authorization) {
        if (!configured()) return ResponseEntity.status(503).body(error("configuration_missing", "la configuración bancaria no está disponible"));
        try { return ResponseEntity.ok(Map.of("accounts", service.accounts(participantId, customerToken(authorization)))); }
        catch (SecurityException unauthorized) { return ResponseEntity.status(401).body(error("unauthorized", "la autorización del cliente no es válida")); }
    }

    @GetMapping("/accounts/{accountId}/balance")
    public ResponseEntity<?> balance(@PathVariable UUID accountId,
                                     @RequestHeader(value = "Authorization", required = false) String authorization) {
        if (!configured()) return ResponseEntity.status(503).body(error("configuration_missing", "la configuración bancaria no está disponible"));
        try {
            var balance = service.balance(participantId, customerToken(authorization), accountId);
            if (balance == null) return ResponseEntity.notFound().build();
            return ResponseEntity.ok(balance);
        } catch (SecurityException unauthorized) { return ResponseEntity.status(401).body(error("unauthorized", "la autorización del cliente no es válida")); }
    }

    private boolean configured() { return participantId != null && !participantId.isBlank() && bankApiToken != null && !bankApiToken.isBlank(); }
    private boolean authorized(String value) { return value != null && value.equals("Bearer " + bankApiToken); }
    private String customerToken(String value) {
        if (value == null || !value.startsWith("Bearer ") || value.length() <= 7) throw new SecurityException("token ausente");
        return value.substring(7);
    }
    private static ErrorResponse error(String error, String message) { return new ErrorResponse(error, message); }
    public record CustomerRequest(@JsonProperty("synthetic_reference") String syntheticReference) {}
    public record ErrorResponse(String error, String message) {}
}
