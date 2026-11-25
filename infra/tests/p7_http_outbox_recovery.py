"""Docker-only probe for HTTP outbox recovery; orchestration stays in the host shell."""

import json
import os
import time
import urllib.error
import urllib.request
import uuid

import psycopg

BANK_URL = "http://bank-core:8080"
NETWORK_URL = "http://network:8080"
DB_USER = os.environ["POSTGRES_USER"]
DB_PASSWORD = os.environ["POSTGRES_PASSWORD"]
PREFIX = os.environ.get("P7_IDEMPOTENCY_PREFIX", "p7-http-recovery-")
COSTA = "00000000-0000-4000-8000-000000000101"
SIERRA = "00000000-0000-4000-8000-000000000102"
COSTA_NETWORK = "00000000-0000-4000-8000-000000000201"
SIERRA_NETWORK = "00000000-0000-4000-8000-000000000202"
SOURCE_ACCOUNT = "00000000-0000-4000-8000-000000000001"
DESTINATION_ACCOUNT = "00000000-0000-4000-8000-000000000002"


def http_json(method, url, body=None, headers=None):
    encoded = json.dumps(body).encode() if body is not None else None
    request = urllib.request.Request(url, data=encoded, method=method, headers=headers or {})
    if encoded is not None:
        request.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(request, timeout=5) as response:
            data = response.read()
            return response.status, json.loads(data) if data else {}
    except urllib.error.HTTPError as error:
        data = error.read()
        return error.code, json.loads(data) if data else {}


def query(host, database, sql, params=()):
    with psycopg.connect(host=host, dbname=database, user=DB_USER, password=DB_PASSWORD) as connection:
        with connection.cursor() as cursor:
            cursor.execute(sql, params)
            return cursor.fetchall()


def wait_ready(url, label, timeout=90):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        status, _ = http_json("GET", f"{url}/health/ready")
        if status == 200:
            return
        time.sleep(1)
    raise AssertionError(f"{label} no recuperó readiness dentro de {timeout}s")


def latest_outbox():
    rows = query(
        "costa-db", "cholopay_costa",
        "SELECT event_id,aggregate_id,payload_hash,serialized_envelope,state,attempt_count "
        "FROM inter_network_outbox WHERE idempotency_key LIKE %s ORDER BY created_at DESC LIMIT 1",
        (PREFIX + "%",),
    )
    assert rows, f"no existe outbox sintético con prefijo {PREFIX}"
    return rows[0]


def wait_bank_http(timeout=90):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        status, _ = http_json("GET", f"{BANK_URL}/internal/v1/inter-network/transfers")
        if status in (404, 405):
            return
        time.sleep(1)
    raise AssertionError(f"bank-core no recuperó su listener HTTP dentro de {timeout}s")


def prepare():
    wait_bank_http()
    key = PREFIX + str(uuid.uuid4())
    correlation_id = str(uuid.uuid4())
    causation_id = str(uuid.uuid4())
    body = {
        "sourceAccountId": SOURCE_ACCOUNT,
        "destinationAccountId": DESTINATION_ACCOUNT,
        "sourceParticipantId": COSTA,
        "destinationParticipantId": SIERRA,
        "sourceNetworkId": COSTA_NETWORK,
        "destinationNetworkId": SIERRA_NETWORK,
        "amountMinor": 125,
        "currency": "USD",
        "correlationId": correlation_id,
        "causationId": causation_id,
        "traceId": "p7-http-outbox-recovery",
    }
    status, response = http_json(
        "POST", f"{BANK_URL}/internal/v1/inter-network/transfers", body,
        {"Idempotency-Key": key},
    )
    assert status == 202, (status, response)
    event_id, aggregate_id, payload_hash, envelope, state, attempts = latest_outbox()
    assert state in ("PENDING", "IN_FLIGHT"), state
    assert str(aggregate_id) == response["transferId"]
    assert str(event_id) == response["eventId"]
    assert payload_hash == response["payloadHash"]
    assert attempts >= 0
    print(f"P7 PREPARE PASS: transfer_id={aggregate_id} event_id={event_id} state={state}")


def assert_pending_or_leased():
    event_id, aggregate_id, payload_hash, envelope, state, attempts = latest_outbox()
    assert state in ("PENDING", "IN_FLIGHT"), (state, attempts)
    assert attempts >= 1, (state, attempts)
    print(f"P7 OUTAGE PASS: event_id={event_id} aggregate_id={aggregate_id} state={state} attempts={attempts}")


def assert_final():
    wait_ready(NETWORK_URL, "network")
    wait_bank_http()
    deadline = time.monotonic() + 90
    while time.monotonic() < deadline:
        row = latest_outbox()
        if row[4] == "PUBLISHED":
            break
        time.sleep(1)
    event_id, aggregate_id, payload_hash, envelope, state, attempts = latest_outbox()
    assert state == "PUBLISHED", (state, attempts)
    assert attempts >= 1
    envelope_json = json.loads(envelope)
    assert envelope_json["event_id"] == str(event_id)
    assert envelope_json["aggregate_id"] == str(aggregate_id)
    assert len(payload_hash) == 64 and all(character in "0123456789abcdef" for character in payload_hash)

    rows = query(
        "network-db", "cholopay_network",
        "SELECT COUNT(*), COUNT(DISTINCT message_id), COUNT(DISTINCT idempotency_key), "
        "MIN(message_id::text), MIN(idempotency_key), MIN(payload_hash), MIN(payload::text) "
        "FROM inter_network_message WHERE idempotency_key LIKE %s",
        ("transfer:" + PREFIX + "%",),
    )
    count, message_ids, idempotency_keys, message_id, message_key, network_hash, network_payload = rows[0]
    assert (count, message_ids, idempotency_keys) == (1, 1, 1), rows[0]
    assert str(message_id) == str(aggregate_id)
    assert message_key == "transfer:" + query(
        "costa-db", "cholopay_costa", "SELECT idempotency_key FROM inter_network_outbox WHERE event_id=%s", (event_id,)
    )[0][0]
    assert network_payload and network_hash
    print(
        "P7 PASS: HTTP outbox recovery; "
        f"event_id={event_id} aggregate_id={aggregate_id} payload_hash={payload_hash} "
        f"network_message_id={message_id} durable_messages=1 state={state} readiness=200 "
        "(synthetic at-least-once boundary; no universal exactly-once claim)"
    )


MODES = {"prepare": prepare, "assert-pending": assert_pending_or_leased, "assert-final": assert_final}


if __name__ == "__main__":
    mode = os.environ.get("P7_MODE", "prepare")
    if mode not in MODES:
        raise SystemExit(f"modo inválido: {mode}")
    MODES[mode]()
