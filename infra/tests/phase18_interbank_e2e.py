"""Aceptación Docker-only de settlement y entrega hacia Sierra y Oriente."""

import base64
import hashlib
import hmac
import json
import os
import time
import urllib.error
import urllib.request
import uuid

import psycopg

NETWORK_URL = "http://network:8080"
NETWORK_SECRET = os.environ["NETWORK_ADAPTER_SECRET"]
DB_USER = os.environ["POSTGRES_USER"]
DB_PASSWORD = os.environ["POSTGRES_PASSWORD"]

COSTA = uuid.UUID("00000000-0000-4000-8000-000000000101")
TARGETS = (
    {
        "name": "sierra",
        "participant": uuid.UUID("00000000-0000-4000-8000-000000000102"),
        "network_db": "sierra-db",
        "bank_db": "cholopay_sierra",
        "receiver_url": "http://bank-core-sierra:8080/internal/v1/receiver-credits",
        "receiver_secret": "sierra-local-test-only",
        "destination": uuid.UUID("00000000-0000-4000-8000-000000000013"),
        "clearing": uuid.UUID("00000000-0000-4000-8000-000000000012"),
    },
    {
        "name": "oriente",
        "participant": uuid.UUID("00000000-0000-4000-8000-000000000103"),
        "network_db": "oriente-db",
        "bank_db": "cholopay_oriente",
        "receiver_url": "http://bank-core-oriente:8080/internal/v1/receiver-credits",
        "receiver_secret": "oriente-local-test-only",
        "destination": uuid.UUID("00000000-0000-4000-8000-000000000023"),
        "clearing": uuid.UUID("00000000-0000-4000-8000-000000000022"),
    },
)


def http_json(method, url, body=None, headers=None):
    encoded = json.dumps(body).encode() if body is not None else None
    request = urllib.request.Request(url, data=encoded, method=method, headers=headers or {})
    if encoded is not None:
        request.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(request, timeout=5) as response:
            payload = response.read()
            return response.status, json.loads(payload) if payload else {}
    except urllib.error.HTTPError as error:
        payload = error.read()
        return error.code, json.loads(payload) if payload else {}


def wait_for_services():
    deadline = time.time() + 120
    while time.time() < deadline:
        status, _ = http_json("GET", f"{NETWORK_URL}/health/ready")
        if status == 200:
            return
        time.sleep(2)
    raise AssertionError("CholoNet no estuvo disponible dentro del tiempo límite")


def db_query(host, dbname, query, params=()):
    with psycopg.connect(host=host, dbname=dbname, user=DB_USER, password=DB_PASSWORD) as connection:
        with connection.cursor() as cursor:
            cursor.execute(query, params)
            return cursor.fetchall()


def sign_receiver(payload, secret):
    canonical = ":".join(
        [
            str(payload["deliveryId"]),
            str(payload["eventId"]),
            str(payload["transferId"]),
            str(payload["destinationAccountId"]),
            str(payload["clearingAccountId"]),
            str(payload["amountMinor"]),
            "USD",
        ]
    )
    timestamp = str(int(time.time()))
    request_id = str(payload["deliveryId"])
    signature = base64.b64encode(hmac.new(secret.encode(), f"{timestamp}.{request_id}.{canonical}".encode(), hashlib.sha256).digest()).decode()
    return {
        "Content-Type": "application/json",
        "X-CholoPay-Request-Timestamp": timestamp,
        "X-CholoPay-Request-Id": request_id,
        "X-CholoPay-Signature": signature,
    }


def wait_for_completion(transfer_id):
    deadline = time.time() + 90
    while time.time() < deadline:
        status, body = http_json("GET", f"{NETWORK_URL}/internal/v1/transfers/{transfer_id}")
        observed_state = body.get("state", body.get("State"))
        if status == 200 and observed_state == "COMPLETED":
            return body
        time.sleep(2)
    transfer_state = db_query(
        "network-db",
        "cholopay_network",
        "SELECT state FROM transfer_instruction WHERE transfer_id=%s",
        (transfer_id,),
    )
    delivery_state = db_query(
        "network-db",
        "cholopay_network",
        "SELECT state,attempt_count,failure_code,failure_detail FROM settlement_delivery WHERE transfer_id=%s",
        (transfer_id,),
    )
    raise AssertionError(f"la transferencia {transfer_id} no llegó a COMPLETED: instruction={transfer_state}, delivery={delivery_state}")


