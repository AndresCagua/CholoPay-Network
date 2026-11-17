package com.cholopay.bankcore;

import com.cholopay.bankcore.api.ObligationPaymentController;
import com.cholopay.bankcore.service.ObligationPaymentGatewayService;
import org.junit.jupiter.api.Test;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import java.util.Map;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

class ObligationPaymentControllerTest {
    private final ObligationPaymentGatewayService service = mock(ObligationPaymentGatewayService.class);
    private final MockMvc mvc = MockMvcBuilders.standaloneSetup(new ObligationPaymentController(service, "bank-token")).build();

    @Test
    void requiresAuthenticationAndIdempotencyKeyBeforeDelegating() throws Exception {
        String body = "{\"participantId\":\"COSTA\",\"sourceAccountId\":\"" + UUID.randomUUID() + "\",\"biller\":\"transit\",\"amount\":\"25.00\",\"currency\":\"SYN\"}";
        mvc.perform(post("/v1/obligations/fine-1/payment-intents").contentType("application/json").content(body))
                .andExpect(status().isUnauthorized());
        mvc.perform(post("/v1/obligations/fine-1/payment-intents").header("Authorization", "Bearer bank-token").contentType("application/json").content(body))
                .andExpect(status().isBadRequest());
        verifyNoInteractions(service);
    }

    @Test
    void preservesLegacyGatewayForNonElectricityUtilities() throws Exception {
        when(service.start(eq("water-1"), eq("water-1"), any())).thenReturn(new ObligationPaymentGatewayService.GatewayResult(202, Map.of("status", "PENDING_BILLER"), null));
        String body = "{\"participantId\":\"COSTA\",\"sourceAccountId\":\"" + UUID.randomUUID() + "\",\"biller\":\"utilities\",\"service\":\"water\",\"amount\":\"18.00\",\"currency\":\"USD\"}";
        mvc.perform(post("/v1/obligations/water-1/payment-intents").header("Authorization", "Bearer bank-token").header("Idempotency-Key", "water-1").contentType("application/json").content(body))
                .andExpect(status().isAccepted()).andExpect(jsonPath("$.status").value("PENDING_BILLER"));
        verify(service).start(eq("water-1"), eq("water-1"), any());
    }

    @Test
    void rejectsUnsupportedObligationWithoutCallingEitherPath() throws Exception {
        String body = "{\"participantId\":\"COSTA\",\"sourceAccountId\":\"" + UUID.randomUUID() + "\",\"biller\":\"utilities\",\"service\":\"gas\",\"amount\":\"1.00\",\"currency\":\"USD\"}";
        mvc.perform(post("/v1/obligations/gas-1/payment-intents").header("Authorization", "Bearer bank-token").header("Idempotency-Key", "gas-1").contentType("application/json").content(body))
                .andExpect(status().isUnprocessableEntity());
        verifyNoInteractions(service);
    }

    @Test
    void returnsGatewaySyntheticStateWithoutTurningItIntoALedgerResult() throws Exception {
        when(service.start(eq("fine-1"), eq("pay-1"), any())).thenReturn(new ObligationPaymentGatewayService.GatewayResult(202, Map.of("status", "PENDING_BILLER", "ledger_boundary", "no ledger"), null));
        String body = "{\"participantId\":\"COSTA\",\"sourceAccountId\":\"" + UUID.randomUUID() + "\",\"biller\":\"transit\",\"amount\":\"25.00\",\"currency\":\"SYN\"}";
        mvc.perform(post("/v1/obligations/fine-1/payment-intents").header("Authorization", "Bearer bank-token").header("Idempotency-Key", "pay-1").contentType("application/json").content(body))
                .andExpect(status().isAccepted()).andExpect(jsonPath("$.status").value("PENDING_BILLER"));
        verify(service).start(eq("fine-1"), eq("pay-1"), any());
    }
}
