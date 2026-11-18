package com.cholopay.bankcore.service;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionTemplate;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/** Persiste el límite local del pago sintético; no liquida dinero real. */
@Service
public class JdbcObligationPaymentService {
    private final JdbcTemplate jdbc;
    private final ObjectMapper json;
    private final TransactionTemplate transactions;
    private final ObligationPaymentGatewayService gateway;
    private final String configuredParticipant;
    private final String clearingAccount;

    public JdbcObligationPaymentService(JdbcTemplate jdbc, ObjectMapper json,
                                        org.springframework.transaction.PlatformTransactionManager transactionManager,
                                        ObligationPaymentGatewayService gateway,
                                        @Value("${cholopay.participant-id:}") String configuredParticipant,
                                        @Value("${BILLER_UTILITIES_CLEARING_ACCOUNT_ID:}") String clearingAccount) {
        this.jdbc = jdbc;
        this.json = json;
        this.transactions = new TransactionTemplate(transactionManager);
        this.gateway = gateway;
        this.configuredParticipant = configuredParticipant;
        this.clearingAccount = clearingAccount;
    }

    /** La transacción de débito termina antes de que este método aplique el pago por HTTP. */
    public Result start(String obligationId, String idempotencyKey, Request request) {
        Result persisted = transactions.execute(status -> persist(obligationId, idempotencyKey, request));
        if (persisted == null || persisted.error != null || persisted.paymentId == null
                || "APPLIED".equals(persisted.body.get("status"))) return persisted;
        return applyPersistedOutbox(persisted.paymentId, idempotencyKey, persisted);
    }

