package com.cholopay.bankcore.api;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.Map;

/** Expone únicamente la identidad pública sintética del banco configurado. */
@RestController
@RequestMapping("/v1/participants")
public class ParticipantController {
    private final String participantId;
    private final String bankApiToken;

    @Autowired
    public ParticipantController(@Value("${cholopay.participant-id:}") String participantId,
                                 @Value("${BANK_API_TOKEN:}") String bankApiToken) {
        this.participantId = participantId;
        this.bankApiToken = bankApiToken;
    }

    @GetMapping("/{requestedParticipantId}")
    public ResponseEntity<?> get(@PathVariable String requestedParticipantId,
                                 @RequestHeader(value = "Authorization", required = false) String authorization) {
        if (!configured()) return problem(503, "configuration_missing", "la configuración bancaria no está disponible");
        if (!authorized(authorization)) return problem(401, "unauthorized", "la autorización bancaria no es válida");
        // No revelar si otro identificador pertenece a una institución distinta.
        if (!participantId.equals(requestedParticipantId)) {
            return problem(404, "participant_not_found", "el participante no está disponible");
        }
        return ResponseEntity.ok(new ParticipantResponse(participantId,
                Map.of("environment", "synthetic", "status", "ACTIVE")));
    }

    private boolean configured() {
        return participantId != null && !participantId.isBlank()
                && bankApiToken != null && !bankApiToken.isBlank();
    }

    private boolean authorized(String authorization) {
        return authorization != null && authorization.equals("Bearer " + bankApiToken);
    }

    private ResponseEntity<Problem> problem(int status, String code, String detail) {
        String title = switch (code) {
            case "configuration_missing" -> "Configuración no disponible";
            case "unauthorized" -> "Autorización inválida";
            case "participant_not_found" -> "Participante no disponible";
            default -> "Solicitud inválida";
        };
        return ResponseEntity.status(status)
                .contentType(MediaType.valueOf("application/problem+json"))
                .body(new Problem("https://api.cholopay.dev/problems/" + code, title, status, detail, code));
    }

    public record ParticipantResponse(@JsonProperty("participant_id") String participantId,
                                      Map<String, String> metadata) {}
    public record Problem(String type, String title, int status, String detail, String code) {}
}
