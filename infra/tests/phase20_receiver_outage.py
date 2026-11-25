"""Aceptación Docker-only de recuperación ante caída breve del receptor Sierra."""

import os
import sys
import time
import uuid

import psycopg
import urllib.error
import urllib.request
import json

NETWORK_URL = "http://network:8080"
NETWORK_SECRET = os.environ["NETWORK_ADAPTER_SECRET"]
OPERATOR_TOKEN = os.environ["OPERATOR_CONSOLE_TOKEN"]
DB_USER = os.environ["POSTGRES_USER"]
DB_PASSWORD = os.environ["POSTGRES_PASSWORD"]
COSTA = uuid.UUID("00000000-0000-4000-8000-000000000101")
SIERRA = uuid.UUID("00000000-0000-4000-8000-000000000102")
DESTINATION = uuid.UUID("00000000-0000-4000-8000-000000000013")
CLEARING = uuid.UUID("00000000-0000-4000-8000-000000000012")
PREFIX = "phase20-outage-"
REVERSE_PREFIX = "phase20-reverse3-"


def http_json(method, url, body=None, token=None):
    encoded = json.dumps(body).encode() if body is not None else None
    request = urllib.request.Request(url, data=encoded, method=method)
    if encoded is not None:
        request.add_header("Content-Type", "application/json")
    if token is not None:
        request.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(request, timeout=5) as response:
            payload = response.read()
            return response.status, json.loads(payload) if payload else {}
    except urllib.error.HTTPError as error:
        payload = error.read()
        return error.code, json.loads(payload) if payload else {}


def db_query(host, dbname, query, params=()):
    with psycopg.connect(host=host, dbname=dbname, user=DB_USER, password=DB_PASSWORD) as connection:
        with connection.cursor() as cursor:
            cursor.execute(query, params)
            return cursor.fetchall()


def wait_ready():
    deadline = time.time() + 120
    while time.time() < deadline:
        status, _ = http_json("GET", f"{NETWORK_URL}/health/ready")
        if status == 200:
            return
        time.sleep(2)
    raise AssertionError("CholoNet no estuvo disponible dentro del tiempo límite")


def transfer():
    rows = db_query(
        "network-db", "cholopay_network",
        "SELECT transfer_id,idempotency_key FROM transfer_instruction "
        "WHERE idempotency_key LIKE %s ORDER BY created_at DESC LIMIT 1",
        (PREFIX + "%",),
    )
    assert rows, "No existe la transferencia preparada de Phase 20"
    return rows[0]


def prepare():
    wait_ready()
    transfer_id = uuid.uuid4()
    correlation_id = uuid.uuid4()
    key = PREFIX + str(uuid.uuid4())
    body = {
        "transfer_id": str(transfer_id), "idempotency_key": key,
        "source_participant_id": str(COSTA), "destination_participant_id": str(SIERRA),
        "destination_account_id": str(DESTINATION), "clearing_account_id": str(CLEARING),
        "amount_minor": 125, "currency": "USD", "correlation_id": str(correlation_id),
    }
    status, _ = http_json("POST", f"{NETWORK_URL}/internal/v1/transfers", body)
    assert status == 202, ("crear transferencia", status)
    for state in ("VALIDATING", "FUNDS_RESERVED", "READY_FOR_SETTLEMENT"):
        status, _ = http_json(
            "POST", f"{NETWORK_URL}/internal/v1/transfers/{transfer_id}/transitions",
            {"state": state, "causation_id": str(uuid.uuid4())},
        )
        assert status == 204, (state, status)
    print(f"PHASE20 PREPARE PASS: transfer_id={transfer_id} idempotency_key={key}")