    private Result persist(String obligationId, String idempotencyKey, Request request) {
        if (isBlank(configuredParticipant) || isBlank(clearingAccount)) return Result.configurationMissing();
        if (request == null || isBlank(obligationId) || isBlank(idempotencyKey)
                || !configuredParticipant.equals(request.participantId()) || request.sourceAccountId() == null
                || !"utilities".equals(request.biller()) || !"electricity".equals(request.service())) {
            return Result.invalid("solicitud de pago de electricidad inválida");
        }
        long amountMinor;
        try {
            BigDecimal amount = new BigDecimal(request.amount()).setScale(2, RoundingMode.UNNECESSARY);
            if (amount.signum() <= 0) throw new ArithmeticException();
            amountMinor = amount.movePointRight(2).longValueExact();
        } catch (Exception invalidAmount) {
            return Result.invalid("el importe debe ser un monto USD positivo con hasta dos decimales");
        }
        if (!"USD".equals(request.currency())) return Result.invalid("la moneda debe ser USD");
        UUID clearingId;
        try { clearingId = UUID.fromString(clearingAccount); }
        catch (IllegalArgumentException invalidConfiguration) { return Result.configurationMissing(); }
        String hash = hash(String.join(":", configuredParticipant, obligationId, request.sourceAccountId().toString(),
                request.biller(), request.service(), Long.toString(amountMinor), "USD", String.valueOf(request.invoiceId())));
        Existing existing = jdbc.query("SELECT id, request_hash, status, journal_id, application_result FROM payment_intent WHERE idempotency_key = ? FOR UPDATE",
                (rs, row) -> new Existing(rs.getObject("id", UUID.class), rs.getString("request_hash"), rs.getString("status"), rs.getObject("journal_id", UUID.class), rs.getString("application_result")), idempotencyKey)
                .stream().findFirst().orElse(null);
        if (existing != null) {
            if (!hash.equals(existing.hash())) return Result.conflict();
            return readResult(existing);
        }
        if (request.sourceAccountId().equals(clearingId)) return Result.invalid("la cuenta de origen y la cuenta de compensación deben ser distintas");
        List<UUID> ordered = List.of(request.sourceAccountId(), clearingId).stream().sorted(Comparator.comparing(UUID::toString)).toList();
        List<UUID> locked = jdbc.query("SELECT id FROM ledger_account WHERE id IN (?, ?) ORDER BY id FOR UPDATE",
                (rs, row) -> rs.getObject("id", UUID.class), ordered.get(0), ordered.get(1));
        if (locked.size() != 2) return Result.invalid("cuenta bancaria no encontrada");
        Long available = jdbc.queryForObject("SELECT posted_balance_minor - reserved_balance_minor FROM ledger_account WHERE id = ?", Long.class, request.sourceAccountId());
        if (available == null || available < amountMinor) return Result.insufficient();

        UUID paymentId = UUID.randomUUID(), journalId = UUID.randomUUID(), correlationId = UUID.randomUUID();
        jdbc.update("UPDATE ledger_account SET posted_balance_minor = posted_balance_minor - ? WHERE id = ? AND posted_balance_minor - reserved_balance_minor >= ?", amountMinor, request.sourceAccountId(), amountMinor);
        jdbc.update("UPDATE ledger_account SET posted_balance_minor = posted_balance_minor + ? WHERE id = ?", amountMinor, clearingId);
        jdbc.update("INSERT INTO balance_reservation(id, account_id, transfer_id, amount_minor, status) VALUES (?, ?, ?, ?, 'CAPTURED')", UUID.randomUUID(), request.sourceAccountId(), paymentId, amountMinor);
        String journalKey = "obligation:" + configuredParticipant + ":" + idempotencyKey;
        jdbc.update("INSERT INTO journal(id, transaction_id, idempotency_key, correlation_id, request_hash) VALUES (?, ?, ?, ?, ?)", journalId, paymentId, journalKey, correlationId, hash);
        jdbc.update("INSERT INTO journal_entry(id, journal_id, account_id, amount_minor, direction, currency) VALUES (?, ?, ?, ?, 'DEBIT', 'USD'), (?, ?, ?, ?, 'CREDIT', 'USD')",
                UUID.randomUUID(), journalId, request.sourceAccountId(), amountMinor, UUID.randomUUID(), journalId, clearingId, amountMinor);
        jdbc.update("INSERT INTO payment_intent(id, obligation_id, idempotency_key, request_hash, participant_id, source_account_id, clearing_account_id, amount_minor, currency, biller, service, invoice_id, journal_id, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'USD', ?, ?, ?, ?, 'PENDING_APPLICATION')",
                paymentId, obligationId, idempotencyKey, hash, configuredParticipant, request.sourceAccountId(), clearingId, amountMinor, request.biller(), request.service(), request.invoiceId(), journalId);
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("payment_intent_id", paymentId); payload.put("obligation_id", obligationId); payload.put("participant_id", configuredParticipant);
        payload.put("source_account_id", request.sourceAccountId()); payload.put("biller", request.biller()); payload.put("service", request.service()); payload.put("amount", request.amount()); payload.put("currency", "USD"); payload.put("invoice_id", request.invoiceId());
        try {
            String payloadJson = json.writeValueAsString(payload);
            jdbc.update("INSERT INTO bill_payment_outbox(event_id, payment_intent_id, event_name, schema_version, idempotency_key, payload, payload_hash) VALUES (?, ?, 'BILL_PAYMENT_REQUESTED', '1.0', ?, ?::jsonb, ?)", UUID.randomUUID(), paymentId, idempotencyKey, payloadJson, hash(payloadJson));
        } catch (Exception e) { throw new IllegalStateException("no se pudo serializar el evento de pago", e); }
        return Result.pending(paymentId, body(paymentId, obligationId, amountMinor, "PENDING_APPLICATION", journalId));
    }

    private Result applyPersistedOutbox(UUID paymentId, String idempotencyKey, Result persisted) {
        String payloadJson = jdbc.queryForObject("SELECT payload::text FROM bill_payment_outbox WHERE payment_intent_id = ?", String.class, paymentId);
        try {
            JsonNode payload = json.readTree(payloadJson);
            var gatewayResult = gateway.start(payload.get("obligation_id").asText(), idempotencyKey,
                    new ObligationPaymentGatewayService.ObligationPaymentRequest(payload.get("participant_id").asText(), UUID.fromString(payload.get("source_account_id").asText()),
                            payload.get("biller").asText(), payload.get("amount").asText(), payload.get("currency").asText(),
                            payload.get("service").asText(), payload.get("invoice_id").isNull() ? null : payload.get("invoice_id").asText()));
            if (gatewayResult.status() == 201 && "APPLIED".equals(gatewayResult.body().get("status"))) {
                Map<String, Object> applied = new LinkedHashMap<>(persisted.body);
                applied.putAll(gatewayResult.body()); applied.put("payment_intent_id", paymentId); applied.put("settlement", "not_claimed");
                String appliedJson = json.writeValueAsString(applied);
                transactions.executeWithoutResult(tx -> {
                    jdbc.update("UPDATE payment_intent SET status='APPLIED', application_result=?::jsonb WHERE id=?", appliedJson, paymentId);
                    jdbc.update("UPDATE bill_payment_outbox SET state='APPLIED' WHERE payment_intent_id=?", paymentId);
                });
                return new Result(201, applied, null, paymentId);
            }
        } catch (Exception applicationFailure) {
            try {
                transactions.executeWithoutResult(tx -> jdbc.update("UPDATE payment_intent SET status='UNKNOWN_PENDING_RECONCILIATION' WHERE id=? AND status='PENDING_APPLICATION'", paymentId));
                return new Result(HttpStatus.ACCEPTED.value(), bodyWithStatus(persisted.body, "UNKNOWN_PENDING_RECONCILIATION"), null, paymentId);
            } catch (RuntimeException reconciliationFailure) {
                // Nunca informar UNKNOWN si no se actualizó la fila durable; el replay debe observar el mismo estado.
                return persisted;
            }
        }
        return persisted;
    }

