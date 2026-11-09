package com.cholopay.bankcore.service;

import com.cholopay.bankcore.domain.Journal;
import com.cholopay.bankcore.domain.LedgerEntry;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Instant;
import java.util.List;
import java.util.UUID;

@Service
public class JdbcInternalTransferService {
    private final JdbcTemplate jdbc;

    public JdbcInternalTransferService(JdbcTemplate jdbc) { this.jdbc = jdbc; }

    @Transactional
    public Journal transfer(UUID sourceId, UUID destinationId, long amountMinor, String idempotencyKey, UUID correlationId) {
        if (amountMinor <= 0) throw new IllegalArgumentException("amount_minor must be positive");
        if (sourceId.equals(destinationId)) throw new IllegalArgumentException("source and destination must differ");
        String requestHash = hash(sourceId + ":" + destinationId + ":" + amountMinor + ":USD");

        List<ExistingJournal> previous = jdbc.query(
                "SELECT transaction_id, request_hash FROM journal WHERE idempotency_key = ?",
                (rs, row) -> new ExistingJournal(rs.getObject("transaction_id", UUID.class), rs.getString("request_hash")), idempotencyKey);
        if (!previous.isEmpty()) {
            if (!requestHash.equals(previous.getFirst().requestHash())) throw new InternalTransferService.IdempotencyConflictException();
            return readJournal(previous.getFirst().transactionId());
        }

        List<UUID> ordered = sourceId.toString().compareTo(destinationId.toString()) < 0
                ? List.of(sourceId, destinationId) : List.of(destinationId, sourceId);
        jdbc.query("SELECT id FROM ledger_account WHERE id IN (?, ?) ORDER BY id FOR UPDATE",
                (org.springframework.jdbc.core.RowMapper<UUID>) (rs, row) -> rs.getObject("id", UUID.class), ordered.get(0), ordered.get(1));
        Balance source = balance(sourceId);
        Balance destination = balance(destinationId);
        if (source == null || destination == null) throw new IllegalArgumentException("account not found");
        if (source.postedMinor() - source.reservedMinor() < amountMinor) throw new IllegalStateException("insufficient available funds");

        UUID transactionId = UUID.randomUUID();
        UUID journalId = UUID.randomUUID();
        int debited = jdbc.update("UPDATE ledger_account SET posted_balance_minor = posted_balance_minor - ? WHERE id = ? AND posted_balance_minor - reserved_balance_minor >= ?",
                amountMinor, sourceId, amountMinor);
        if (debited != 1) throw new IllegalStateException("insufficient available funds");
        jdbc.update("UPDATE ledger_account SET posted_balance_minor = posted_balance_minor + ? WHERE id = ?", amountMinor, destinationId);
        jdbc.update("INSERT INTO balance_reservation(id, account_id, transfer_id, amount_minor, status) VALUES (?, ?, ?, ?, 'CAPTURED')",
                UUID.randomUUID(), sourceId, transactionId, amountMinor);
        jdbc.update("INSERT INTO journal(id, transaction_id, idempotency_key, correlation_id, request_hash) VALUES (?, ?, ?, ?, ?)",
                journalId, transactionId, idempotencyKey, correlationId, requestHash);
        jdbc.update("INSERT INTO journal_entry(id, journal_id, account_id, amount_minor, direction, currency) VALUES (?, ?, ?, ?, 'DEBIT', 'USD'), (?, ?, ?, ?, 'CREDIT', 'USD')",
                UUID.randomUUID(), journalId, sourceId, amountMinor, UUID.randomUUID(), journalId, destinationId, amountMinor);
        return new Journal(journalId, transactionId, idempotencyKey, correlationId,
                List.of(new LedgerEntry(sourceId, amountMinor, LedgerEntry.Direction.DEBIT, "USD"),
                        new LedgerEntry(destinationId, amountMinor, LedgerEntry.Direction.CREDIT, "USD")), Instant.now());
    }

    private Balance balance(UUID id) {
        return jdbc.query("SELECT posted_balance_minor, reserved_balance_minor FROM ledger_account WHERE id = ?",
                rs -> rs.next() ? new Balance(rs.getLong(1), rs.getLong(2)) : null, id);
    }

    private Journal readJournal(UUID transactionId) {
        JournalHeader header = jdbc.queryForObject("SELECT id, idempotency_key, correlation_id FROM journal WHERE transaction_id = ?",
                (rs, row) -> new JournalHeader(rs.getObject("id", UUID.class), rs.getString("idempotency_key"), rs.getObject("correlation_id", UUID.class)), transactionId);
        List<LedgerEntry> entries = jdbc.query("SELECT account_id, amount_minor, direction, currency FROM journal_entry WHERE journal_id = ? ORDER BY id",
                (rs, row) -> new LedgerEntry(rs.getObject("account_id", UUID.class), rs.getLong("amount_minor"),
                        LedgerEntry.Direction.valueOf(rs.getString("direction")), rs.getString("currency")), header.id());
        return new Journal(header.id(), transactionId, header.idempotencyKey(), header.correlationId(), entries, Instant.now());
    }

    private static String hash(String value) {
        try {
            byte[] digest = MessageDigest.getInstance("SHA-256").digest(value.getBytes(StandardCharsets.UTF_8));
            return java.util.HexFormat.of().formatHex(digest);
        } catch (NoSuchAlgorithmException e) { throw new IllegalStateException(e); }
    }

    private record ExistingJournal(UUID transactionId, String requestHash) {}
    private record JournalHeader(UUID id, String idempotencyKey, UUID correlationId) {}
    private record Balance(long postedMinor, long reservedMinor) {}
}