def reverse_transfers():
    rows = db_query(
        "network-db", "cholopay_network",
        "SELECT transfer_id,idempotency_key,amount_minor FROM ("
        "SELECT transfer_id,idempotency_key,amount_minor,created_at FROM transfer_instruction "
        "WHERE idempotency_key LIKE %s ORDER BY created_at DESC LIMIT 2"
        ") recent ORDER BY created_at",
        (REVERSE_PREFIX + "%",),
    )
    assert len(rows) == 2, rows
    # The second prepared transfer is settled first, making arrival order explicit and reverse
    # relative to preparation. The transfers are independent; no event ordering is asserted.
    for transfer_id, key, amount in reversed(rows):
        correlation_id = db_query(
            "network-db", "cholopay_network",
            "SELECT correlation_id FROM transfer_instruction WHERE transfer_id=%s", (transfer_id,),
        )[0][0]
        status, body = http_json(
            "POST", f"{NETWORK_URL}/internal/v1/settlements",
            {"transfer_id": str(transfer_id), "source_participant_id": str(COSTA),
             "destination_participant_id": str(SIERRA), "destination_account_id": str(DESTINATION),
             "clearing_account_id": str(CLEARING), "amount_minor": amount,
             "idempotency_key": key + "-settlement", "correlation_id": str(correlation_id)},
        )
        assert status == 201 and body.get("state") == "SETTLED", (status, body)
    print(f"PHASE20 REVERSE SETTLE PASS: transfers={[str(row[0]) for row in rows]}")


def prepare_reverse():
    wait_ready()
    for amount in (125, 200):
        transfer_id = uuid.uuid4()
        correlation_id = uuid.uuid4()
        key = REVERSE_PREFIX + str(uuid.uuid4())
        body = {
            "transfer_id": str(transfer_id), "idempotency_key": key,
            "source_participant_id": str(COSTA), "destination_participant_id": str(SIERRA),
            "destination_account_id": str(DESTINATION), "clearing_account_id": str(CLEARING),
            "amount_minor": amount, "currency": "USD", "correlation_id": str(correlation_id),
        }
        status, _ = http_json("POST", f"{NETWORK_URL}/internal/v1/transfers", body)
        assert status == 202, ("crear transferencia", status)
        for state in ("VALIDATING", "FUNDS_RESERVED", "READY_FOR_SETTLEMENT"):
            status, _ = http_json("POST", f"{NETWORK_URL}/internal/v1/transfers/{transfer_id}/transitions", {"state": state, "causation_id": str(uuid.uuid4())})
            assert status == 204, (state, status)
    print("PHASE20 REVERSE PREPARE PASS: two independent transfers")


def complete_reverse():
    rows = db_query(
        "network-db", "cholopay_network",
        "SELECT transfer_id FROM (SELECT transfer_id,created_at FROM transfer_instruction "
        "WHERE idempotency_key LIKE %s ORDER BY created_at DESC LIMIT 2) recent ORDER BY created_at",
        (REVERSE_PREFIX + "%",),
    )
    assert len(rows) == 2, rows
    transfer_ids = [row[0] for row in rows]
    deadline = time.time() + 120
    while time.time() < deadline:
        deliveries = db_query("network-db", "cholopay_network", "SELECT transfer_id, state FROM settlement_delivery WHERE transfer_id = ANY(%s)", (transfer_ids,))
        states = {row[0]: row[1] for row in deliveries}
        instructions = db_query("network-db", "cholopay_network", "SELECT transfer_id, state FROM transfer_instruction WHERE transfer_id = ANY(%s)", (transfer_ids,))
        if len(deliveries) == 2 and all(states.get(t) == "APPLIED" for t in transfer_ids) and all(dict(instructions).get(t) == "COMPLETED" for t in transfer_ids):
            delivery_counts = db_query("network-db", "cholopay_network", "SELECT COUNT(DISTINCT d.delivery_id), COUNT(DISTINCT d.delivery_id), COUNT(DISTINCT d.event_id), COALESCE(SUM(CASE WHEN e.direction='DEBIT' THEN e.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN e.direction='CREDIT' THEN e.amount_minor ELSE 0 END),0) FROM settlement_delivery d JOIN settlement_entry e ON e.journal_id=(SELECT journal_id FROM settlement_journal WHERE transfer_id=d.transfer_id) WHERE d.transfer_id = ANY(%s)", (transfer_ids,))[0]
            assert delivery_counts == (2, 2, 2, 325, 325), delivery_counts
            inbox = db_query("sierra-db", "cholopay_sierra", "SELECT COUNT(*), COUNT(DISTINCT delivery_id), COUNT(DISTINCT journal_id) FROM receiver_credit_inbox WHERE transfer_id = ANY(%s)", (transfer_ids,))[0]
            totals = db_query("sierra-db", "cholopay_sierra", "SELECT COUNT(DISTINCT j.id), COALESCE(SUM(CASE WHEN e.direction='DEBIT' THEN e.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN e.direction='CREDIT' THEN e.amount_minor ELSE 0 END),0) FROM journal j JOIN journal_entry e ON e.journal_id=j.id WHERE j.transaction_id = ANY(%s)", (transfer_ids,))[0]
            assert inbox == (2, 2, 2), inbox
            assert totals == (2, 325, 325), totals
            balances = db_query("sierra-db", "cholopay_sierra", "SELECT id, posted_balance_minor-reserved_balance_minor FROM ledger_account WHERE id IN (%s,%s)", (DESTINATION, CLEARING))
            balance_values = [value for _, value in balances]
            assert len(balance_values) == 2 and sum(balance_values) == 1000000 and max(balance_values) >= 325, balances
            print(f"PHASE20 REVERSE COMPLETE PASS: transfers={len(transfer_ids)} deliveries=2 inbox=2 journals=2 balanced=325/325 balances={balances}")
            return
        time.sleep(2)
    raise AssertionError(f"No se completaron las entregas inversas: instructions={instructions} deliveries={deliveries}")


