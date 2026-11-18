package com.cholopay.bankcore;

import com.cholopay.bankcore.api.BankIdentityController;
import org.junit.jupiter.api.Test;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

class BankIdentityControllerTest {
    private final MockMvc mvc = MockMvcBuilders.standaloneSetup(
            new BankIdentityController("BANCO_SIERRA", "sierra-token")
    ).build();

    @Test
    void returnsParticipantOnlyWithItsBankToken() throws Exception {
        mvc.perform(get("/internal/v1/bank-identity"))
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.error").value("unauthorized"));

        mvc.perform(get("/internal/v1/bank-identity").header("Authorization", "Bearer wrong-token"))
                .andExpect(status().isUnauthorized());

        mvc.perform(get("/internal/v1/bank-identity").header("Authorization", "Bearer sierra-token"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.participantId").value("BANCO_SIERRA"));
    }
}
