package com.cholopay.bankcore.service;

import com.cholopay.bankcore.service.PublicAccountService.*;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;

import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Base64;
import java.util.List;
import java.util.UUID;

/** Proyección pública de solo lectura, limitada a la cuenta del cliente autenticado. */
@Service
public class JdbcPublicLedgerService {
    private final JdbcTemplate jdbc;
    private final ObjectMapper json = new ObjectMapper().findAndRegisterModules();

    public JdbcPublicLedgerService(JdbcTemplate jdbc) { this.jdbc = jdbc; }

    public PublicAccountService.PublicLedgerPage<PublicAccountService.PublicMovement> movements(String participantId, String customerToken, UUID accountId, int limit, String cursor) {
        verifyAccount(participantId, customerToken, accountId);
        Cursor after = decode(cursor, accountId, "movements");
        List<PublicMovement> rows = jdbc.query("SELECT e.id, e.journal_id, e.account_id, e.amount_minor, e.direction, e.currency, e.created_at " +
                        "FROM journal_entry e JOIN ledger_account a ON a.id=e.account_id JOIN public_customer c ON c.id=a.customer_id " +
                        "WHERE e.account_id=? AND c.participant_id=? AND c.api_token_hash=? " +
                        (after == null ? "" : "AND (e.created_at,e.id) < (?,?) ") +
                        "ORDER BY e.created_at DESC, e.id DESC LIMIT ?",
                (rs, row) -> new PublicMovement(rs.getObject(1, UUID.class), rs.getObject(2, UUID.class), rs.getObject(3, UUID.class),
                        rs.getLong(4), rs.getString(5), rs.getString(6), rs.getTimestamp(7).toInstant()),
                parameters(accountId, participantId, tokenHash(customerToken), after, limit));
        return page(rows, limit, row -> new Cursor(row.createdAt(), row.id(), accountId, "movements"));
    }

    public PublicAccountService.PublicLedgerPage<PublicAccountService.PublicJournal> journals(String participantId, String customerToken, UUID accountId, int limit, String cursor) {
        verifyAccount(participantId, customerToken, accountId);
        Cursor after = decode(cursor, accountId, "journals");
        List<PublicJournal> rows = jdbc.query("SELECT j.id,j.transaction_id,j.idempotency_key,j.correlation_id,j.created_at " +
                        "FROM journal j JOIN journal_entry e ON e.journal_id=j.id JOIN ledger_account a ON a.id=e.account_id JOIN public_customer c ON c.id=a.customer_id " +
                        "WHERE e.account_id=? AND c.participant_id=? AND c.api_token_hash=? " +
                        (after == null ? "" : "AND (j.created_at,j.id) < (?,?) ") +
                        "GROUP BY j.id,j.transaction_id,j.idempotency_key,j.correlation_id,j.created_at " +
                        "ORDER BY j.created_at DESC,j.id DESC LIMIT ?",
                (rs, row) -> readJournal(rs.getObject(1, UUID.class), rs.getObject(2, UUID.class), rs.getString(3), rs.getObject(4, UUID.class), rs.getTimestamp(5).toInstant()),
                parameters(accountId, participantId, tokenHash(customerToken), after, limit));
        return page(rows, limit, row -> new Cursor(row.createdAt(), row.id(), accountId, "journals"));
    }

    private PublicJournal readJournal(UUID id, UUID transactionId, String key, UUID correlationId, Instant createdAt) {
        List<PublicJournalEntry> entries = jdbc.query("SELECT account_id,amount_minor,direction,currency FROM journal_entry WHERE journal_id=? ORDER BY id",
                (rs, row) -> new PublicJournalEntry(rs.getObject(1, UUID.class), rs.getLong(2), rs.getString(3), rs.getString(4)), id);
        return new PublicJournal(id, transactionId, key, correlationId, entries, createdAt);
    }

    private void verifyAccount(String participantId, String token, UUID accountId) {
        if (token == null || token.isBlank()) throw new SecurityException("invalid customer token");
        Integer count = jdbc.queryForObject("SELECT COUNT(*) FROM ledger_account a JOIN public_customer c ON c.id=a.customer_id WHERE a.id=? AND c.participant_id=? AND c.api_token_hash=?", Integer.class, accountId, participantId, tokenHash(token));
        if (count == null || count == 0) throw new AccountNotAccessibleException();
    }

    private String tokenHash(String token) { return JdbcPublicAccountService.hashToken(token); }
    private Object[] parameters(UUID accountId, String participant, String tokenHash, Cursor after, int limit) {
        List<Object> p = new ArrayList<>(List.of(accountId, participant, tokenHash));
        if (after != null) { p.add(java.sql.Timestamp.from(after.createdAt)); p.add(after.id); }
        p.add(limit + 1);
        return p.toArray();
    }
    private <T> PublicLedgerPage<T> page(List<T> rows, int limit, java.util.function.Function<T, Cursor> cursor) {
        if (rows.size() <= limit) return new PublicLedgerPage<>(rows, null);
        T last = rows.get(limit - 1);
        return new PublicLedgerPage<>(rows.subList(0, limit), encode(cursor.apply(last)));
    }
    private Cursor decode(String value, UUID accountId, String kind) {
        if (value == null || value.isBlank()) return null;
        try {
            JsonNode n = json.readTree(new String(Base64.getUrlDecoder().decode(value), StandardCharsets.UTF_8));
            Cursor c = new Cursor(Instant.parse(n.path("created_at").asText()), UUID.fromString(n.path("id").asText()), UUID.fromString(n.path("account_id").asText()), n.path("scope").asText());
            if (!accountId.equals(c.accountId) || !kind.equals(c.scope)) throw new InvalidCursorException();
            return c;
        } catch (Exception e) { throw new InvalidCursorException(); }
    }
    private String encode(Cursor c) {
        try { return Base64.getUrlEncoder().withoutPadding().encodeToString(json.writeValueAsBytes(java.util.Map.of("created_at", c.createdAt, "id", c.id, "account_id", c.accountId, "scope", c.scope))); }
        catch (Exception e) { throw new IllegalStateException(e); }
    }
    public static class AccountNotAccessibleException extends RuntimeException {}
    public static class InvalidCursorException extends RuntimeException {}
    private record Cursor(Instant createdAt, UUID id, UUID accountId, String scope) {}
}
