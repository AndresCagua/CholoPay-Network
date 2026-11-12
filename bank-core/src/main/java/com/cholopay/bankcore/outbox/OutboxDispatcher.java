package com.cholopay.bankcore.outbox;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.time.Instant;
import java.util.Optional;
import java.util.concurrent.ThreadLocalRandom;

@Service
public class OutboxDispatcher {
    private final OutboxRepository repository;
    private final OutboxClient client;
    private final Duration leaseDuration;
    private final int maxAttempts;
    private final Duration baseBackoff;
    private final Duration maxBackoff;

    public OutboxDispatcher(OutboxRepository repository, OutboxClient client,
                            @Value("${cholopay.outbox.lease-duration:30s}") Duration leaseDuration,
                            @Value("${cholopay.outbox.max-attempts:5}") int maxAttempts,
                            @Value("${cholopay.outbox.base-backoff:1s}") Duration baseBackoff,
                            @Value("${cholopay.outbox.max-backoff:1m}") Duration maxBackoff) {
        this(repository, client, leaseDuration, maxAttempts, baseBackoff, maxBackoff, true);
    }

    public OutboxDispatcher(OutboxRepository repository, OutboxClient client, Duration leaseDuration,
                            int maxAttempts, Duration baseBackoff, Duration maxBackoff, boolean ignored) {
        if (maxAttempts < 1) throw new IllegalArgumentException("max-attempts debe ser positivo");
        this.repository = repository; this.client = client; this.leaseDuration = leaseDuration;
        this.maxAttempts = maxAttempts; this.baseBackoff = baseBackoff; this.maxBackoff = maxBackoff;
    }

    public boolean dispatchOnce(Instant now) {
        Optional<OutboxRecord> claimed = repository.claimNext(now, leaseDuration);
        if (claimed.isEmpty()) return false;
        OutboxRecord record = claimed.get();
        OutboxClient.DeliveryResult result = client.send(record.serializedEnvelope());
        switch (result) {
            case SUCCESS, REPLAY -> repository.markPublished(record.eventId(), record.leaseToken());
            case CONTRACT_FAILURE, AUTH_FAILURE, CONFLICT -> repository.markDeadLetter(record.eventId(), record.leaseToken(), result.name());
            case AMBIGUOUS_FAILURE -> {
                if (record.attemptCount() >= maxAttempts) repository.markDeadLetter(record.eventId(), record.leaseToken(), "MAX_ATTEMPTS");
                else repository.markRetry(record.eventId(), record.leaseToken(), now.plus(backoff(record.attemptCount())), result.name());
            }
        }
        return true;
    }

    private Duration backoff(int attempt) {
        long exponential = baseBackoff.toMillis() * (1L << Math.min(30, Math.max(0, attempt - 1)));
        long bounded = Math.min(maxBackoff.toMillis(), exponential);
        long jitter = bounded == 0 ? 0 : ThreadLocalRandom.current().nextLong(Math.max(1, bounded / 2 + 1));
        return Duration.ofMillis(Math.min(maxBackoff.toMillis(), bounded + jitter));
    }
}
