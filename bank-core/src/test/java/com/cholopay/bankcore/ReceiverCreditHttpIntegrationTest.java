package com.cholopay.bankcore;

import com.cholopay.bankcore.api.ReceiverCreditController;
import com.cholopay.bankcore.security.AdapterRequestAuthenticator;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.web.servlet.MockMvc;

import java.time.Instant;
import java.util.UUID;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@Tag("integration")
@SpringBootTest
@AutoConfigureMockMvc
class ReceiverCreditHttpIntegrationTest {
    @Autowired MockMvc mvc;
    @Autowired JdbcTemplate jdbc;
    @Autowired AdapterRequestAuthenticator authenticator;
    private UUID clearing;
    private UUID destination;

    @BeforeEach
    void reset() {
        jdbc.update("DELETE FROM receiver_credit_inbox"); jdbc.update("DELETE FROM journal_entry"); jdbc.update("DELETE FROM journal"); jdbc.update("DELETE FROM ledger_account");
        clearing = UUID.randomUUID(); destination = UUID.randomUUID();
        jdbc.update("INSERT INTO ledger_account(id,owner_ref,currency,posted_balance_minor) VALUES (?, 'synthetic-network-clearing', 'USD', 500)", clearing);
        jdbc.update("INSERT INTO ledger_account(id,owner_ref,currency,posted_balance_minor) VALUES (?, 'synthetic-receiver', 'USD', 0)", destination);
    }

    @Test
    void acceptsSignedCommandAndRejectsTamperedPayload() throws Exception {
        UUID delivery = UUID.randomUUID(), event = UUID.randomUUID(), transfer = UUID.randomUUID(), requestId = UUID.randomUUID();
        ReceiverCreditController.ReceiverCreditRequest request = new ReceiverCreditController.ReceiverCreditRequest(delivery, event, transfer, destination, clearing, 100);
        String payload = ReceiverCreditController.canonical(request), timestamp = Long.toString(Instant.now().getEpochSecond());
        String json = "{\"deliveryId\":\"" + delivery + "\",\"eventId\":\"" + event + "\",\"transferId\":\"" + transfer + "\",\"destinationAccountId\":\"" + destination + "\",\"clearingAccountId\":\"" + clearing + "\",\"amountMinor\":100}";
        mvc.perform(post("/internal/v1/receiver-credits").header("X-CholoPay-Request-Timestamp", timestamp).header("X-CholoPay-Request-Id", requestId).header("X-CholoPay-Signature", authenticator.sign(timestamp, requestId.toString(), payload)).contentType("application/json").content(json)).andExpect(status().isOk());
        mvc.perform(post("/internal/v1/receiver-credits").header("X-CholoPay-Request-Timestamp", timestamp).header("X-CholoPay-Request-Id", UUID.randomUUID()).header("X-CholoPay-Signature", "bad").contentType("application/json").content(json)).andExpect(status().isUnauthorized());
    }
}
