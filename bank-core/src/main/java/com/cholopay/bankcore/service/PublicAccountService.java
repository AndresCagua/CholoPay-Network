package com.cholopay.bankcore.service;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;
import java.util.UUID;

/** Casos públicos sintéticos de alta de cliente y consulta de cuentas propias. */
public interface PublicAccountService {
    OnboardedCustomer onboard(String participantId, String syntheticReference);
    List<PublicAccount> accounts(String participantId, String customerToken);
    PublicBalance balance(String participantId, String customerToken, UUID accountId);

    record OnboardedCustomer(UUID customerId, String syntheticReference, String accessToken, List<PublicAccount> accounts) {}
    record PublicAccount(@JsonProperty("account_id") UUID id, String currency,
                         @JsonProperty("balance_minor") long balanceMinor,
                         @JsonProperty("available_balance_minor") long availableBalanceMinor) {}
    record PublicBalance(@JsonProperty("account_id") UUID accountId, String currency,
                         @JsonProperty("balance_minor") long balanceMinor,
                         @JsonProperty("available_balance_minor") long availableBalanceMinor) {}
}
