package com.cholopay.bankcore.outbox;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.Instant;
import java.util.HexFormat;
import java.util.UUID;

@Component
public class HttpOutboxClient implements OutboxClient {
    private final HttpClient http = HttpClient.newHttpClient();
    private final String endpoint;
    private final String secret;
    private final Duration timeout;

    public HttpOutboxClient(@Value("${cholopay.outbox.network-url:}") String endpoint,
                            @Value("${cholopay.outbox.hmac-secret:}") String secret,
                            @Value("${cholopay.outbox.http-timeout:2s}") Duration timeout) {
        this.endpoint = endpoint;
        this.secret = secret;
        this.timeout = timeout;
    }

    @Override
    public DeliveryResult send(String serializedEnvelope) {
        if (endpoint.isBlank() || secret.isBlank()) return DeliveryResult.AUTH_FAILURE;
        long timestamp = Instant.now().getEpochSecond();
        String requestId = UUID.randomUUID().toString();
        String signature = sign(timestamp, requestId, serializedEnvelope);
        try {
            HttpRequest request = HttpRequest.newBuilder(URI.create(endpoint))
                    .timeout(timeout).header("Content-Type", "application/json")
                    .header("X-CholoPay-Timestamp", Long.toString(timestamp))
                    .header("X-CholoPay-Request-Id", requestId)
                    .header("X-CholoPay-Signature", signature)
                    .POST(HttpRequest.BodyPublishers.ofString(serializedEnvelope, StandardCharsets.UTF_8)).build();
            int status = http.send(request, HttpResponse.BodyHandlers.discarding()).statusCode();
            if (status >= 200 && status < 300) return status == 200 ? DeliveryResult.REPLAY : DeliveryResult.SUCCESS;
            if (status == 401 || status == 403) return DeliveryResult.AUTH_FAILURE;
            if (status == 400 || status == 409 || status == 422) return status == 409 ? DeliveryResult.CONFLICT : DeliveryResult.CONTRACT_FAILURE;
            return DeliveryResult.AMBIGUOUS_FAILURE;
        } catch (Exception e) {
            return DeliveryResult.AMBIGUOUS_FAILURE;
        }
    }

    static String sign(long timestamp, String requestId, String body, String secret) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            return HexFormat.of().formatHex(mac.doFinal((timestamp + "." + requestId + "." + body).getBytes(StandardCharsets.UTF_8)));
        } catch (Exception e) { throw new IllegalStateException("no se pudo firmar la solicitud", e); }
    }
    private String sign(long timestamp, String requestId, String body) { return sign(timestamp, requestId, body, secret); }
}
