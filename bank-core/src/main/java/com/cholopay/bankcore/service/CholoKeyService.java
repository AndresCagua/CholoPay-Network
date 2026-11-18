package com.cholopay.bankcore.service;

import java.time.Instant;
import java.util.UUID;

public interface CholoKeyService {
    Key create(String participant, UUID participantUuid, UUID network, String customerToken, UUID accountId, String type, String value, String kind, String idempotency);
    Key get(String participant, UUID participantUuid, String customerToken, UUID keyId);
    void transition(String participant, UUID participantUuid, String customerToken, UUID keyId, String action, String idempotency);
    String staticQr(String participant, UUID participantUuid, String customerToken, UUID keyId);
    String dynamicQr(String participant, UUID participantUuid, String customerToken, UUID keyId, long amountMinor, String reference, Instant expiresAt);
    Verification verifyQr(String payload);
    record Key(UUID keyId, String keyType, String keyValue, UUID accountId, String accountKind, String status, long version) {}
    record Verification(boolean accepted, String reason, UUID keyId, long amountMinor, String reference, Instant expiresAt) {}
}
