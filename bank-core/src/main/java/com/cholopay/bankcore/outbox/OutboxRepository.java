package com.cholopay.bankcore.outbox;

import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

import java.time.Duration;
import java.time.Instant;
import java.util.Optional;
import java.util.UUID;

@Repository
public class OutboxRepository {
    private final JdbcTemplate jdbc;
    public OutboxRepository(JdbcTemplate jdbc) { this.jdbc = jdbc; }

    @Transactional
    public Optional<OutboxRecord> claimNext(Instant now, Duration leaseDuration) {
        UUID token = UUID.randomUUID();
        return jdbc.query("""
                WITH candidate AS (
                  SELECT event_id FROM inter_network_outbox
                  WHERE (state = 'PENDING' AND next_attempt_at <= ?)
                     OR (state = 'IN_FLIGHT' AND lease_until <= ?)
                  ORDER BY next_attempt_at, created_at
                  FOR UPDATE SKIP LOCKED LIMIT 1
                )
                UPDATE inter_network_outbox o
                   SET state = 'IN_FLIGHT', lease_token = ?, lease_until = ?,
                       attempt_count = attempt_count + 1, last_attempt_at = ?
                  FROM candidate c
                 WHERE o.event_id = c.event_id
                RETURNING o.event_id, o.serialized_envelope, o.attempt_count, o.lease_token, o.lease_until
                """, (rs, row) -> new OutboxRecord(rs.getObject("event_id", UUID.class), rs.getString("serialized_envelope"),
                        rs.getInt("attempt_count"), rs.getObject("lease_token", UUID.class), rs.getTimestamp("lease_until").toInstant()),
                now, now, token, now.plus(leaseDuration), now).stream().findFirst();
    }

    public boolean markPublished(UUID eventId, UUID leaseToken) {
        return jdbc.update("UPDATE inter_network_outbox SET state='PUBLISHED', lease_token=NULL, lease_until=NULL, last_error=NULL WHERE event_id=? AND state='IN_FLIGHT' AND lease_token=?", eventId, leaseToken) == 1;
    }

    public boolean markDeadLetter(UUID eventId, UUID leaseToken, String error) {
        return jdbc.update("UPDATE inter_network_outbox SET state='DEAD_LETTER', lease_token=NULL, lease_until=NULL, last_error=? WHERE event_id=? AND state='IN_FLIGHT' AND lease_token=?", error, eventId, leaseToken) == 1;
    }

    public boolean markRetry(UUID eventId, UUID leaseToken, Instant nextAttempt, String error) {
        return jdbc.update("UPDATE inter_network_outbox SET state='PENDING', lease_token=NULL, lease_until=NULL, next_attempt_at=?, last_error=? WHERE event_id=? AND state='IN_FLIGHT' AND lease_token=?", nextAttempt, error, eventId, leaseToken) == 1;
    }
}
