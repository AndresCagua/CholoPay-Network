"""Docker-only P6 probe; run once before and once after recreating utilities."""
import hashlib, hmac, json, os, urllib.request
import psycopg

BASE = os.getenv("UTILITIES_URL", "http://utilities:8080")
SECRET = os.getenv("INTERNAL_EVENT_SECRET", "local-test-only"); KEY = "p6-electricity-intent-001"; EVENT_ID = "p6-electricity-event-001"; TS = "1700000000"


def require(condition, message):
    if not condition:
        raise AssertionError(message)

def call(method, path, body=None, headers=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(BASE + path, data=data, method=method, headers={"Content-Type":"application/json", **(headers or {})})
    with urllib.request.urlopen(req) as response: return response.status, json.loads(response.read())

status, invoice = call("GET", "/api/v1/invoices/invoice-elec-2025-001")
require(status == 200 and invoice["service"] == "electricity", f"invoice unavailable: {status} {invoice}")
status, first = call("POST", "/api/v1/payment-intents", {"invoice_id": invoice["id"], "amount": "42.50", "currency": "USD"}, {"Idempotency-Key": KEY})
require(status in (201, 200), f"intent creation failed: {status} {first}")
status, replay = call("POST", "/api/v1/payment-intents", {"invoice_id": invoice["id"], "amount": "42.50", "currency": "USD"}, {"Idempotency-Key": KEY})
require(status == 200 and replay["id"] == first["id"], f"intent replay failed: {status} {replay}")
event = {"event_type":"payment.settled", "invoice_id":invoice["id"], "payment_intent_id":first["id"]}
canonical = json.dumps(event, sort_keys=True, separators=(",", ":"))
signature = hmac.new(SECRET.encode(), f"{TS}.{canonical}".encode(), hashlib.sha256).hexdigest()
eh = {"X-CholoPay-Event-Id": EVENT_ID, "X-CholoPay-Timestamp": TS, "X-CholoPay-Signature": signature}
status, receipt = call("POST", "/api/v1/internal/payment-events", event, eh)
require(status == 200 and receipt["status"] == "PAID", f"event application failed: {status} {receipt}")
status, replay_event = call("POST", "/api/v1/internal/payment-events", event, eh)
require(status == 200 and replay_event == receipt, f"event replay failed: {status} {replay_event}")
status, state = call("GET", "/api/v1/invoices/invoice-elec-2025-001/payment-status")
require(status == 200 and state["status"] == "paid" and state["settlement_state"] == "APPLIED" and state["payment_intent_id"] == first["id"], f"payment state invalid: {status} {state}")
with psycopg.connect(os.environ["UTILITIES_DB_URL"]) as db:
    with db.cursor() as cur:
        cur.execute("SELECT count(*) FROM utilities_payment_intent WHERE idempotency_key=%s", (KEY,)); require(cur.fetchone()[0] == 1, "intent count is not one")
        cur.execute("SELECT count(*) FROM utilities_payment_event WHERE event_id=%s", (EVENT_ID,)); require(cur.fetchone()[0] == 1, "event count is not one")
print("P6 utilities recovery acceptance: PASS")
