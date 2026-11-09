package com.cholopay.bankcore;

import com.cholopay.bankcore.domain.Journal;
import com.cholopay.bankcore.service.InternalTransferService;
import org.junit.jupiter.api.Test;

import java.util.UUID;
import java.util.concurrent.*;
import java.util.stream.IntStream;

import static org.junit.jupiter.api.Assertions.*;

class InternalTransferServiceTest {
    @Test
    void transfersWithIntegerCentsRemainBalancedAndIdempotent() {
        var service = new InternalTransferService();
        var source = service.openAccount("synthetic-alice", 10_00);
        var destination = service.openAccount("synthetic-bob", 0);
        UUID correlation = UUID.randomUUID();
        Journal first = service.transfer(source.id(), destination.id(), 250, "transfer-1", correlation);
        Journal retry = service.transfer(source.id(), destination.id(), 250, "transfer-1", correlation);
        assertSame(first, retry);
        assertEquals(750, source.availableMinor());
        assertEquals(250, destination.availableMinor());
        assertEquals(0, first.entries().stream().mapToLong(e -> e.signedAmount()).sum());
        assertEquals(1, service.journals().size());
    }

    @Test
    void concurrentTransfersCannotOverdraw() throws Exception {
        var service = new InternalTransferService();
        var source = service.openAccount("synthetic-alice", 1_000);
        var destination = service.openAccount("synthetic-bob", 0);
        ExecutorService pool = Executors.newFixedThreadPool(8);
        try {
            var results = IntStream.range(0, 100).mapToObj(i -> pool.submit(() -> {
                try { service.transfer(source.id(), destination.id(), 100, "unique-" + i, UUID.randomUUID()); return true; }
                catch (IllegalStateException e) { return false; }
            })).toList();
            long successful = 0;
            for (var result : results) successful += result.get() ? 1 : 0;
            assertEquals(10, successful);
            assertEquals(0, source.availableMinor());
            assertEquals(1_000, destination.availableMinor());
            assertTrue(source.availableMinor() >= 0);
        } finally { pool.shutdownNow(); }
    }

    @Test
    void differentPayloadUnderSameKeyIsRejected() {
        var service = new InternalTransferService();
        var source = service.openAccount("synthetic-alice", 1_000);
        var destination = service.openAccount("synthetic-bob", 0);
        service.transfer(source.id(), destination.id(), 100, "same-key", UUID.randomUUID());
        assertThrows(InternalTransferService.IdempotencyConflictException.class,
                () -> service.transfer(source.id(), destination.id(), 200, "same-key", UUID.randomUUID()));
        assertEquals(900, source.availableMinor());
        assertEquals(100, destination.availableMinor());
    }
}
