package com.cholopay.bankcore;

import com.cholopay.bankcore.service.JdbcInternalTransferService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;

import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;

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

    private long balance(UUID accountId) {
        return jdbc.queryForObject("SELECT posted_balance_minor - reserved_balance_minor FROM ledger_account WHERE id = ?", Long.class, accountId);
    }
}
