package com.cholopay.bankcore;

import com.cholopay.bankcore.service.JdbcInternalTransferService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;

import java.util.UUID;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.stream.IntStream;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag("integration")
@SpringBootTest
class JdbcInternalTransferIntegrationTest {
    @Autowired JdbcTemplate jdbc;
    @Autowired JdbcInternalTransferService transfers;

    private UUID source;
    private UUID destination;

    @BeforeEach
    void resetSyntheticAccounts() {
        jdbc.update("DELETE FROM journal_entry");
        jdbc.update("DELETE FROM balance_reservation");
        jdbc.update("DELETE FROM journal");
        jdbc.update("DELETE FROM ledger_account");
        source = UUID.randomUUID();
        destination = UUID.randomUUID();
        jdbc.update("INSERT INTO ledger_account(id, owner_ref, currency, posted_balance_minor) VALUES (?, 'synthetic-alice', 'USD', 1000)", source);
        jdbc.update("INSERT INTO ledger_account(id, owner_ref, currency, posted_balance_minor) VALUES (?, 'synthetic-bob', 'USD', 0)", destination);
    }

    @Test
    void persistsBalancedTransferAndReplaysIdempotently() {
        UUID correlation = UUID.randomUUID();
        var first = transfers.transfer(source, destination, 250, "integration-key", correlation);
        var retry = transfers.transfer(source, destination, 250, "integration-key", correlation);
        assertEquals(first.transactionId(), retry.transactionId());
        assertEquals(750, balance(source));
        assertEquals(250, balance(destination));
        assertEquals(0, jdbc.queryForObject("SELECT COALESCE(SUM(CASE WHEN direction = 'CREDIT' THEN amount_minor ELSE -amount_minor END), 0) FROM journal_entry", Long.class));
    }

    @Test
    void concurrentTransfersAgainstLimitedBalancePersistOnlyAffordableBalancedJournals() throws Exception {
        int transferCount = 100;
        int transferAmount = 100;
        ExecutorService pool = Executors.newFixedThreadPool(16);
        CountDownLatch start = new CountDownLatch(1);
        try {
            var attempts = IntStream.range(0, transferCount)
                    .mapToObj(index -> pool.submit(() -> {
                        start.await();
                        try {
                            transfers.transfer(source, destination, transferAmount,
                                    "concurrent-key-" + index, UUID.randomUUID());
                            return true;
                        } catch (IllegalStateException insufficientFunds) {
                            assertEquals("fondos disponibles insuficientes", insufficientFunds.getMessage());
                            return false;
                        }
                    }))
                    .toList();
            start.countDown();

            long successful = 0;
            for (Future<Boolean> attempt : attempts) {
                if (attempt.get(5, TimeUnit.SECONDS)) successful++;
            }

            assertEquals(10, successful);
            assertEquals(0, balance(source));
            assertEquals(1_000, balance(destination));
            assertTrue(balance(source) >= 0);
            assertEquals(10, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Long.class));
            assertEquals(10, jdbc.queryForObject("SELECT COUNT(DISTINCT transaction_id) FROM journal", Long.class));
            assertEquals(10, jdbc.queryForObject("SELECT COUNT(DISTINCT idempotency_key) FROM journal", Long.class));
            assertEquals(0, jdbc.queryForObject("""
                    SELECT COUNT(*) FROM (
                        SELECT idempotency_key
                        FROM journal
                        GROUP BY idempotency_key
                        HAVING COUNT(*) > 1
                    ) duplicate_journals
                    """, Long.class));
            assertEquals(0, jdbc.queryForObject("""
                    SELECT COUNT(*) FROM (
                        SELECT j.id
                        FROM journal j
                        JOIN journal_entry e ON e.journal_id = j.id
                        GROUP BY j.id
                        HAVING COUNT(*) <> 2
                            OR COUNT(DISTINCT e.currency) <> 1
                            OR MIN(e.currency) <> 'USD'
                            OR SUM(CASE WHEN e.direction = 'DEBIT' THEN e.amount_minor ELSE -e.amount_minor END) <> 0
                    ) unbalanced_journals
                    """, Long.class));
        } finally {
            pool.shutdownNow();
        }
    }

    private long balance(UUID accountId) {
        return jdbc.queryForObject("SELECT posted_balance_minor - reserved_balance_minor FROM ledger_account WHERE id = ?", Long.class, accountId);
    }
}
