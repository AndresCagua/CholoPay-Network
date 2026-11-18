package com.cholopay.bankcore;

import com.cholopay.bankcore.api.PublicAccountController;
import com.cholopay.bankcore.service.PublicAccountService;
import org.junit.jupiter.api.Test;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import java.util.List;
import java.util.UUID;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

class PublicAccountControllerTest {
    private final UUID own = UUID.randomUUID();
    private final UUID other = UUID.randomUUID();
    private final MockMvc mvc = MockMvcBuilders.standaloneSetup(new PublicAccountController(new FakeService(), "BANCO_SIERRA", "bank-token")).build();

    @Test
    void onboardingRequiresBankAuthorizationAndReturnsSyntheticCustomerToken() throws Exception {
        mvc.perform(post("/v1/customers").contentType("application/json").content("{\"synthetic_reference\":\"customer-001\"}"))
                .andExpect(status().isUnauthorized());
        mvc.perform(post("/v1/customers").header("Authorization", "Bearer bank-token")
                        .contentType("application/json").content("{\"synthetic_reference\":\"customer-001\"}"))
                .andExpect(status().isCreated()).andExpect(jsonPath("$.access_token").value("customer-token"))
                .andExpect(jsonPath("$.accounts[0].account_id").value(own.toString()));
    }

    @Test
    void listingAndBalanceUseCustomerTokenAndDoNotCrossAccount() throws Exception {
        mvc.perform(get("/v1/accounts")).andExpect(status().isUnauthorized());
        mvc.perform(get("/v1/accounts").header("Authorization", "Bearer customer-token"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.accounts[0].account_id").value(own.toString()));
        mvc.perform(get("/v1/accounts/" + own + "/balance").header("Authorization", "Bearer customer-token"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.account_id").value(own.toString()));
        mvc.perform(get("/v1/accounts/" + other + "/balance").header("Authorization", "Bearer customer-token"))
                .andExpect(status().isNotFound());
    }

    private class FakeService implements PublicAccountService {
        public OnboardedCustomer onboard(String participantId, String reference) {
            return new OnboardedCustomer(UUID.randomUUID(), reference, "customer-token", List.of(new PublicAccount(own, "USD", 1000, 900)));
        }
        public List<PublicAccount> accounts(String participantId, String token) {
            if (!"customer-token".equals(token)) throw new SecurityException();
            return List.of(new PublicAccount(own, "USD", 1000, 900));
        }
        public PublicBalance balance(String participantId, String token, UUID accountId) {
            if (!"customer-token".equals(token)) throw new SecurityException();
            return own.equals(accountId) ? new PublicBalance(own, "USD", 1000, 900) : null;
        }
    }
}
