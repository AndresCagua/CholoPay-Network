package com.cholopay.bankcore;

import com.cholopay.bankcore.domain.Journal;
import com.cholopay.bankcore.domain.LedgerEntry;
import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertThrows;

class JournalTest {
    @Test
    void rejectsUnbalancedJournal() {
        assertThrows(IllegalArgumentException.class, () -> new Journal(UUID.randomUUID(), UUID.randomUUID(), "k", UUID.randomUUID(),
                List.of(new LedgerEntry(UUID.randomUUID(), 10, LedgerEntry.Direction.DEBIT, "USD")), Instant.now()));
    }
}
