package com.cholopay.bankcore.api;

import com.cholopay.bankcore.service.PublicAccountService;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/v1")
public class PublicAccountController {
    private final PublicAccountService service;
    private final String participantId;
    private final String bankApiToken;

    public PublicAccountController(PublicAccountService service,
                                   @Value("${cholopay.participant-id:}") String participantId,
                                   @Value("${BANK_API_TOKEN:}") String bankApiToken) {
        this.service = service;
        this.participantId = participantId;
        this.bankApiToken = bankApiToken;
    }

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
