package com.cholopay.bankcore;

import com.cholopay.bankcore.service.InterNetworkTransferService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;

import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

@Tag("integration")
@SpringBootTest
class InterNetworkTransferIntegrationTest {
    @Autowired JdbcTemplate jdbc;
    @Autowired InterNetworkTransferService transfers;
    private UUID source, destination, sourceParticipant, destinationParticipant, sourceNetwork, destinationNetwork;

    @BeforeEach
    void reset() {
        jdbc.update("DELETE FROM inter_network_outbox");
        jdbc.update("DELETE FROM journal_entry");
        jdbc.update("DELETE FROM balance_reservation");
        jdbc.update("DELETE FROM journal");
        jdbc.update("DELETE FROM ledger_account");
        source = UUID.randomUUID(); destination = UUID.randomUUID();
        sourceParticipant = UUID.randomUUID(); destinationParticipant = UUID.randomUUID();
        sourceNetwork = UUID.randomUUID(); destinationNetwork = UUID.randomUUID();
        jdbc.update("INSERT INTO ledger_account(id, owner_ref, currency, posted_balance_minor) VALUES (?, 'synthetic-source', 'USD', 1000)", source);
        jdbc.update("INSERT INTO ledger_account(id, owner_ref, currency, posted_balance_minor) VALUES (?, 'synthetic-destination', 'USD', 0)", destination);
    }

    @Test
    void commitsJournalAndOutboxAtomically() {
        var result = transfers.order(request("atomic", 250));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM inter_network_outbox", Integer.class));
        assertEquals(result.transferId(), jdbc.queryForObject("SELECT aggregate_id FROM inter_network_outbox", UUID.class));
    }

    @Test
    void identicalReplayReturnsStableIdentitiesWithoutSideEffects() {
        var original = request("replay", 250);
        var first = transfers.order(original);
        var retry = transfers.order(original);
        assertEquals(first, retry);
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM inter_network_outbox", Integer.class));
    }

    @Test
    void conflictingPayloadHasNoSideEffects() {
        transfers.order(request("conflict", 250));
        assertThrows(RuntimeException.class, () -> transfers.order(request("conflict", 251)));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM inter_network_outbox", Integer.class));
        assertEquals(750, balance(source));
    }

    @Test
    void persistsEnvelopeIdentityAndFullPayloadHash() {
        var result = transfers.order(request("hash", 125));
        assertEquals(result.eventId(), jdbc.queryForObject("SELECT event_id FROM inter_network_outbox", UUID.class));
        assertEquals(result.payloadHash(), jdbc.queryForObject("SELECT payload_hash FROM inter_network_outbox", String.class));
        assertEquals("TRANSFER_REQUESTED", jdbc.queryForObject("SELECT event_name FROM inter_network_outbox", String.class));
        assertEquals("1.0", jdbc.queryForObject("SELECT schema_version FROM inter_network_outbox", String.class));
        assertEquals(1, jdbc.queryForObject("SELECT event_version FROM inter_network_outbox", Integer.class));
    }

    private InterNetworkTransferService.OrderRequest request(String key, long amount) {
        UUID correlation = UUID.randomUUID();
        return new InterNetworkTransferService.OrderRequest(source, destination, sourceParticipant, destinationParticipant,
                sourceNetwork, destinationNetwork, amount, "USD", key, correlation, UUID.randomUUID(), "trace-" + key, null);
    }

    private long balance(UUID account) {
        return jdbc.queryForObject("SELECT posted_balance_minor - reserved_balance_minor FROM ledger_account WHERE id = ?", Long.class, account);
    }
}
