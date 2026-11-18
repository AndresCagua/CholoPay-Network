package com.cholopay.bankcore.service;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

/** Casos públicos sintéticos de alta de cliente y consulta de cuentas propias. */
public interface PublicAccountService {
    OnboardedCustomer onboard(String participantId, String syntheticReference);
    List<PublicAccount> accounts(String participantId, String customerToken);
    PublicBalance balance(String participantId, String customerToken, UUID accountId);

    default PublicLedgerPage<PublicMovement> movements(String participantId, String customerToken, UUID accountId, int limit, String cursor) {
        throw new UnsupportedOperationException();
    }

    default PublicLedgerPage<PublicJournal> journals(String participantId, String customerToken, UUID accountId, int limit, String cursor) {
        throw new UnsupportedOperationException();
    }

    record OnboardedCustomer(UUID customerId, String syntheticReference, String accessToken, List<PublicAccount> accounts) {}
    record PublicAccount(@JsonProperty("account_id") UUID id, String currency,
                         @JsonProperty("balance_minor") long balanceMinor,
                         @JsonProperty("available_balance_minor") long availableBalanceMinor) {}
    record PublicBalance(@JsonProperty("account_id") UUID accountId, String currency,
                         @JsonProperty("balance_minor") long balanceMinor,
                         @JsonProperty("available_balance_minor") long availableBalanceMinor) {}

    record PublicLedgerPage<T>(List<T> data, @JsonProperty("next_cursor") String nextCursor) {}
    record PublicMovement(UUID id, @JsonProperty("journal_id") UUID journalId,
                          @JsonProperty("account_id") UUID accountId,
                          @JsonProperty("amount_minor") long amountMinor, String direction, String currency,
                          @JsonProperty("created_at") Instant createdAt) {}
    record PublicJournal(UUID id, @JsonProperty("transaction_id") UUID transactionId,
                         @JsonProperty("idempotency_key") String idempotencyKey,
                         @JsonProperty("correlation_id") UUID correlationId,
                         List<PublicJournalEntry> entries, @JsonProperty("created_at") Instant createdAt) {}
    record PublicJournalEntry(@JsonProperty("account_id") UUID accountId,
                              @JsonProperty("amount_minor") long amountMinor, String direction, String currency) {}
}
