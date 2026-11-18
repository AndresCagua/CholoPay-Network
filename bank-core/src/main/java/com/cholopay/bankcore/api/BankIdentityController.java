package com.cholopay.bankcore.api;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/internal/v1/bank-identity")
public class BankIdentityController {
    private final String participantId;
    private final String bankApiToken;

    public BankIdentityController(@Value("${PARTICIPANT_ID:}") String participantId,
                                  @Value("${BANK_API_TOKEN:}") String bankApiToken) {
        this.participantId = participantId;
        this.bankApiToken = bankApiToken;
    }

    @GetMapping
    public ResponseEntity<?> identity(@RequestHeader(value = "Authorization", required = false) String authorization) {
        if (bankApiToken == null || bankApiToken.isBlank()) {
            return ResponseEntity.status(503).body(new ErrorResponse("configuration_missing", "la configuración bancaria no está disponible"));
        }
        if (authorization == null || !authorization.equals("Bearer " + bankApiToken)) {
            return ResponseEntity.status(401).body(new ErrorResponse("unauthorized", "la autorización bancaria no es válida"));
        }
        return ResponseEntity.ok(new IdentityResponse(participantId));
    }

    public record IdentityResponse(String participantId) {}
    public record ErrorResponse(String error, String message) {}
}
