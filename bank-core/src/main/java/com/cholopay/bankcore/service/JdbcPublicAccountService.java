package com.cholopay.bankcore.service;

import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.List;
import java.util.UUID;

/** Persistencia JDBC del perímetro público; cada consulta exige participante y cliente. */
@Service
public class JdbcPublicAccountService implements PublicAccountService {
    private final JdbcTemplate jdbc;

    public JdbcPublicAccountService(JdbcTemplate jdbc) { this.jdbc = jdbc; }

    @Override
    @Transactional
    public OnboardedCustomer onboard(String participantId, String syntheticReference) {
        if (participantId == null || participantId.isBlank()) throw new IllegalStateException("el participante bancario no está configurado");
        if (syntheticReference == null || syntheticReference.isBlank() || syntheticReference.length() > 160) {
            throw new IllegalArgumentException("la referencia sintética del cliente es obligatoria");
        }
        UUID customerId = UUID.randomUUID();
        String token = UUID.randomUUID().toString();
        try {
            jdbc.update("INSERT INTO public_customer(id, participant_id, synthetic_reference, api_token_hash) VALUES (?, ?, ?, ?)",
                    customerId, participantId, syntheticReference, hashToken(token));
        } catch (org.springframework.dao.DuplicateKeyException duplicate) {
            throw new IllegalArgumentException("la referencia sintética del cliente ya existe");
        }
        UUID accountId = UUID.randomUUID();
        jdbc.update("INSERT INTO ledger_account(id, owner_ref, currency, posted_balance_minor, reserved_balance_minor, participant_id, customer_id) VALUES (?, ?, 'USD', 0, 0, ?, ?)",
                accountId, "customer:" + customerId, participantId, customerId);
        return new OnboardedCustomer(customerId, syntheticReference, token,
                List.of(new PublicAccount(accountId, "USD", 0, 0)));
    }

    @Override
    public List<PublicAccount> accounts(String participantId, String customerToken) {
        String tokenHash = verifiedToken(customerToken);
        return jdbc.query("SELECT a.id, a.currency, a.posted_balance_minor, a.posted_balance_minor - a.reserved_balance_minor " +
                        "FROM ledger_account a JOIN public_customer c ON c.id = a.customer_id " +
                        "WHERE c.participant_id = ? AND c.api_token_hash = ? ORDER BY a.id",
                (rs, row) -> new PublicAccount(rs.getObject(1, UUID.class), rs.getString(2), rs.getLong(3), rs.getLong(4)), participantId, tokenHash);
    }

    @Override
    public PublicBalance balance(String participantId, String customerToken, UUID accountId) {
        String tokenHash = verifiedToken(customerToken);
        return jdbc.query("SELECT a.id, a.currency, a.posted_balance_minor, a.posted_balance_minor - a.reserved_balance_minor " +
                        "FROM ledger_account a JOIN public_customer c ON c.id = a.customer_id " +
                        "WHERE a.id = ? AND c.participant_id = ? AND c.api_token_hash = ?",
                rs -> rs.next() ? new PublicBalance(rs.getObject(1, UUID.class), rs.getString(2), rs.getLong(3), rs.getLong(4)) : null,
                accountId, participantId, tokenHash);
    }

    private String verifiedToken(String token) {
        if (token == null || token.isBlank()) throw new SecurityException("la autorización del cliente no es válida");
        return hashToken(token);
    }

    public static String hashToken(String value) {
        try { return java.util.HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(value.getBytes(StandardCharsets.UTF_8))); }
        catch (Exception e) { throw new IllegalStateException("no se pudo validar el token sintético", e); }
    }
}
