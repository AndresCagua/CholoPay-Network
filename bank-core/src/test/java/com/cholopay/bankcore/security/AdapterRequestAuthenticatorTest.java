package com.cholopay.bankcore.security;

import org.junit.jupiter.api.Test;

import java.time.Clock;
import java.time.Instant;
import java.time.ZoneOffset;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class AdapterRequestAuthenticatorTest {
    private final long now = 1_700_000_000L;
    private final AdapterRequestAuthenticator authenticator = new AdapterRequestAuthenticator("synthetic-secret", Clock.fixed(Instant.ofEpochSecond(now), ZoneOffset.UTC), 300);

    @Test
    void acceptsCurrentSignedRequest() {
        String timestamp = Long.toString(now), requestId = UUID.randomUUID().toString(), payload = "delivery:event:transfer";
        assertTrue(authenticator.verify(timestamp, requestId, payload, authenticator.sign(timestamp, requestId, payload)));
    }

    @Test
    void rejectsExpiredAndTamperedRequests() {
        String requestId = UUID.randomUUID().toString();
        String signature = authenticator.sign(Long.toString(now), requestId, "payload");
        assertFalse(authenticator.verify(Long.toString(now - 301), requestId, "payload", signature));
        assertFalse(authenticator.verify(Long.toString(now), requestId, "tampered", signature));
        assertFalse(authenticator.verify("not-a-timestamp", requestId, "payload", signature));
    }
}
