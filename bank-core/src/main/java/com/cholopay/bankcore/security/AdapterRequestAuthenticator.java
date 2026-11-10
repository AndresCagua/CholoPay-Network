package com.cholopay.bankcore.security;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Clock;
import java.time.Instant;
import java.util.Base64;
import java.util.UUID;

@Component
public class AdapterRequestAuthenticator {
    private final String secret;
    private final Clock clock;
    private final long maxSkewSeconds;

    @Autowired
    public AdapterRequestAuthenticator(@Value("${NETWORK_ADAPTER_SECRET:}") String secret) { this(secret, Clock.systemUTC(), 300); }
    AdapterRequestAuthenticator(String secret, Clock clock, long maxSkewSeconds) { this.secret = secret; this.clock = clock; this.maxSkewSeconds = maxSkewSeconds; }

    public boolean verify(String timestamp, String requestId, String payload, String providedSignature) {
        if (secret == null || secret.isBlank() || timestamp == null || requestId == null || providedSignature == null) return false;
        long seconds;
        try { seconds = Long.parseLong(timestamp); UUID.fromString(requestId); } catch (RuntimeException e) { return false; }
        if (Math.abs(Instant.now(clock).getEpochSecond() - seconds) > maxSkewSeconds) return false;
        String expected = sign(timestamp, requestId, payload);
        return MessageDigest.isEqual(expected.getBytes(StandardCharsets.US_ASCII), providedSignature.getBytes(StandardCharsets.US_ASCII));
    }

    public String sign(String timestamp, String requestId, String payload) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256"); mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            return Base64.getEncoder().encodeToString(mac.doFinal((timestamp + "." + requestId + "." + payload).getBytes(StandardCharsets.UTF_8)));
        } catch (Exception e) { throw new IllegalStateException("no se pudo firmar la solicitud del adaptador", e); }
    }
}
