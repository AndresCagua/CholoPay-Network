package com.cholopay.bankcore.service;

import com.cholopay.bankcore.domain.Journal;
import com.cholopay.bankcore.domain.LedgerEntry;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.Comparator;
import java.util.List;
import java.util.UUID;

@Service
public class ReceiverCreditService {
    private final JdbcTemplate jdbc;
    public ReceiverCreditService(JdbcTemplate jdbc) { this.jdbc = jdbc; }

    @Transactional
    public Receipt apply(UUID deliveryId, UUID eventId, UUID transferId, UUID destinationAccountId, UUID clearingAccountId, long amountMinor) {
        if (amountMinor <= 0 || destinationAccountId.equals(clearingAccountId)) throw new IllegalArgumentException("crédito de receptor inválido");
        String hash = hash(deliveryId + ":" + eventId + ":" + transferId + ":" + destinationAccountId + ":" + clearingAccountId + ":" + amountMinor + ":USD");
        List<UUID> ordered = List.of(destinationAccountId, clearingAccountId).stream().sorted(Comparator.comparing(UUID::toString)).toList();
        jdbc.query("SELECT id FROM ledger_account WHERE id IN (?, ?) ORDER BY id FOR UPDATE", (rs, row) -> rs.getObject("id", UUID.class), ordered.get(0), ordered.get(1));
        Existing existing = jdbc.query("SELECT payload_hash,journal_id FROM receiver_credit_inbox WHERE delivery_id = ?", (rs, row) -> new Existing(rs.getString(1), rs.getObject(2, UUID.class)), deliveryId).stream().findFirst().orElse(null);
        if (existing != null) {
            if (!existing.payloadHash().equals(hash)) throw new IllegalStateException("conflicto en la carga útil de entrega al receptor");
            return new Receipt(deliveryId, eventId, transferId, existing.journalId(), "APPLIED");
        }
        long clearingBalance = jdbc.queryForObject("SELECT posted_balance_minor FROM ledger_account WHERE id = ?", Long.class, clearingAccountId);
        if (clearingBalance < amountMinor) throw new IllegalStateException("saldo de compensación insuficiente");
        UUID journalId = UUID.randomUUID();
        jdbc.update("UPDATE ledger_account SET posted_balance_minor = posted_balance_minor - ? WHERE id = ?", amountMinor, clearingAccountId);
        jdbc.update("UPDATE ledger_account SET posted_balance_minor = posted_balance_minor + ? WHERE id = ?", amountMinor, destinationAccountId);
        jdbc.update("INSERT INTO journal(id,transaction_id,idempotency_key,correlation_id,request_hash) VALUES (?, ?, ?, ?, ?)", journalId, transferId, "receiver:" + deliveryId, UUID.randomUUID(), hash);
        jdbc.update("INSERT INTO journal_entry(id,journal_id,account_id,amount_minor,direction,currency) VALUES (?, ?, ?, ?, 'DEBIT','USD'), (?, ?, ?, ?, 'CREDIT','USD')", UUID.randomUUID(), journalId, clearingAccountId, amountMinor, UUID.randomUUID(), journalId, destinationAccountId, amountMinor);
        jdbc.update("INSERT INTO receiver_credit_inbox(delivery_id,event_id,transfer_id,destination_account_id,clearing_account_id,amount_minor,currency,payload_hash,journal_id) VALUES (?, ?, ?, ?, ?, ?, 'USD', ?, ?)", deliveryId, eventId, transferId, destinationAccountId, clearingAccountId, amountMinor, hash, journalId);
        return new Receipt(deliveryId, eventId, transferId, journalId, "APPLIED");
    }

    private static String hash(String value) { try { return java.util.HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(value.getBytes(StandardCharsets.UTF_8))); } catch (Exception e) { throw new IllegalStateException(e); } }
    private record Existing(String payloadHash, UUID journalId) {}
    public record Receipt(UUID deliveryId, UUID eventId, UUID transferId, UUID journalId, String state) {}
}