def settle():
    transfer_id, key = transfer()
    correlation_id = db_query(
        "network-db", "cholopay_network",
        "SELECT correlation_id FROM transfer_instruction WHERE transfer_id=%s", (transfer_id,),
    )[0][0]
    status, body = http_json(
        "POST", f"{NETWORK_URL}/internal/v1/settlements",
        {"transfer_id": str(transfer_id), "source_participant_id": str(COSTA),
         "destination_participant_id": str(SIERRA), "destination_account_id": str(DESTINATION),
         "clearing_account_id": str(CLEARING), "amount_minor": 125,
         "idempotency_key": key + "-settlement", "correlation_id": str(correlation_id)},
    )
    assert status == 201 and body.get("state") == "SETTLED", (status, body)
    print(f"PHASE20 SETTLE PASS: transfer_id={transfer_id} receptor no disponible esperado")


def observe():
    transfer_id, _ = transfer()
    deadline = time.time() + 45
    while time.time() < deadline:
        rows = db_query(
            "network-db", "cholopay_network",
            "SELECT attempt_count,failure_detail,next_attempt_at,state FROM settlement_delivery WHERE transfer_id=%s",
            (transfer_id,),
        )
        if rows and rows[0][0] > 1 and rows[0][1] and rows[0][2] is not None and rows[0][2].timestamp() > time.time() and rows[0][3] == "PENDING":
            print(f"PHASE20 OBSERVE PASS: transfer_id={transfer_id} retry/backoff={rows[0]}")
            return
        time.sleep(1)
    raise AssertionError(f"No se observó retry/backoff real para {transfer_id}: {rows}")


def exhaust():
    transfer_id, _ = transfer()
    deadline = time.time() + 120
    while time.time() < deadline:
        rows = db_query("network-db", "cholopay_network", "SELECT state,failure_code,attempt_count FROM settlement_delivery WHERE transfer_id=%s", (transfer_id,))
        if rows and rows[0] == ("FAILED", "DELIVERY_DLQ", rows[0][2]):
            assert rows[0][2] >= 5
            print(f"PHASE20 DLQ PASS: transfer_id={transfer_id} state={rows[0][0]} failure_code={rows[0][1]} attempts={rows[0][2]}")
            return
        time.sleep(2)
    raise AssertionError(f"No se alcanzó DELIVERY_DLQ para {transfer_id}: {rows}")


def replay():
    transfer_id, _ = transfer()
    delivery_id, event_id, state, attempts = db_query("network-db", "cholopay_network", "SELECT delivery_id,event_id,state,attempt_count FROM settlement_delivery WHERE transfer_id=%s", (transfer_id,))[0]
    status, body = http_json("POST", f"{NETWORK_URL}/internal/v1/settlement-deliveries/{delivery_id}/replay", token=OPERATOR_TOKEN)
    assert status == 200 and body["state"] == "PENDING" and uuid.UUID(body["delivery_id"]) == delivery_id and uuid.UUID(body["event_id"]) == event_id and body["attempt_count"] == attempts, (status, body)
    status2, body2 = http_json("POST", f"{NETWORK_URL}/internal/v1/settlement-deliveries/{delivery_id}/replay", token=OPERATOR_TOKEN)
    assert status2 == 200 and body2 == body, (status2, body2)
    print(f"PHASE20 REPLAY PASS: delivery_id={delivery_id} event_id={event_id} attempts={attempts}")


