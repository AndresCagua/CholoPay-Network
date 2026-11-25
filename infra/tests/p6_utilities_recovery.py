"""Docker-only P6 probe; run once before and once after recreating utilities."""
import hashlib, hmac, json, os, urllib.request
import psycopg

BASE = os.getenv("UTILITIES_URL", "http://utilities:8080")
SECRET = os.getenv("INTERNAL_EVENT_SECRET", "local-test-only")
TS = "1700000000"


def require(condition, message):
    if not condition:
        raise RuntimeError(f"P6 utilities validation failed: {message}")


def call(method, path, body=None, headers=None):
    data = json.dumps(body).encode() if body is not None else None
    request = urllib.request.Request(
        BASE + path, data=data, method=method,
        headers={"Content-Type": "application/json", **(headers or {})},
    )
    with urllib.request.urlopen(request) as response:
        return response.status, json.loads(response.read())


def exercise_invoice(invoice_id, key, event_id):
    status, invoice = call("GET", f"/api/v1/invoices/{invoice_id}")
    require(status == 200 and invoice["id"] == invoice_id, f"{invoice_id} unavailable: {status} {invoice}")
    status, first = call(
        "POST", "/api/v1/payment-intents",
        {"invoice_id": invoice_id, "amount": invoice["amount"], "currency": "USD"},
        {"Idempotency-Key": key},
    )
    require(status in (201, 200), f"{invoice_id} intent creation/replay failed: {status} {first}")
    status, replay = call(
        "POST", "/api/v1/payment-intents",
        {"invoice_id": invoice_id, "amount": invoice["amount"], "currency": "USD"},
        {"Idempotency-Key": key},
    )
    require(status == 200 and replay["id"] == first["id"], f"{invoice_id} intent replay failed: {status} {replay}")
    event = {"event_type": "payment.settled", "invoice_id": invoice_id, "payment_intent_id": first["id"]}
    canonical = json.dumps(event, sort_keys=True, separators=(",", ":"))
    signature = hmac.new(SECRET.encode(), f"{TS}.{canonical}".encode(), hashlib.sha256).hexdigest()
    event_headers = {"X-CholoPay-Event-Id": event_id, "X-CholoPay-Timestamp": TS, "X-CholoPay-Signature": signature}
    status, receipt = call("POST", "/api/v1/internal/payment-events", event, event_headers)
    require(status == 200 and receipt["status"] == "PAID", f"{invoice_id} event application failed: {status} {receipt}")
    status, replay_event = call("POST", "/api/v1/internal/payment-events", event, event_headers)
    require(status == 200 and replay_event == receipt, f"{invoice_id} event replay failed: {status} {replay_event}")
    status, state = call("GET", f"/api/v1/invoices/{invoice_id}/payment-status")
    require(status == 200 and state["status"] == "paid" and state["settlement_state"] == "APPLIED" and state["payment_intent_id"] == first["id"], f"{invoice_id} state invalid: {status} {state}")
    return first["id"]


water_intent = exercise_invoice("invoice-water-2025-001", "p6-water-intent-001", "p6-water-event-001")
status, electricity = call("GET", "/api/v1/invoices/invoice-elec-2025-001")
require(status == 200 and electricity["service"] == "electricity", f"electricity invoice unavailable: {status} {electricity}")
status, water = call("GET", "/api/v1/invoices/invoice-water-2025-001/payment-status")
require(status == 200 and water["payment_intent_id"] == water_intent, f"water identity/state unavailable: {status} {water}")
with psycopg.connect(os.environ["UTILITIES_DB_URL"]) as db:
    with db.cursor() as cur:
        cur.execute("SELECT count(*) FROM utilities_payment_intent WHERE idempotency_key=%s", ("p6-water-intent-001",))
        require(cur.fetchone()[0] == 1, "water durable intent count is not one")
        cur.execute("SELECT count(*) FROM utilities_payment_event WHERE event_id=%s", ("p6-water-event-001",))
        require(cur.fetchone()[0] == 1, "water durable event count is not one")
print("P6 utilities recovery acceptance: PASS (water and electricity synthetic slices)")