def settle_target(target):
    transfer_id = uuid.uuid4()
    correlation_id = uuid.uuid4()
    idempotency_key = f"phase18-{target['name']}"
    create_body = {
        "transfer_id": str(transfer_id),
        "idempotency_key": idempotency_key,
        "source_participant_id": str(COSTA),
        "destination_participant_id": str(target["participant"]),
        "destination_account_id": str(target["destination"]),
        "clearing_account_id": str(target["clearing"]),
        "amount_minor": 125,
        "currency": "USD",
        "correlation_id": str(correlation_id),
    }
    status, _ = http_json("POST", f"{NETWORK_URL}/internal/v1/transfers", create_body)
    assert status == 202, (target["name"], "create", status)
    for state in ("VALIDATING", "FUNDS_RESERVED", "READY_FOR_SETTLEMENT"):
        status, _ = http_json(
            "POST",
            f"{NETWORK_URL}/internal/v1/transfers/{transfer_id}/transitions",
            {"state": state, "causation_id": str(uuid.uuid4())},
        )
        assert status == 204, (target["name"], state, status)
    status, body = http_json(
        "POST",
        f"{NETWORK_URL}/internal/v1/settlements",
        {
            "transfer_id": str(transfer_id),
            "source_participant_id": str(COSTA),
            "destination_participant_id": str(target["participant"]),
            "destination_account_id": str(target["destination"]),
            "clearing_account_id": str(target["clearing"]),
            "amount_minor": 125,
            "idempotency_key": f"{idempotency_key}-settlement",
            "correlation_id": str(correlation_id),
        },
    )
    assert status == 201 and body.get("state") == "SETTLED", (target["name"], "settle", status, body)
    wait_for_completion(transfer_id)

    delivery = db_query(
        "network-db",
        "cholopay_network",
        "SELECT delivery_id,event_id,transfer_id,destination_account_id,clearing_account_id,amount_minor,attempt_count,state FROM settlement_delivery WHERE transfer_id=%s",
        (transfer_id,),
    )
    assert len(delivery) == 1, (target["name"], delivery)
    delivery_id, event_id, stored_transfer, destination, clearing, amount, attempts, state = delivery[0]
    assert stored_transfer == transfer_id and state == "APPLIED" and attempts >= 1

    inbox_before = db_query(
        target["network_db"],
        target["bank_db"],
        "SELECT COUNT(*), COUNT(DISTINCT journal_id) FROM receiver_credit_inbox WHERE delivery_id=%s",
        (delivery_id,),
    )[0]
    replay_payload = {
        "deliveryId": str(delivery_id),
        "eventId": str(event_id),
        "transferId": str(transfer_id),
        "destinationAccountId": str(destination),
        "clearingAccountId": str(clearing),
        "amountMinor": amount,
    }
    replay_status, replay_body = http_json(
        "POST", target["receiver_url"], replay_payload, sign_receiver(replay_payload, target["receiver_secret"])
    )
    assert replay_status == 200 and replay_body.get("state") == "APPLIED", (target["name"], replay_status, replay_body)
    inbox_after = db_query(
        target["network_db"],
        target["bank_db"],
        "SELECT COUNT(*), COUNT(DISTINCT journal_id) FROM receiver_credit_inbox WHERE delivery_id=%s",
        (delivery_id,),
    )[0]
    assert inbox_before == inbox_after == (1, 1), (target["name"], inbox_before, inbox_after)
    other_destinations = [
        other["destination"] for other in TARGETS if other["name"] != target["name"]
    ]
    cross_bank_rows = db_query(
        target["network_db"],
        target["bank_db"],
        "SELECT COUNT(*) FROM ledger_account WHERE id = ANY(%s)",
        ([str(account) for account in other_destinations],),
    )[0][0]
    assert cross_bank_rows == 0, (target["name"], "cross-bank fixtures", cross_bank_rows)
    return target["name"]


def main():
    wait_for_services()
    completed = [settle_target(target) for target in TARGETS]
    print(f"PHASE18 PASS: settlement y entrega idempotente completados para {','.join(completed)}")


if __name__ == "__main__":
    main()
