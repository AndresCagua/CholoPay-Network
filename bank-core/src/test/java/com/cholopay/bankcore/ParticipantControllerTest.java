package com.cholopay.bankcore;

import com.cholopay.bankcore.api.ParticipantController;
import org.junit.jupiter.api.Test;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

class ParticipantControllerTest {
    private final MockMvc mvc = MockMvcBuilders.standaloneSetup(
            new ParticipantController("BANCO_SIERRA", "bank-token")
    ).build();

    @Test
    void returnsConfiguredParticipantOnlyWithBankAuthorization() throws Exception {
        mvc.perform(get("/v1/participants/BANCO_SIERRA"))
                .andExpect(status().isUnauthorized())
                .andExpect(content().contentType("application/problem+json"))
                .andExpect(jsonPath("$.code").value("unauthorized"));

        mvc.perform(get("/v1/participants/BANCO_SIERRA")
                        .header("Authorization", "Bearer wrong-token"))
                .andExpect(status().isUnauthorized());

        mvc.perform(get("/v1/participants/BANCO_SIERRA")
                        .header("Authorization", "Bearer bank-token"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.participant_id").value("BANCO_SIERRA"))
                .andExpect(jsonPath("$.metadata.environment").value("synthetic"))
                .andExpect(jsonPath("$.metadata.status").value("ACTIVE"))
                .andExpect(jsonPath("$.bank_api_token").doesNotExist());
    }

    @Test
    void mismatchedParticipantIsNonEnumeratingNotFound() throws Exception {
        mvc.perform(get("/v1/participants/OTRO_BANCO")
                        .header("Authorization", "Bearer bank-token"))
                .andExpect(status().isNotFound())
                .andExpect(content().contentType("application/problem+json"))
                .andExpect(jsonPath("$.code").value("participant_not_found"))
                .andExpect(jsonPath("$.detail").value("el participante no está disponible"))
                .andExpect(jsonPath("$.participant_id").doesNotExist());
    }

    @Test
    void missingConfigurationFailsClosed() throws Exception {
        MockMvc unconfigured = MockMvcBuilders.standaloneSetup(
                new ParticipantController("", "")
        ).build();

        unconfigured.perform(get("/v1/participants/BANCO_SIERRA")
                        .header("Authorization", "Bearer bank-token"))
                .andExpect(status().isServiceUnavailable())
                .andExpect(content().contentType("application/problem+json"))
                .andExpect(jsonPath("$.code").value("configuration_missing"));
    }
}
