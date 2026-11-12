package com.cholopay.bankcore.outbox;

public interface OutboxClient {
    DeliveryResult send(String serializedEnvelope);
    enum DeliveryResult { SUCCESS, REPLAY, CONTRACT_FAILURE, AUTH_FAILURE, CONFLICT, AMBIGUOUS_FAILURE }
}
