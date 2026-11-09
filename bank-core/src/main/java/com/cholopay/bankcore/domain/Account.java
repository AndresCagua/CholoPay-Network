package com.cholopay.bankcore.domain;

import java.util.UUID;

public final class Account {
    private final UUID id;
    private final String ownerRef;
    private long postedMinor;
    private long reservedMinor;

    public Account(UUID id, String ownerRef, long openingMinor) {
        if (openingMinor < 0) throw new IllegalArgumentException("opening balance must not be negative");
        this.id = id; this.ownerRef = ownerRef; this.postedMinor = openingMinor;
    }
    public UUID id() { return id; }
    public String ownerRef() { return ownerRef; }
    public long postedMinor() { return postedMinor; }
    public long reservedMinor() { return reservedMinor; }
    public long availableMinor() { return postedMinor - reservedMinor; }
    public void reserve(long amount) {
        if (amount <= 0 || amount > availableMinor()) throw new IllegalStateException("insufficient available funds");
        reservedMinor += amount;
    }
    public void release(long amount) { reservedMinor = checkedSub(reservedMinor, amount); }
    public void debitReserved(long amount) { reservedMinor = checkedSub(reservedMinor, amount); postedMinor = checkedSub(postedMinor, amount); }
    public void credit(long amount) { if (amount <= 0) throw new IllegalArgumentException("credit must be positive"); postedMinor = Math.addExact(postedMinor, amount); }
    private static long checkedSub(long current, long amount) { if (amount < 0 || amount > current) throw new IllegalStateException("invalid reservation transition"); return current - amount; }
}
