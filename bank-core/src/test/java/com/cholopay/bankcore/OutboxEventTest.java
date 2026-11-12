package com.cholopay.bankcore;

import com.cholopay.bankcore.domain.OutboxEvent;
import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

class OutboxEventTest {
    @Test
    void serializesPayloadWithNetworkContractNames() {
        UUID transferId = UUID.randomUUID();
        OutboxEvent event = new OutboxEvent(
                UUID.randomUUID(), "TRANSFER", transferId, "TRANSFER_REQUESTED", "1.0", 1,
                Instant.parse("2026-11-10T00:00:00Z"), UUID.randomUUID(), UUID.randomUUID(),
                "outbox-test", "trace-test",
                new OutboxEvent.Payload(transferId, UUID.randomUUID(), UUID.randomUUID(), 250,
                        "USD", "INTER_NETWORK", UUID.randomUUID(), UUID.randomUUID()), "");

        String json = event.canonicalJson();
        assertTrue(json.contains("\"transfer_id\""));
        assertTrue(json.contains("\"amount_minor\""));
        assertTrue(json.contains("\"source_network_id\""));
        assertFalse(json.contains("\"transferId\""));
        assertFalse(json.contains("\"amountMinor\""));
    }

    @Test
    void preservesHashForBackfilledEnvelope() {
        UUID transferId = UUID.randomUUID();
        OutboxEvent event = new OutboxEvent(
                UUID.randomUUID(), "TRANSFER", transferId, "TRANSFER_REQUESTED", "1.0", 1,
                Instant.parse("2026-11-10T00:00:00Z"), UUID.randomUUID(), UUID.randomUUID(),
                "backfilled", "trace-backfilled",
                new OutboxEvent.Payload(transferId, UUID.randomUUID(), UUID.randomUUID(), 250,
                        "USD", "INTER_NETWORK", UUID.randomUUID(), UUID.randomUUID()), "legacy-hash");

        assertEquals("legacy-hash", event.payloadHash());
    }
}
