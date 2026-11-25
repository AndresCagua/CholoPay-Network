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


def exercise_invoice(invoice_id, key, event_id, use_bill_route=False):
    status, invoice = call("GET", f"/api/v1/invoices/{invoice_id}")
    require(status == 200 and invoice["id"] == invoice_id, f"{invoice_id} unavailable: {status} {invoice}")
    intent_path = f"/v1/bills/{invoice_id}/payment-intents" if use_bill_route else "/api/v1/payment-intents"
    intent_body = {"amount": invoice["amount"], "currency": "USD"} if use_bill_route else {"invoice_id": invoice_id, "amount": invoice["amount"], "currency": "USD"}
    status, first = call(
        "POST", intent_path, intent_body,
        {"Idempotency-Key": key},
    )
    require(status in (201, 200), f"{invoice_id} intent creation/replay failed: {status} {first}")
    status, replay = call(
        "POST", intent_path, intent_body,
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


internet_intent = exercise_invoice("bill-internet-2025-001", "p6-internet-intent-001", "p6-internet-event-001", use_bill_route=True)
mobile_intent = exercise_invoice("bill-mobile-2025-001", "p6-mobile-intent-001", "p6-mobile-event-001", use_bill_route=True)
water_intent = exercise_invoice("invoice-water-2025-001", "p6-water-intent-001", "p6-water-event-001")
status, catalog = call("GET", "/api/v1/services/catalog")
require(status == 200 and [item["id"] for item in catalog["services"]] == ["electricity", "water", "internet", "mobile"], f"catalog is incomplete: {status} {catalog}")
status, electricity = call("GET", "/api/v1/invoices/invoice-elec-2025-001")
require(status == 200 and electricity["service"] == "electricity", f"electricity invoice unavailable: {status} {electricity}")
status, internet = call("GET", "/api/v1/invoices/bill-internet-2025-001")
require(status == 200 and internet["service"] == "internet", f"internet legacy invoice unavailable: {status} {internet}")
status, internet_state = call("GET", "/v1/bills/bill-internet-2025-001/payment-status")
require(status == 200 and internet_state["payment_intent_id"] == internet_intent and internet_state["status"] == "paid" and internet_state["settlement_state"] == "APPLIED", f"internet identity/state unavailable: {status} {internet_state}")
status, mobile = call("GET", "/v1/bills/bill-mobile-2025-001/payment-status")
require(status == 200 and mobile["payment_intent_id"] == mobile_intent and mobile["status"] == "paid" and mobile["settlement_state"] == "APPLIED", f"mobile identity/state unavailable: {status} {mobile}")
status, water = call("GET", "/api/v1/invoices/invoice-water-2025-001/payment-status")
require(status == 200 and water["payment_intent_id"] == water_intent, f"water identity/state unavailable: {status} {water}")
with psycopg.connect(os.environ["UTILITIES_DB_URL"]) as db:
    with db.cursor() as cur:
        for key, event_id, label in (("p6-internet-intent-001", "p6-internet-event-001", "internet"), ("p6-mobile-intent-001", "p6-mobile-event-001", "mobile"), ("p6-water-intent-001", "p6-water-event-001", "water")):
            cur.execute("SELECT count(*) FROM utilities_payment_intent WHERE idempotency_key=%s", (key,))
            require(cur.fetchone()[0] == 1, f"{label} durable intent count is not one")
            cur.execute("SELECT count(*) FROM utilities_payment_event WHERE event_id=%s", (event_id,))
            require(cur.fetchone()[0] == 1, f"{label} durable event count is not one")
print("P6 utilities recovery acceptance: PASS (internet, mobile and water durable slices; electricity and all four catalog billers visible)")
