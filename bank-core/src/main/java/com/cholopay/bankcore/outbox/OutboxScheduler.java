package com.cholopay.bankcore.outbox;

import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.stereotype.Component;

import java.time.Instant;

/** Invokes the dispatcher only when outbox delivery is explicitly enabled. */
@Component
@ConditionalOnProperty(name = "cholopay.outbox.enabled", havingValue = "true")
public class OutboxScheduler {
    private final OutboxDispatcher dispatcher;

    public OutboxScheduler(OutboxDispatcher dispatcher) {
        this.dispatcher = dispatcher;
    }

    @Scheduled(fixedDelayString = "${cholopay.outbox.poll-delay-ms:1000}")
    public void dispatch() {
        dispatcher.dispatchOnce(Instant.now());
    }
}