    public Map<String, Object> retrieve(String participantId, String customerToken, UUID paymentId) {
        String tokenHash = JdbcPublicAccountService.hashToken(requiredToken(customerToken));
        return jdbc.query("SELECT p.id, p.obligation_id, p.amount_minor, p.currency, p.status, p.journal_id, p.application_result "
                        + "FROM payment_intent p JOIN ledger_account a ON a.id=p.source_account_id "
                        + "JOIN public_customer c ON c.id=a.customer_id WHERE p.id=? AND p.participant_id=? AND c.api_token_hash=?",
                rs -> rs.next() ? paymentBody(rs.getObject(1, UUID.class), rs.getString(2), rs.getLong(3), rs.getString(4), rs.getString(5), rs.getObject(6, UUID.class), rs.getString(7)) : null,
                paymentId, participantId, tokenHash);
    }

    public Map<String, Object> refund(String participantId, String customerToken, UUID paymentId, String idempotencyKey, String requestedAmount) {
        String tokenHash = JdbcPublicAccountService.hashToken(requiredToken(customerToken));
        if (isBlank(idempotencyKey)) throw new IllegalArgumentException("idempotency_key_required");
        return transactions.execute(status -> {
            RefundExisting prior = jdbc.query("SELECT r.id, r.payment_intent_id, r.amount_minor, r.currency, r.status, r.request_hash FROM payment_intent_refund r WHERE r.idempotency_key=? FOR UPDATE",
                    (rs, row) -> new RefundExisting(rs.getObject(1, UUID.class), rs.getObject(2, UUID.class), rs.getLong(3), rs.getString(4), rs.getString(5), rs.getString(6)), idempotencyKey).stream().findFirst().orElse(null);
            Payment payment = jdbc.query("SELECT p.id,p.amount_minor,p.currency,p.status FROM payment_intent p JOIN ledger_account a ON a.id=p.source_account_id JOIN public_customer c ON c.id=a.customer_id WHERE p.id=? AND p.participant_id=? AND c.api_token_hash=? FOR UPDATE",
                    (rs, row) -> new Payment(rs.getObject(1, UUID.class), rs.getLong(2), rs.getString(3), rs.getString(4)), paymentId, participantId, tokenHash).stream().findFirst().orElse(null);
            if (payment == null) return null;
            long amount = payment.amountMinor();
            if (requestedAmount != null) {
                try { amount = new BigDecimal(requestedAmount).setScale(2, RoundingMode.UNNECESSARY).movePointRight(2).longValueExact(); }
                catch (Exception e) { throw new IllegalArgumentException("refund_amount_invalid"); }
            }
            if (amount != payment.amountMinor()) throw new IllegalArgumentException("refund_must_be_full");
            String hash = hash(paymentId + ":" + amount + ":USD");
            if (prior != null) {
                if (!hash.equals(prior.requestHash())) throw new IllegalStateException("idempotency_key_conflict");
                return refundBody(prior.id(), paymentId, prior.amountMinor(), prior.currency(), prior.status());
            }
            if (!"APPLIED".equals(payment.status())) throw new IllegalStateException("payment_not_refundable");
            RefundExisting existingForPayment = jdbc.query("SELECT id, payment_intent_id, amount_minor, currency, status, request_hash FROM payment_intent_refund WHERE payment_intent_id=? FOR UPDATE",
                    (rs,row) -> new RefundExisting(rs.getObject(1,UUID.class),rs.getObject(2,UUID.class),rs.getLong(3),rs.getString(4),rs.getString(5),rs.getString(6)), paymentId).stream().findFirst().orElse(null);
            if (existingForPayment != null) throw new IllegalStateException("idempotency_key_conflict");
            UUID refundId = UUID.randomUUID();
            jdbc.update("INSERT INTO payment_intent_refund(id,payment_intent_id,idempotency_key,request_hash,amount_minor,currency,status) VALUES (?,?,?,?,?,'USD','SUCCEEDED')", refundId, paymentId, idempotencyKey, hash, amount);
            return refundBody(refundId, paymentId, amount, "USD", "SUCCEEDED");
        });
    }

