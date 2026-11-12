package com.cholopay.bankcore.outbox;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;

import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;

import static org.junit.jupiter.api.Assertions.*;

@Tag("integration")
@SpringBootTest
class OutboxRepositoryIntegrationTest {
    @Autowired OutboxRepository repository;
    @Autowired JdbcTemplate jdbc;
    private final UUID eventId = UUID.randomUUID();

    @AfterEach
    void cleanup() {
        jdbc.update("DELETE FROM inter_network_outbox WHERE event_id = ?", eventId);
    }

    @Test
    void onlyOneConcurrentDispatcherOwnsTheLease() throws Exception {
        Instant now = Instant.parse("2026-11-10T00:00:00Z");
        insert("PENDING", 0, now.minusSeconds(1), now.minusSeconds(1), null, null);
        ExecutorService pool = Executors.newFixedThreadPool(2);
        CountDownLatch ready = new CountDownLatch(2);
        Future<java.util.Optional<OutboxRecord>> first = pool.submit(() -> claimAfterReady(ready, now));
        Future<java.util.Optional<OutboxRecord>> second = pool.submit(() -> claimAfterReady(ready, now));
        List<java.util.Optional<OutboxRecord>> claims = List.of(first.get(), second.get());
        pool.shutdownNow();

        assertEquals(1, claims.stream().filter(java.util.Optional::isPresent).count());
        assertEquals("IN_FLIGHT", jdbc.queryForObject("SELECT state FROM inter_network_outbox WHERE event_id = ?", String.class, eventId));
        assertEquals(1, jdbc.queryForObject("SELECT attempt_count FROM inter_network_outbox WHERE event_id = ?", Integer.class, eventId));
    }

    @Test
    void expiredLeaseIsRecoveredWithANewToken() {
        Instant now = Instant.parse("2026-11-10T00:00:00Z");
        UUID oldToken = UUID.randomUUID();
        insert("IN_FLIGHT", 2, now.minusSeconds(30), now.minusSeconds(30), oldToken, now.minusSeconds(1));

        OutboxRecord recovered = repository.claimNext(now, Duration.ofSeconds(30)).orElseThrow();

        assertEquals(eventId, recovered.eventId());
        assertEquals(3, recovered.attemptCount());
        assertNotEquals(oldToken, recovered.leaseToken());
        assertEquals("IN_FLIGHT", jdbc.queryForObject("SELECT state FROM inter_network_outbox WHERE event_id = ?", String.class, eventId));
    }

    @Test
    void stateTransitionsRequireTheCurrentLease() {
        Instant now = Instant.parse("2026-11-10T00:00:00Z");
        insert("PENDING", 0, now.minusSeconds(1), now.minusSeconds(1), null, null);
        OutboxRecord claimed = repository.claimNext(now, Duration.ofSeconds(30)).orElseThrow();

        assertFalse(repository.markPublished(eventId, UUID.randomUUID()));
        assertTrue(repository.markRetry(eventId, claimed.leaseToken(), now.plusSeconds(2), "timeout"));
        assertEquals("PENDING", jdbc.queryForObject("SELECT state FROM inter_network_outbox WHERE event_id = ?", String.class, eventId));
    }

    private java.util.Optional<OutboxRecord> claimAfterReady(CountDownLatch ready, Instant now) throws InterruptedException {
        ready.countDown();
        ready.await();
        return repository.claimNext(now, Duration.ofSeconds(30));
    }

    private void insert(String state, int attempts, Instant nextAttempt, Instant lastAttempt, UUID token, Instant leaseUntil) {
        jdbc.update("""
                INSERT INTO inter_network_outbox
                (event_id, aggregate_type, aggregate_id, event_name, schema_version, event_version,
                 occurred_at, correlation_id, causation_id, idempotency_key, trace_id, payload,
                 serialized_envelope, payload_hash, state, attempt_count, next_attempt_at,
                 last_attempt_at, lease_token, lease_until)
                VALUES (?, 'TRANSFER', ?, 'TRANSFER_REQUESTED', '1.0', 1, ?, ?, ?, ?, ?, ?::jsonb, ?, ?, ?, ?, ?, ?, ?, ?)
                """, eventId, UUID.randomUUID(), nextAttempt, UUID.randomUUID(), UUID.randomUUID(),
                "test-" + eventId, "trace-test", "{}", "{\"event_id\":\"stable\"}", "hash", state,
                attempts, nextAttempt, lastAttempt, token, leaseUntil);
    }
}
