package com.cholopay.bankcore.outbox;

import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.time.Instant;
import java.util.Optional;
import java.util.UUID;

import static org.mockito.Mockito.*;
import static org.junit.jupiter.api.Assertions.*;

class OutboxDispatcherTest {
    private final OutboxRepository repository = mock(OutboxRepository.class);
    private final OutboxClient client = mock(OutboxClient.class);
    private final UUID eventId = UUID.randomUUID();
    private final UUID lease = UUID.randomUUID();
    private final Instant now = Instant.parse("2026-11-10T00:00:00Z");
    private final String envelope = "{\"event_id\":\"stable\",\"payload\":{\"amount_minor\":250}}";

    private OutboxDispatcher dispatcher(int maxAttempts) {
        return new OutboxDispatcher(repository, client, Duration.ofSeconds(30), maxAttempts,
                Duration.ofSeconds(1), Duration.ofSeconds(10), true);
    }
    private void claim(int attempt) {
        when(repository.claimNext(now, Duration.ofSeconds(30))).thenReturn(Optional.of(new OutboxRecord(eventId, envelope, attempt, lease, now.plusSeconds(30))));
    }

    @Test void successPublishesAndSendsPersistedEnvelopeVerbatim() {
        claim(1); when(client.send(envelope)).thenReturn(OutboxClient.DeliveryResult.SUCCESS);
        assertTrue(dispatcher(5).dispatchOnce(now));
        verify(client).send(envelope); verify(repository).markPublished(eventId, lease);
    }

    @Test void replayPublishesWithoutChangingEnvelope() {
        claim(1); when(client.send(envelope)).thenReturn(OutboxClient.DeliveryResult.REPLAY);
        dispatcher(5).dispatchOnce(now);
        verify(repository).markPublished(eventId, lease);
        verify(client).send(envelope);
    }

    @Test void ambiguousFailureReturnsToPendingWithBoundedBackoff() {
        claim(1); when(client.send(envelope)).thenReturn(OutboxClient.DeliveryResult.AMBIGUOUS_FAILURE);
        dispatcher(5).dispatchOnce(now);
        verify(repository).markRetry(eq(eventId), eq(lease), argThat(value -> !value.isBefore(now) && !value.isAfter(now.plusSeconds(10))), eq("AMBIGUOUS_FAILURE"));
    }

    @Test void contractFailureDeadLettersImmediately() {
        claim(1); when(client.send(envelope)).thenReturn(OutboxClient.DeliveryResult.CONTRACT_FAILURE);
        dispatcher(5).dispatchOnce(now);
        verify(repository).markDeadLetter(eventId, lease, "CONTRACT_FAILURE");
    }

    @Test void maximumAttemptDeadLettersAmbiguousFailure() {
        claim(5); when(client.send(envelope)).thenReturn(OutboxClient.DeliveryResult.AMBIGUOUS_FAILURE);
        dispatcher(5).dispatchOnce(now);
        verify(repository).markDeadLetter(eventId, lease, "MAX_ATTEMPTS");
        verify(repository, never()).markRetry(any(), any(), any(), any());
    }
}