    private Map<String, Object> retrieveById(UUID paymentId) {
        return jdbc.queryForObject("SELECT id, obligation_id, amount_minor, currency, status, journal_id, application_result FROM payment_intent WHERE id=?", (rs,row) -> paymentBody(rs.getObject(1,UUID.class),rs.getString(2),rs.getLong(3),rs.getString(4),rs.getString(5),rs.getObject(6,UUID.class),rs.getString(7)), paymentId);
    }
    private Map<String, Object> paymentBody(UUID id, String obligation, long amount, String currency, String status, UUID journal, String applicationResult) {
        if (applicationResult != null) try { return new LinkedHashMap<>(json.readValue(applicationResult, Map.class)); } catch (Exception ignored) { }
        return new LinkedHashMap<>(Map.of("payment_intent_id", id, "obligation_id", obligation, "amount", BigDecimal.valueOf(amount, 2), "currency", currency, "status", status, "journal_id", journal, "settlement", "not_claimed"));
    }
    private Map<String,Object> refundBody(UUID id, UUID paymentId, long amount, String currency, String status) {
        return Map.of("refund_id", id, "payment_intent_id", paymentId, "amount", BigDecimal.valueOf(amount, 2),
                "currency", currency, "status", status, "ledger_boundary", "not_claimed");
    }
    private String requiredToken(String token) { if (isBlank(token)) throw new SecurityException("customer token required"); return token; }
    private record Payment(UUID id, long amountMinor, String currency, String status) {}
    private record RefundExisting(UUID id, UUID paymentIntentId, long amountMinor, String currency, String status, String requestHash) {}

    private Result readResult(Existing existing) {
        if (existing.applicationResult() != null) {
            try { return new Result(201, json.readValue(existing.applicationResult(), Map.class), null, existing.id()); } catch (Exception ignored) { }
        }
        Map<String, Object> body = jdbc.queryForObject("SELECT obligation_id, amount_minor FROM payment_intent WHERE id = ?", (rs, row) -> body(existing.id(), rs.getString(1), rs.getLong(2), existing.status(), existing.journalId()), existing.id());
        if ("APPLIED".equals(existing.status())) return new Result(201, body, null, existing.id());
        return Result.pending(existing.id(), body);
    }
    private Map<String, Object> body(UUID id, String obligation, long amount, String status, UUID journal) { return Map.of("payment_intent_id", id, "obligation_id", obligation, "amount", BigDecimal.valueOf(amount, 2), "currency", "USD", "status", status, "journal_id", journal, "settlement", "not_claimed"); }
    private Map<String, Object> bodyWithStatus(Map<String, Object> original, String status) {
        Map<String, Object> copy = new LinkedHashMap<>(original);
        copy.put("status", status);
        return copy;
    }
    private static boolean isBlank(String value) { return value == null || value.isBlank(); }
    private static String hash(String value) { try { return java.util.HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(value.getBytes(StandardCharsets.UTF_8))); } catch (Exception e) { throw new IllegalStateException(e); } }
    private record Existing(UUID id, String hash, String status, UUID journalId, String applicationResult) {}
    public record Request(String participantId, UUID sourceAccountId, String biller, String amount, String currency, String service, String invoiceId) {}
    public record Result(int status, Map<String, Object> body, String error, UUID paymentId) {
        static Result pending(UUID id, Map<String, Object> body) { return new Result(HttpStatus.ACCEPTED.value(), body, null, id); }
        static Result configurationMissing() { return new Result(503, Map.of("error", "configuration_missing", "message", "la configuración de compensación no está disponible"), "configuration", null); }
        static Result invalid(String message) { return new Result(422, Map.of("error", "invalid_request", "message", message), "invalid", null); }
        static Result conflict() { return new Result(409, Map.of("error", "idempotency_key_conflict", "message", "la clave de idempotencia ya fue utilizada con otra solicitud"), "conflict", null); }
        static Result insufficient() { return new Result(422, Map.of("error", "insufficient_funds", "message", "los fondos disponibles son insuficientes"), "insufficient_funds", null); }
    }
}
