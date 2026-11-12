package com.cholopay.bankcore.service;

import com.cholopay.bankcore.domain.Journal;
import com.cholopay.bankcore.domain.LedgerEntry;
import com.cholopay.bankcore.domain.OutboxEvent;
import com.fasterxml.jackson.databind.ObjectMapper;
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
    private final ObjectMapper json = new ObjectMapper().findAndRegisterModules();

    public JdbcInternalTransferService(JdbcTemplate jdbc) { this.jdbc = jdbc; }

    @Transactional
    public Journal transfer(UUID sourceId, UUID destinationId, long amountMinor, String idempotencyKey, UUID correlationId) {
        if (amountMinor <= 0) throw new IllegalArgumentException("amount_minor debe ser positivo");
        if (sourceId.equals(destinationId)) throw new IllegalArgumentException("el origen y el destino deben ser distintos");
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
        if (source == null || destination == null) throw new IllegalArgumentException("cuenta no encontrada");
        if (source.postedMinor() - source.reservedMinor() < amountMinor) throw new IllegalStateException("fondos disponibles insuficientes");

        UUID transactionId = UUID.randomUUID();
        UUID journalId = UUID.randomUUID();
        int debited = jdbc.update("UPDATE ledger_account SET posted_balance_minor = posted_balance_minor - ? WHERE id = ? AND posted_balance_minor - reserved_balance_minor >= ?",
                amountMinor, sourceId, amountMinor);
        if (debited != 1) throw new IllegalStateException("fondos disponibles insuficientes");
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

    /** Executes the local journal and its inter-network outbox insert in one transaction. */
    @Transactional
    public InterNetworkTransferService.OrderResult transferInterNetwork(InterNetworkTransferService.OrderRequest request) {
        if (request.amountMinor() <= 0) throw new IllegalArgumentException("amount_minor debe ser positivo");
        if (request.sourceAccountId().equals(request.destinationAccountId())) throw new IllegalArgumentException("el origen y el destino deben ser distintos");
        String requestHash = hash(request.sourceAccountId() + ":" + request.destinationAccountId() + ":" + request.sourceParticipantId() + ":" + request.destinationParticipantId() + ":" + request.amountMinor() + ":" + request.currency() + ":" + request.sourceNetworkId() + ":" + request.destinationNetworkId() + ":" + request.correlationId() + ":" + request.causationId() + ":" + request.traceId());
        ExistingJournal existing = jdbc.query("SELECT transaction_id, request_hash FROM journal WHERE idempotency_key = ? FOR UPDATE", (rs, row) -> new ExistingJournal(rs.getObject("transaction_id", UUID.class), rs.getString("request_hash")), request.idempotencyKey()).stream().findFirst().orElse(null);
        if (existing != null) {
            if (!requestHash.equals(existing.requestHash())) throw new InternalTransferService.IdempotencyConflictException();
            OutboxEvent event = readOutbox(existing.transactionId());
            return new InterNetworkTransferService.OrderResult(existing.transactionId(), event.eventId(), event.payloadHash(), "PENDING");
        }
        List<UUID> ordered = List.of(request.sourceAccountId(), request.destinationAccountId()).stream().sorted().toList();
        jdbc.query("SELECT id FROM ledger_account WHERE id IN (?, ?) ORDER BY id FOR UPDATE", (org.springframework.jdbc.core.RowMapper<UUID>) (rs, row) -> rs.getObject("id", UUID.class), ordered.get(0), ordered.get(1));
        Balance source = balance(request.sourceAccountId());
        Balance destination = balance(request.destinationAccountId());
        if (source == null || destination == null) throw new IllegalArgumentException("cuenta no encontrada");
        if (source.postedMinor() - source.reservedMinor() < request.amountMinor()) throw new IllegalStateException("fondos disponibles insuficientes");
        UUID transferId = UUID.randomUUID();
        UUID journalId = UUID.randomUUID();
        jdbc.update("UPDATE ledger_account SET posted_balance_minor = posted_balance_minor - ? WHERE id = ? AND posted_balance_minor - reserved_balance_minor >= ?", request.amountMinor(), request.sourceAccountId(), request.amountMinor());
        jdbc.update("UPDATE ledger_account SET posted_balance_minor = posted_balance_minor + ? WHERE id = ?", request.amountMinor(), request.destinationAccountId());
        jdbc.update("INSERT INTO balance_reservation(id, account_id, transfer_id, amount_minor, status) VALUES (?, ?, ?, ?, 'CAPTURED')", UUID.randomUUID(), request.sourceAccountId(), transferId, request.amountMinor());
        jdbc.update("INSERT INTO journal(id, transaction_id, idempotency_key, correlation_id, request_hash) VALUES (?, ?, ?, ?, ?)", journalId, transferId, request.idempotencyKey(), request.correlationId(), requestHash);
        jdbc.update("INSERT INTO journal_entry(id, journal_id, account_id, amount_minor, direction, currency) VALUES (?, ?, ?, ?, 'DEBIT', ?), (?, ?, ?, ?, 'CREDIT', ?)", UUID.randomUUID(), journalId, request.sourceAccountId(), request.amountMinor(), request.currency(), UUID.randomUUID(), journalId, request.destinationAccountId(), request.amountMinor(), request.currency());
        OutboxEvent event = new OutboxEvent(UUID.randomUUID(), "TRANSFER", transferId, "TRANSFER_REQUESTED", "1.0", 1, request.occurredAt(), request.correlationId(), request.causationId(), request.idempotencyKey(), request.traceId(), new OutboxEvent.Payload(transferId, request.sourceParticipantId(), request.destinationParticipantId(), request.amountMinor(), request.currency(), "INTER_NETWORK", request.sourceNetworkId(), request.destinationNetworkId()), "");
        try {
            jdbc.update("INSERT INTO inter_network_outbox(event_id, aggregate_type, aggregate_id, event_name, schema_version, event_version, occurred_at, correlation_id, causation_id, idempotency_key, trace_id, payload, serialized_envelope, payload_hash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?)", event.eventId(), event.aggregateType(), event.aggregateId(), event.eventName(), event.schemaVersion(), event.version(), event.occurredAt(), event.correlationId(), event.causationId(), event.idempotencyKey(), event.traceId(), event.canonicalJson(), event.canonicalJson(), event.payloadHash());
        } catch (RuntimeException failure) { throw failure; }
        return new InterNetworkTransferService.OrderResult(transferId, event.eventId(), event.payloadHash(), "PENDING");
    }

    private OutboxEvent readOutbox(UUID transferId) {
        return jdbc.queryForObject("SELECT event_id, aggregate_type, aggregate_id, event_name, schema_version, event_version, occurred_at, correlation_id, causation_id, idempotency_key, trace_id, payload_hash, payload->>'source_participant_id', payload->>'destination_participant_id', (payload->>'amount_minor')::bigint, payload->>'currency', payload->>'route_kind', payload->>'source_network_id', payload->>'destination_network_id' FROM inter_network_outbox WHERE aggregate_id = ?", (rs, row) -> new OutboxEvent(rs.getObject(1, UUID.class), rs.getString(2), rs.getObject(3, UUID.class), rs.getString(4), rs.getString(5), rs.getInt(6), rs.getTimestamp(7).toInstant(), rs.getObject(8, UUID.class), rs.getObject(9, UUID.class), rs.getString(10), rs.getString(11), new OutboxEvent.Payload(transferId, UUID.fromString(rs.getString(13)), UUID.fromString(rs.getString(14)), rs.getLong(15), rs.getString(16), rs.getString(17), UUID.fromString(rs.getString(18)), UUID.fromString(rs.getString(19))), rs.getString(12)), transferId);
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
