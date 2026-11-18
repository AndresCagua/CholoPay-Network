package com.cholopay.bankcore;

import com.cholopay.bankcore.api.CholoKeyController;
import com.cholopay.bankcore.service.CholoKeyService;
import org.junit.jupiter.api.Test;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import java.time.Instant;
import java.util.UUID;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

class CholoKeyControllerTest {
    private final UUID account = UUID.randomUUID();
    private final MockMvc mvc = MockMvcBuilders.standaloneSetup(new CholoKeyController(new FakeService(), "BANCO_COSTA",
            "00000000-0000-4000-8000-000000000101", "00000000-0000-4000-8000-000000000201", "bank-token")).build();

    @Test
    void customerTokenIsRequiredAndIdempotencyIsRequired() throws Exception {
        mvc.perform(post("/v1/keys").contentType("application/json").content("{\"accountId\":\"" + account + "\",\"keyType\":\"PHONE\",\"keyValue\":\"+593990000001\"}"))
                .andExpect(status().isUnauthorized());
        mvc.perform(post("/v1/keys").header("Authorization", "Bearer customer-token").contentType("application/json")
                .content("{\"accountId\":\"" + account + "\",\"keyType\":\"PHONE\",\"keyValue\":\"+593990000001\"}"))
                .andExpect(status().isBadRequest());
    }

    private class FakeService implements CholoKeyService {
        public Key create(String p, UUID pu, UUID n, String t, UUID a, String type, String value, String kind, String idem) { return new Key(UUID.randomUUID(), type, value, a, "PERSONAL", "ACTIVE", 1); }
        public Key get(String p, UUID pu, String t, UUID id) { throw new SecurityException(); }
        public void transition(String p, UUID pu, String t, UUID id, String action, String idem) {}
        public String staticQr(String p, UUID pu, String t, UUID id) { return "cholokey.qr.v1.static"; }
        public String dynamicQr(String p, UUID pu, String t, UUID id, long amount, String ref, Instant exp) { return "cholokey.qr.v1.dynamic"; }
        public Verification verifyQr(String payload) { return new Verification(false, "payload_no_reconocido", null, 0, null, null); }
    }
}
