package com.cholopay.bankcore;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;

import java.util.UUID;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@Tag("integration")
@SpringBootTest
@AutoConfigureMockMvc
@TestPropertySource(properties = {
        "BANK_API_TOKEN=test-token",
        "cholopay.participant-id=BANCO_COSTA"
})
class PublicAccountIntegrationTest {
    @Autowired MockMvc mvc;
    @Autowired JdbcTemplate jdbc;
    @Autowired ObjectMapper objectMapper;

    @AfterEach
    void cleanup() {
        jdbc.update("DELETE FROM ledger_account WHERE customer_id IS NOT NULL");
        jdbc.update("DELETE FROM public_customer");
    }

    @Test
    void onboardingListsOwnAccountAndRejectsCrossAccountRead() throws Exception {
        String reference = "public-customer-" + UUID.randomUUID();
        String response = mvc.perform(post("/v1/customers")
                        .header("Authorization", "Bearer test-token")
                        .contentType("application/json")
                        .content("{\"synthetic_reference\":\"" + reference + "\"}"))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.accounts[0].account_id").exists())
                .andReturn().getResponse().getContentAsString();
        JsonNode created = objectMapper.readTree(response);
        String token = created.path("access_token").asText();
        UUID accountId = UUID.fromString(created.path("accounts").get(0).path("account_id").asText());

        mvc.perform(get("/v1/accounts").header("Authorization", "Bearer " + token))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.accounts[0].account_id").value(accountId.toString()));
        mvc.perform(get("/v1/accounts").header("Authorization", "Bearer test-token"))
                .andExpect(status().isUnauthorized());
        mvc.perform(get("/v1/accounts/" + UUID.randomUUID() + "/balance")
                        .header("Authorization", "Bearer " + token))
                .andExpect(status().isNotFound());
    }

    @Test
    void onboardingRequiresBankAuthorization() throws Exception {
        mvc.perform(post("/v1/customers")
                        .contentType("application/json")
                        .content("{\"synthetic_reference\":\"customer-unauthorized\"}"))
                .andExpect(status().isUnauthorized());
    }
}
