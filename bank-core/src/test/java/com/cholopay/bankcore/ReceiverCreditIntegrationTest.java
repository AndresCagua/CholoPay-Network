package com.cholopay.bankcore;

import com.cholopay.bankcore.service.ReceiverCreditService;
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
class ReceiverCreditIntegrationTest {
    @Autowired JdbcTemplate jdbc;
    @Autowired ReceiverCreditService credits;
    private UUID clearing;
    private UUID destination;

    @BeforeEach
    void resetSyntheticLedger() {
        jdbc.update("DELETE FROM receiver_credit_inbox");
        jdbc.update("DELETE FROM journal_entry");
        jdbc.update("DELETE FROM journal");
        jdbc.update("DELETE FROM ledger_account");
        clearing = UUID.randomUUID(); destination = UUID.randomUUID();
        jdbc.update("INSERT INTO ledger_account(id,owner_ref,currency,posted_balance_minor) VALUES (?, 'synthetic-network-clearing', 'USD', 1000)", clearing);
        jdbc.update("INSERT INTO ledger_account(id,owner_ref,currency,posted_balance_minor) VALUES (?, 'synthetic-receiver', 'USD', 0)", destination);
    }

    @Test
    void appliesBalancedCreditAndReplaysSameDelivery() {
        UUID delivery = UUID.randomUUID(), event = UUID.randomUUID(), transfer = UUID.randomUUID();
        var first = credits.apply(delivery, event, transfer, destination, clearing, 200);
        var replay = credits.apply(delivery, event, transfer, destination, clearing, 200);
        assertEquals(first.journalId(), replay.journalId());
        assertEquals(800, balance(clearing));
        assertEquals(200, balance(destination));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class));
        assertEquals(0, jdbc.queryForObject("SELECT COALESCE(SUM(CASE WHEN direction='CREDIT' THEN amount_minor ELSE -amount_minor END),0) FROM journal_entry", Long.class));
    }

    @Test
    void rejectsConflictingDeliveryPayloadAndInsufficientClearing() {
        UUID delivery = UUID.randomUUID(), event = UUID.randomUUID(), transfer = UUID.randomUUID();
        credits.apply(delivery, event, transfer, destination, clearing, 200);
        assertThrows(IllegalStateException.class, () -> credits.apply(delivery, UUID.randomUUID(), transfer, destination, clearing, 200));
        assertThrows(IllegalStateException.class, () -> credits.apply(UUID.randomUUID(), UUID.randomUUID(), UUID.randomUUID(), destination, clearing, 900));
        assertEquals(800, balance(clearing));
        assertEquals(200, balance(destination));
    }

    private long balance(UUID id) { return jdbc.queryForObject("SELECT posted_balance_minor - reserved_balance_minor FROM ledger_account WHERE id = ?", Long.class, id); }
}
