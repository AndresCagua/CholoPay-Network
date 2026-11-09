package com.cholopay.bankcore.service;

import com.cholopay.bankcore.domain.Account;
import com.cholopay.bankcore.domain.Journal;
import com.cholopay.bankcore.domain.LedgerEntry;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

@Service
public class InternalTransferService {
    private final Map<UUID, Account> accounts = new ConcurrentHashMap<>();
    private final Map<String, Journal> completedByIdempotency = new ConcurrentHashMap<>();
    private final List<Journal> journals = new ArrayList<>();

    public Account openAccount(String ownerRef, long openingMinor) {
        Account account = new Account(UUID.randomUUID(), ownerRef, openingMinor);
        accounts.put(account.id(), account);
        return account;
    }

    public synchronized Journal transfer(UUID sourceId, UUID destinationId, long amountMinor, String idempotencyKey, UUID correlationId) {
        if (amountMinor <= 0) throw new IllegalArgumentException("amount_minor must be positive");
        if (sourceId.equals(destinationId)) throw new IllegalArgumentException("source and destination must differ");
        Journal existing = completedByIdempotency.get(idempotencyKey);
        if (existing != null) {
            if (existing.entries().get(0).amountMinor() != amountMinor) throw new IdempotencyConflictException();
            return existing;
        }
        List<Account> locked = List.of(accounts.get(sourceId), accounts.get(destinationId)).stream()
                .sorted(Comparator.comparing(a -> a.id().toString())).toList();
        if (locked.contains(null)) throw new IllegalArgumentException("account not found");
        for (Account ignored : locked) { /* synchronized service provides deterministic critical section */ }
        Account source = accounts.get(sourceId), destination = accounts.get(destinationId);
        source.reserve(amountMinor);
        source.debitReserved(amountMinor);
        destination.credit(amountMinor);
        Journal journal = new Journal(UUID.randomUUID(), UUID.randomUUID(), idempotencyKey, correlationId,
                List.of(new LedgerEntry(sourceId, amountMinor, LedgerEntry.Direction.DEBIT, "USD"),
                        new LedgerEntry(destinationId, amountMinor, LedgerEntry.Direction.CREDIT, "USD")), Instant.now());
        completedByIdempotency.put(idempotencyKey, journal);
        journals.add(journal);
        return journal;
    }

    public Account account(UUID id) { return accounts.get(id); }
    public List<Journal> journals() { return List.copyOf(journals); }

    public static final class IdempotencyConflictException extends RuntimeException {
        public IdempotencyConflictException() { super("idempotency key was already used with a different request"); }
    }
}
