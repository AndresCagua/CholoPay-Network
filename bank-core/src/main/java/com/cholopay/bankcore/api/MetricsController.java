package com.cholopay.bankcore.api;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.concurrent.atomic.AtomicLong;

@RestController
public class MetricsController {
    private final long startedAt = System.currentTimeMillis();
    private final AtomicLong httpRequests = new AtomicLong();

    @GetMapping(value = "/metrics", produces = "text/plain; version=0.0.4; charset=utf-8")
    public String metrics() {
        long requests = httpRequests.incrementAndGet();
        long uptimeSeconds = (System.currentTimeMillis() - startedAt) / 1000;
        return "# HELP cholonet_process_info Process information for the local synthetic service.\n"
                + "# TYPE cholonet_process_info gauge\n"
                + "cholonet_process_info{service=\"bank-core\"} 1\n"
                + "# HELP cholonet_process_uptime_seconds Process uptime in seconds.\n"
                + "# TYPE cholonet_process_uptime_seconds gauge\n"
                + "cholonet_process_uptime_seconds " + uptimeSeconds + "\n"
                + "# HELP cholonet_http_requests_total HTTP requests observed by the local service.\n"
                + "# TYPE cholonet_http_requests_total counter\n"
                + "cholonet_http_requests_total{service=\"bank-core\"} " + requests + "\n"
                + "# HELP cholonet_synthetic_acceptance_info Scope marker for local synthetic acceptance only.\n"
                + "# TYPE cholonet_synthetic_acceptance_info gauge\n"
                + "cholonet_synthetic_acceptance_info{scope=\"local\"} 1\n";
    }
}
