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
        jdbc.update("DELETE FROM journal_entry WHERE account_id IN (SELECT id FROM ledger_account WHERE customer_id IS NOT NULL)");
        jdbc.update("DELETE FROM journal WHERE id NOT IN (SELECT DISTINCT journal_id FROM journal_entry)");
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

    @Test
    void movementsUseCursorIsolationAndProblemDetails() throws Exception {
        String reference = "ledger-customer-" + UUID.randomUUID();
        JsonNode created = objectMapper.readTree(mvc.perform(post("/v1/customers")
                        .header("Authorization", "Bearer test-token")
                        .contentType("application/json")
                        .content("{\"synthetic_reference\":\"" + reference + "\"}"))
                .andExpect(status().isCreated()).andReturn().getResponse().getContentAsString());
        String token = created.path("access_token").asText();
        UUID accountId = UUID.fromString(created.path("accounts").get(0).path("account_id").asText());
        insertJournal(accountId, "movement-1", 10);
        insertJournal(accountId, "movement-2", 20);

        String first = mvc.perform(get("/v1/accounts/" + accountId + "/movements")
                        .param("limit", "1").header("Authorization", "Bearer " + token))
                .andExpect(status().isOk()).andExpect(jsonPath("$.data.length()").value(1))
                .andExpect(jsonPath("$.next_cursor").isNotEmpty())
                .andReturn().getResponse().getContentAsString();
        String cursor = objectMapper.readTree(first).path("next_cursor").asText();
        mvc.perform(get("/v1/accounts/" + accountId + "/movements")
                        .param("limit", "1").param("cursor", cursor)
                        .header("Authorization", "Bearer " + token))
                .andExpect(status().isOk()).andExpect(jsonPath("$.data.length()").value(1))
                .andExpect(jsonPath("$.next_cursor").isEmpty());
        mvc.perform(get("/v1/accounts/" + accountId + "/movements")
                        .param("limit", "0").header("Authorization", "Bearer " + token))
                .andExpect(status().isBadRequest()).andExpect(jsonPath("$.code").value("invalid_limit"))
                .andExpect(jsonPath("$.detail").value("el límite debe estar entre 1 y 100"));
        mvc.perform(get("/v1/accounts/" + accountId + "/movements")
                        .header("Authorization", "Bearer " + token + "-tampered"))
                .andExpect(status().isUnauthorized()).andExpect(jsonPath("$.code").value("unauthorized"));
    }

    private void insertJournal(UUID accountId, String key, long amount) {
        UUID journalId = UUID.randomUUID();
        jdbc.update("INSERT INTO journal(id, transaction_id, idempotency_key, correlation_id) VALUES (?, ?, ?, ?)",
                journalId, UUID.randomUUID(), key, UUID.randomUUID());
        jdbc.update("INSERT INTO journal_entry(id, journal_id, account_id, amount_minor, direction, currency) VALUES (?, ?, ?, ?, 'DEBIT', 'USD')",
                UUID.randomUUID(), journalId, accountId, amount);
    }
}
