package com.cholopay.bankcore;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;

import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@Tag("integration")
@SpringBootTest
@AutoConfigureMockMvc
@TestPropertySource(properties = {
        "BANK_API_TOKEN=test-token",
        "BILLER_UTILITIES_CLEARING_ACCOUNT_ID=00000000-0000-0000-0000-000000000002",
        "cholopay.participant-id=BANCO_COSTA"
})
class JdbcObligationPaymentIntegrationTest {
    @Autowired MockMvc mvc;
    @Autowired JdbcTemplate jdbc;
    private UUID source = UUID.fromString("00000000-0000-0000-0000-000000000001");
    private UUID clearing = UUID.fromString("00000000-0000-0000-0000-000000000002");

    @BeforeEach
    void reset() {
        jdbc.update("DELETE FROM bill_payment_outbox");
        jdbc.update("DELETE FROM payment_intent");
        jdbc.update("DELETE FROM balance_reservation");
        jdbc.update("DELETE FROM journal_entry");
        jdbc.update("DELETE FROM journal");
        jdbc.update("DELETE FROM ledger_account");
        jdbc.update("INSERT INTO ledger_account(id, owner_ref, currency, posted_balance_minor) VALUES (?, 'synthetic-electricity-source', 'USD', 1000), (?, 'synthetic-utilities-clearing', 'USD', 0)", source, clearing);
    }

    @Test
    void successReplayConflictAndInsufficientFundsArePersistentAndBalanced() throws Exception {
        String body = body("10.00");
        mvc.perform(post("/v1/obligations/electricity-1/payment-intents").header("Authorization", "Bearer test-token").header("Idempotency-Key", "electricity-1").contentType("application/json").content(body))
                .andExpect(status().isAccepted()).andExpect(jsonPath("$.status").value("PENDING_APPLICATION"));
        long sourceAfter = balance(source);
        mvc.perform(post("/v1/obligations/electricity-1/payment-intents").header("Authorization", "Bearer test-token").header("Idempotency-Key", "electricity-1").contentType("application/json").content(body))
                .andExpect(status().isAccepted()).andExpect(jsonPath("$.status").value("PENDING_APPLICATION"));
        assertEquals(sourceAfter, balance(source));
        mvc.perform(post("/v1/obligations/electricity-1/payment-intents").header("Authorization", "Bearer test-token").header("Idempotency-Key", "electricity-1").contentType("application/json").content(body("11.00")))
                .andExpect(status().isConflict());
        mvc.perform(post("/v1/obligations/electricity-2/payment-intents").header("Authorization", "Bearer test-token").header("Idempotency-Key", "electricity-2").contentType("application/json").content(body("20.00")))
                .andExpect(status().isUnprocessableEntity());
        assertEquals(sourceAfter, balance(source));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM journal", Integer.class));
        assertEquals(1, jdbc.queryForObject("SELECT COUNT(*) FROM bill_payment_outbox", Integer.class));
    }

    private String body(String amount) { return "{\"participantId\":\"BANCO_COSTA\",\"sourceAccountId\":\"" + source + "\",\"biller\":\"utilities\",\"service\":\"electricity\",\"amount\":\"" + amount + "\",\"currency\":\"USD\"}"; }
    private long balance(UUID account) { return jdbc.queryForObject("SELECT posted_balance_minor - reserved_balance_minor FROM ledger_account WHERE id = ?", Long.class, account); }
}
