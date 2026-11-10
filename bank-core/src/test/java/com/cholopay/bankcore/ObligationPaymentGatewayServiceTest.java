package com.cholopay.bankcore;

import com.cholopay.bankcore.service.ObligationPaymentGatewayService;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.Test;

import java.net.InetSocketAddress;
import java.net.http.HttpClient;
import java.nio.charset.StandardCharsets;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.*;

class ObligationPaymentGatewayServiceTest {
    @Test
    void delegatesParticipantScopedRequestWithSourceAccountAndReturnsSyntheticGatewayState() throws Exception {
        AtomicReference<String> requestBody = new AtomicReference<>();
        AtomicReference<String> apiKey = new AtomicReference<>();
        AtomicReference<String> idempotency = new AtomicReference<>();
        HttpServer server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/v1/bill-payments", exchange -> {
            requestBody.set(new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8));
            apiKey.set(exchange.getRequestHeaders().getFirst("x-api-key"));
            idempotency.set(exchange.getRequestHeaders().getFirst("idempotency-key"));
            byte[] body = "{\"id\":\"bp_sandbox_1\",\"status\":\"PENDING_BILLER\",\"ledger_boundary\":\"no ledger\"}".getBytes(StandardCharsets.UTF_8);
            exchange.sendResponseHeaders(202, body.length); exchange.getResponseBody().write(body); exchange.close();
        });
        server.start();
        try {
            UUID source = UUID.randomUUID();
            var service = new ObligationPaymentGatewayService("BANCO_COSTA", "http://127.0.0.1:" + server.getAddress().getPort(), "sk_sandbox_merchant_rw", HttpClient.newHttpClient(), new ObjectMapper());
            var result = service.start("obligation-fine-001", "payment-1",
                    new ObligationPaymentGatewayService.ObligationPaymentRequest("BANCO_COSTA", source, "transit", "25.00", "SYN", "traffic_fine", null));
            assertEquals(202, result.status());
            assertEquals("PENDING_BILLER", result.body().get("status"));
            assertEquals(source.toString(), result.body().get("source_account_id"));
            assertEquals("sk_sandbox_merchant_rw", apiKey.get());
            assertEquals("bank-obligation:BANCO_COSTA:payment-1", idempotency.get());
            assertTrue(requestBody.get().contains("\"participant_id\":\"BANCO_COSTA\""));
            assertTrue(requestBody.get().contains("\"source_account_id\":\"" + source + "\""));
        } finally { server.stop(0); }
    }

    @Test
    void failsClosedWhenGatewayConfigurationOrParticipantIsMissing() {
        var service = new ObligationPaymentGatewayService("", "", "", HttpClient.newHttpClient(), new ObjectMapper());
        var missing = service.start("obligation-1", "key", null);
        assertEquals(503, missing.status());

        var configured = new ObligationPaymentGatewayService("BANCO_COSTA", "http://127.0.0.1:1", "key", HttpClient.newHttpClient(), new ObjectMapper());
        var wrongParticipant = configured.start("obligation-1", "key",
                new ObligationPaymentGatewayService.ObligationPaymentRequest("BANCO_SIERRA", UUID.randomUUID(), "transit", "1.00", "SYN", null, null));
        assertEquals(422, wrongParticipant.status());
    }
}