def reconcile(expected_code=None, scoped=True):
    transfer_id, _ = transfer()
    query = f"?transfer_id={transfer_id}" if scoped else ""
    status, body = http_json("POST", f"{NETWORK_URL}/internal/v1/reconciliation/runs{query}")
    assert status == 201, (status, body)
    differences = body.get("Differences", body.get("differences", [])) or []
    target_differences = [
        item for item in differences
        if (item.get("TransferID") or item.get("transfer_id")) == str(transfer_id)
    ]
    codes = [item["Code"] if "Code" in item else item.get("code") for item in target_differences]
    if expected_code:
        assert expected_code in codes, (expected_code, transfer_id, target_differences)
    elif not scoped:
        assert not differences, (transfer_id, differences)
    else:
        assert not codes, (transfer_id, target_differences)
    run_id = body.get("RunID", body.get("run_id"))
    print(f"PHASE20 RECONCILIATION PASS: run_id={run_id} scoped={scoped} transfer_id={transfer_id} expected={expected_code or 'clean'}")


def historical_reconciliation():
    transfer_id, _ = transfer()
    first = db_query(
        "network-db", "cholopay_network",
        "SELECT r.run_id,d.difference_id,d.transfer_id,d.delivery_id,d.code,d.detail "
        "FROM reconciliation_run r JOIN reconciliation_difference d ON d.run_id=r.run_id "
        "WHERE d.transfer_id=%s AND d.code='DELIVERY_DLQ' ORDER BY r.started_at DESC LIMIT 1",
        (transfer_id,),
    )
    assert len(first) == 1, first
    first_run_id, first_difference_id, first_transfer_id, first_delivery_id, first_code, first_detail = first[0]
    delivery_row = db_query(
        "network-db", "cholopay_network",
        "SELECT delivery_id,event_id FROM settlement_delivery WHERE transfer_id=%s", (transfer_id,),
    )
    assert len(delivery_row) == 1 and delivery_row[0][0] == first_delivery_id
    delivery_id, event_id = delivery_row[0]
    status, body = http_json("GET", f"{NETWORK_URL}/internal/v1/reconciliation/runs/{first_run_id}")
    assert status == 200, (status, body)
    differences = body.get("Differences", body.get("differences", [])) or []
    assert body.get("RunID", body.get("run_id")) == str(first_run_id)
    assert body.get("DifferenceCount", body.get("difference_count")) == 1 and len(differences) == 1, body
    observed = differences[0]
    assert (observed.get("DifferenceID") or observed.get("difference_id")) == str(first_difference_id)
    assert (observed.get("TransferID") or observed.get("transfer_id")) == str(first_transfer_id)
    assert (observed.get("DeliveryID") or observed.get("delivery_id")) == str(first_delivery_id)
    assert (observed.get("Code") or observed.get("code")) == first_code
    assert (observed.get("Detail") or observed.get("detail")) == first_detail
    clean_runs = db_query(
        "network-db", "cholopay_network",
        "SELECT run_id,difference_count FROM reconciliation_run WHERE run_id<>%s ORDER BY started_at DESC LIMIT 1",
        (first_run_id,),
    )
    assert len(clean_runs) == 1 and clean_runs[0][1] == 0, clean_runs
    assert delivery_id == delivery_row[0][0] and event_id == db_query(
        "network-db", "cholopay_network", "SELECT event_id FROM settlement_delivery WHERE delivery_id=%s", (delivery_id,)
    )[0][0]
    print(f"PHASE20 HISTORICAL RECONCILIATION PASS: first_run_id={first_run_id} difference_id={first_difference_id} transfer_id={transfer_id} delivery_id={delivery_id} event_id={event_id} original=DELIVERY_DLQ unchanged_after_clean_run={clean_runs[0][0]}")


