package com.cholopay.bankcore;

import com.cholopay.bankcore.service.ReceiverCreditService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;

import java.util.ArrayList;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

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
    void appliesBalancedCreditAndReplaysSameDeliveryWithoutChangingLedger() {
        UUID delivery = UUID.randomUUID(), event = UUID.randomUUID(), transfer = UUID.randomUUID();
        var first = credits.apply(delivery, event, transfer, destination, clearing, 200);
        long clearingAfterFirst = balance(clearing);
        long destinationAfterFirst = balance(destination);
        int journalsAfterFirst = jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class);
        int entriesAfterFirst = jdbc.queryForObject("SELECT COUNT(*) FROM journal_entry", Integer.class);

        var replay = credits.apply(delivery, event, transfer, destination, clearing, 200);

        assertEquals(first.journalId(), replay.journalId());
        assertEquals(clearingAfterFirst, balance(clearing));
        assertEquals(destinationAfterFirst, balance(destination));
        assertEquals(journalsAfterFirst, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class));
        assertEquals(entriesAfterFirst, jdbc.queryForObject("SELECT COUNT(*) FROM journal_entry", Integer.class));
        assertEquals(0, jdbc.queryForObject("SELECT COALESCE(SUM(CASE WHEN direction='CREDIT' THEN amount_minor ELSE -amount_minor END),0) FROM journal_entry", Long.class));
    }

    @Test
    void concurrentDuplicateDeliveryProducesOneJournalAndTwoSuccessfulReceipts() throws Exception {
        UUID delivery = UUID.randomUUID(), event = UUID.randomUUID(), transfer = UUID.randomUUID();
        ExecutorService executor = Executors.newFixedThreadPool(2);
        CountDownLatch start = new CountDownLatch(1);
        try {
            List<Future<ReceiverCreditService.Receipt>> futures = new ArrayList<>();
            for (int i = 0; i < 2; i++) {
                futures.add(executor.submit(() -> {
                    start.await(5, TimeUnit.SECONDS);
                    return credits.apply(delivery, event, transfer, destination, clearing, 200);
                }));
            }
            start.countDown();
            var first = futures.get(0).get(5, TimeUnit.SECONDS);
            var second = futures.get(1).get(5, TimeUnit.SECONDS);
            assertEquals(first.journalId(), second.journalId());
            assertEquals(800, balance(clearing));
            assertEquals(200, balance(destination));
            assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class));
            assertEquals(2, jdbc.queryForObject("SELECT COUNT(*) FROM journal_entry", Integer.class));
        } finally {
            executor.shutdownNow();
        }
    }

    @Test
    void rejectsConflictingPayloadWithoutSideEffects() {
        UUID delivery = UUID.randomUUID(), event = UUID.randomUUID(), transfer = UUID.randomUUID();
        credits.apply(delivery, event, transfer, destination, clearing, 200);
        long clearingBefore = balance(clearing);
        long destinationBefore = balance(destination);
        int journalsBefore = jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class);
        int entriesBefore = jdbc.queryForObject("SELECT COUNT(*) FROM journal_entry", Integer.class);

        var error = assertThrows(IllegalStateException.class, () -> credits.apply(delivery, UUID.randomUUID(), transfer, destination, clearing, 200));

        assertEquals("conflicto en la carga útil de entrega al receptor", error.getMessage());
        assertEquals(clearingBefore, balance(clearing));
        assertEquals(destinationBefore, balance(destination));
        assertEquals(journalsBefore, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class));
        assertEquals(entriesBefore, jdbc.queryForObject("SELECT COUNT(*) FROM journal_entry", Integer.class));
    }

    @Test
    void rejectsInsufficientClearingBalanceDeterministicallyWithoutSideEffects() {
        UUID delivery = UUID.randomUUID(), event = UUID.randomUUID(), transfer = UUID.randomUUID();
        var error = assertThrows(IllegalStateException.class, () -> credits.apply(delivery, event, transfer, destination, clearing, 1001));

        assertEquals("saldo de compensación insuficiente", error.getMessage());
        assertEquals(1000, balance(clearing));
        assertEquals(0, balance(destination));
        assertEquals(0, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class));
        assertEquals(0, jdbc.queryForObject("SELECT COUNT(*) FROM receiver_credit_inbox", Integer.class));
    }

    private long balance(UUID id) { return jdbc.queryForObject("SELECT posted_balance_minor - reserved_balance_minor FROM ledger_account WHERE id = ?", Long.class, id); }
}
