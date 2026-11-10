package com.cholopay.bankcore.domain;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

public record Journal(UUID id, UUID transactionId, String idempotencyKey, UUID correlationId,
                      List<LedgerEntry> entries, Instant createdAt) {
    public Journal {
        if (entries == null || entries.size() < 2) throw new IllegalArgumentException("el diario requiere al menos dos asientos");
        long balance = entries.stream().mapToLong(LedgerEntry::signedAmount).sum();
        if (balance != 0) throw new IllegalArgumentException("el diario debe estar equilibrado");
        entries = List.copyOf(entries);
    }
}
