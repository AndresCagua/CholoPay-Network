package com.cholopay.bankcore.outbox;

import java.time.Instant;
import java.util.UUID;

public record OutboxRecord(UUID eventId, String serializedEnvelope, int attemptCount,
                           UUID leaseToken, Instant leaseUntil) {}
