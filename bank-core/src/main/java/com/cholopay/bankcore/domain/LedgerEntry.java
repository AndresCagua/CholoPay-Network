package com.cholopay.bankcore.domain;

import java.util.UUID;

public record LedgerEntry(UUID accountId, long amountMinor, Direction direction, String currency) {
    public enum Direction { DEBIT, CREDIT }
    public LedgerEntry {
        if (amountMinor <= 0) throw new IllegalArgumentException("ledger amount must be positive");
        if (!"USD".equals(currency)) throw new IllegalArgumentException("currency must be USD");
    }
    public long signedAmount() { return direction == Direction.CREDIT ? amountMinor : -amountMinor; }
}
