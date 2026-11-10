package com.cholopay.bankcore.service;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.UUID;

@Service
public class ObligationPaymentGatewayService {
    private final String participantId;
    private final String gatewayUrl;
    private final String gatewayApiKey;
    private final HttpClient httpClient;
    private final ObjectMapper objectMapper;

    @Autowired
    public ObligationPaymentGatewayService(
            @Value("${cholopay.participant-id:}") String participantId,
            @Value("${GATEWAY_URL:${BILLER_GATEWAY_URL:}}") String gatewayUrl,
            @Value("${GATEWAY_API_KEY:}") String gatewayApiKey,
            ObjectMapper objectMapper) {
        this(participantId, gatewayUrl, gatewayApiKey, HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(1)).build(), objectMapper);
    }

    public ObligationPaymentGatewayService(String participantId, String gatewayUrl, String gatewayApiKey, HttpClient httpClient, ObjectMapper objectMapper) {
        this.participantId = participantId;
        this.gatewayUrl = gatewayUrl;
        this.gatewayApiKey = gatewayApiKey;
        this.httpClient = httpClient;
        this.objectMapper = objectMapper;
    }

    public GatewayResult start(String obligationId, String idempotencyKey, ObligationPaymentRequest request) {
        if (isBlank(participantId) || isBlank(gatewayUrl) || isBlank(gatewayApiKey)) {
            return GatewayResult.configurationMissing();
        }
        if (request == null || !participantId.equals(request.participantId()) || request.sourceAccountId() == null
                || isBlank(obligationId) || isBlank(idempotencyKey) || isBlank(request.biller())
                || isBlank(request.amount()) || isBlank(request.currency())) {
            return GatewayResult.invalidRequest();
        }

        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("participant_id", participantId);
        payload.put("biller", request.biller());
        payload.put("obligation_id", obligationId);
        payload.put("source_account_id", request.sourceAccountId());
        payload.put("amount", request.amount());
        payload.put("currency", request.currency());
        if (request.service() != null) payload.put("service", request.service());
        if (request.invoiceId() != null) payload.put("invoice_id", request.invoiceId());
        try {
            HttpRequest gatewayRequest = HttpRequest.newBuilder(URI.create(gatewayUrl.replaceAll("/$", "") + "/v1/bill-payments"))
                    .timeout(Duration.ofSeconds(2))
                    .header("content-type", "application/json")
                    .header("accept", "application/json")
                    .header("x-api-key", gatewayApiKey)
                    .header("idempotency-key", "bank-obligation:" + participantId + ":" + idempotencyKey)
                    .POST(HttpRequest.BodyPublishers.ofString(objectMapper.writeValueAsString(payload)))
                    .build();
            HttpResponse<String> response = httpClient.send(gatewayRequest, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() == 401 || response.statusCode() == 403) return GatewayResult.unauthorized();
            if (response.statusCode() < 200 || response.statusCode() >= 300) return GatewayResult.gatewayUnavailable();
            Map<String, Object> body = objectMapper.readValue(response.body(), new TypeReference<>() {});
            Object state = body.get("status");
            if (!(state instanceof String)) return GatewayResult.gatewayUnavailable();
            body.put("source_account_id", request.sourceAccountId().toString());
            return new GatewayResult(response.statusCode(), body, null);
        } catch (IllegalArgumentException | IOException e) {
            return GatewayResult.gatewayUnavailable();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return GatewayResult.gatewayUnavailable();
        }
    }

    private static boolean isBlank(String value) { return value == null || value.isBlank(); }

    public record ObligationPaymentRequest(String participantId, UUID sourceAccountId, String biller, String amount,
                                           String currency, String service, String invoiceId) {}
    public record GatewayResult(int status, Map<String, Object> body, String error) {
        static GatewayResult configurationMissing() { return new GatewayResult(HttpStatus.SERVICE_UNAVAILABLE.value(), Map.of("error", "configuration_missing"), "configuration_missing"); }
        static GatewayResult invalidRequest() { return new GatewayResult(HttpStatus.UNPROCESSABLE_ENTITY.value(), Map.of("error", "invalid_request"), "invalid_request"); }
        static GatewayResult unauthorized() { return new GatewayResult(HttpStatus.BAD_GATEWAY.value(), Map.of("error", "gateway_unauthorized"), "gateway_unauthorized"); }
        static GatewayResult gatewayUnavailable() { return new GatewayResult(HttpStatus.BAD_GATEWAY.value(), Map.of("error", "gateway_unavailable"), "gateway_unavailable"); }
    }
}