def complete():
    transfer_id, _ = transfer()
    deadline = time.time() + 120
    while time.time() < deadline:
        delivery = db_query(
            "network-db", "cholopay_network",
            "SELECT delivery_id,event_id,state,attempt_count FROM settlement_delivery WHERE transfer_id=%s",
            (transfer_id,),
        )
        instruction = db_query(
            "network-db", "cholopay_network",
            "SELECT state FROM transfer_instruction WHERE transfer_id=%s", (transfer_id,),
        )
        if delivery and instruction and delivery[0][2] == "APPLIED" and instruction[0][0] == "COMPLETED":
            delivery_id, event_id, state, attempts = delivery[0]
            assert attempts > 1
            inbox_row = db_query(
                "sierra-db", "cholopay_sierra",
                "SELECT delivery_id,event_id,journal_id FROM receiver_credit_inbox WHERE delivery_id=%s",
                (delivery_id,),
            )
            assert len(inbox_row) == 1 and inbox_row[0][0] == delivery_id and inbox_row[0][1] == event_id
            inbox, journals = 1, 1
            journal_rows = db_query(
                "sierra-db", "cholopay_sierra",
                "SELECT journal_id, COUNT(*), COALESCE(SUM(CASE WHEN direction='DEBIT' THEN amount_minor ELSE 0 END),0), "
                "COALESCE(SUM(CASE WHEN direction='CREDIT' THEN amount_minor ELSE 0 END),0) "
                "FROM journal_entry WHERE journal_id IN (SELECT journal_id FROM receiver_credit_inbox WHERE delivery_id=%s) GROUP BY journal_id",
                (delivery_id,),
            )
            assert (inbox, journals) == (1, 1) and len(journal_rows) == 1
            assert journal_rows[0][0] == inbox_row[0][2]
            assert journal_rows[0][1:] == (2, 125, 125), journal_rows
            settlement_totals = db_query(
                "network-db", "cholopay_network",
                "SELECT COUNT(*), COALESCE(SUM(CASE WHEN direction='DEBIT' THEN amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN direction='CREDIT' THEN amount_minor ELSE 0 END),0) FROM settlement_entry WHERE journal_id=(SELECT journal_id FROM settlement_journal WHERE transfer_id=%s)",
                (transfer_id,),
            )[0]
            assert settlement_totals == (2, 125, 125), settlement_totals
            receiver_balances = db_query(
                "sierra-db", "cholopay_sierra",
                "SELECT id, posted_balance_minor-reserved_balance_minor FROM ledger_account WHERE id IN (%s,%s)",
                (DESTINATION, CLEARING),
            )
            balance_values = dict(receiver_balances)
            assert set(balance_values) == {DESTINATION, CLEARING}
            assert balance_values[DESTINATION] >= 125 and balance_values[CLEARING] >= 0
            assert balance_values[DESTINATION] + balance_values[CLEARING] == 1000000, receiver_balances
            stable = db_query(
                "network-db", "cholopay_network",
                "SELECT COUNT(*), COUNT(DISTINCT delivery_id), COUNT(DISTINCT event_id), COUNT(*) FILTER (WHERE state='APPLIED') FROM settlement_delivery WHERE transfer_id=%s",
                (transfer_id,),
            )[0]
            assert stable == (1, 1, 1, 1), stable
            print(f"PHASE20 COMPLETE PASS: transfer_id={transfer_id} delivery_id={delivery_id} event_id={event_id} inbox=1 journal=1 balanced=125/125")
            return
        time.sleep(2)
    raise AssertionError(f"No se completó la recuperación para {transfer_id}: instruction={instruction} delivery={delivery}")


def main():
    modes = {"prepare": prepare, "settle": settle, "observe": observe, "exhaust": exhaust, "replay": replay, "reconcile-dlq": lambda: reconcile("DELIVERY_DLQ"), "complete": complete, "reconcile-clean": lambda: reconcile(scoped=False), "historical-reconciliation": historical_reconciliation, "prepare-reverse": prepare_reverse, "settle-reverse": reverse_transfers, "complete-reverse": complete_reverse}
    mode = sys.argv[1] if len(sys.argv) > 1 else os.environ.get("PHASE20_MODE", "prepare")
    if mode not in modes:
        raise SystemExit(f"Modo inválido: {mode}; use prepare|settle|observe|exhaust|replay|reconcile-dlq|complete|reconcile-clean|historical-reconciliation|prepare-reverse|settle-reverse|complete-reverse")
    modes[mode]()


if __name__ == "__main__":
    main()
