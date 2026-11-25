"""API sintética de facturación; electricidad persistida en PostgreSQL aislado."""
from __future__ import annotations

import hashlib
import hmac
import json
import os
import threading
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

SERVICES = [
    {"id": "electricity", "name": "electricidad", "currency": "USD", "synthetic": True},
    {"id": "water", "name": "agua potable", "currency": "USD", "synthetic": True},
    {"id": "internet", "name": "internet fijo", "currency": "USD", "synthetic": True},
    {"id": "mobile", "name": "telefonía móvil", "currency": "USD", "synthetic": True},
]
CONTRACTS = {
    "contract-elec-001": {"id": "contract-elec-001", "service": "electricity", "customer_ref": "synthetic-customer-001", "status": "active"},
    "contract-water-001": {"id": "contract-water-001", "service": "water", "customer_ref": "synthetic-customer-001", "status": "active"},
    "contract-internet-001": {"id": "contract-internet-001", "service": "internet", "customer_ref": "synthetic-customer-001", "status": "active"},
    "contract-mobile-001": {"id": "contract-mobile-001", "service": "mobile", "customer_ref": "synthetic-customer-001", "status": "active"},
}
INVOICES = {
    "invoice-elec-2025-001": {"id": "invoice-elec-2025-001", "contract_id": "contract-elec-001", "service": "electricity", "amount": "42.50", "currency": "USD", "due_date": "2025-12-31", "status": "open"},
    "invoice-water-2025-001": {"id": "invoice-water-2025-001", "contract_id": "contract-water-001", "service": "water", "amount": "18.00", "currency": "USD", "due_date": "2025-12-31", "status": "open"},
    "bill-internet-2025-001": {"id": "bill-internet-2025-001", "contract_id": "contract-internet-001", "service": "internet", "amount": "35.00", "currency": "USD", "due_date": "2025-12-31", "status": "open"},
    "bill-mobile-2025-001": {"id": "bill-mobile-2025-001", "contract_id": "contract-mobile-001", "service": "mobile", "amount": "22.00", "currency": "USD", "due_date": "2025-12-31", "status": "open"},
}

_payment_intents: dict[str, dict] = {}
_payment_events: dict[str, dict] = {}
_payment_lock = threading.Lock()
_INTERNAL_EVENT_SECRET = os.getenv("INTERNAL_EVENT_SECRET", "local-test-only")


class DurableStore:
    """Small PostgreSQL repository; only the electricity synthetic slice uses it."""
    def __init__(self) -> None:
        self.url = os.getenv("DB_URL", "")
        self.connection = None
        if self.url:
            import psycopg
            self.connection = psycopg.connect(self.url, autocommit=True)
            with self.connection.cursor() as cur:
                cur.execute("""
                    CREATE TABLE IF NOT EXISTS utilities_invoice (
                      id TEXT PRIMARY KEY, contract_id TEXT NOT NULL, service TEXT NOT NULL,
                      amount TEXT NOT NULL, currency TEXT NOT NULL, due_date TEXT NOT NULL,
                      status TEXT NOT NULL, payment_intent_id TEXT, settlement_state TEXT NOT NULL
                    );
                    CREATE TABLE IF NOT EXISTS utilities_payment_intent (
                      id TEXT PRIMARY KEY, idempotency_key TEXT UNIQUE NOT NULL, request_fingerprint TEXT NOT NULL,
                      invoice_id TEXT NOT NULL REFERENCES utilities_invoice(id), amount TEXT NOT NULL,
                      currency TEXT NOT NULL, status TEXT NOT NULL, debit_performed BOOLEAN NOT NULL,
                      ledger_entry_created BOOLEAN NOT NULL
                    );
                    CREATE TABLE IF NOT EXISTS utilities_payment_event (
                      event_id TEXT PRIMARY KEY, fingerprint TEXT UNIQUE NOT NULL, invoice_id TEXT NOT NULL,
                      payment_intent_id TEXT NOT NULL, receipt JSONB NOT NULL
                    );
                """)
                seed = INVOICES["invoice-elec-2025-001"]
                cur.execute("""INSERT INTO utilities_invoice
                    (id, contract_id, service, amount, currency, due_date, status, settlement_state)
                    VALUES (%s,%s,%s,%s,%s,%s,%s,%s) ON CONFLICT (id) DO NOTHING""",
                    (seed["id"], seed["contract_id"], seed["service"], seed["amount"], seed["currency"], seed["due_date"], seed["status"], "NOT_SETTLED"))

    @property
    def durable(self) -> bool:
        return self.connection is not None

    def invoice(self, invoice_id: str) -> dict | None:
        if not self.durable or invoice_id != "invoice-elec-2025-001":
            return INVOICES.get(invoice_id)
        with self.connection.cursor() as cur:
            cur.execute("SELECT id,contract_id,service,amount,currency,due_date,status,payment_intent_id,settlement_state FROM utilities_invoice WHERE id=%s", (invoice_id,))
            row = cur.fetchone()
        if not row: return None
        keys = ("id", "contract_id", "service", "amount", "currency", "due_date", "status", "payment_intent_id", "settlement_state")
        return {k: v for k, v in zip(keys, row) if v is not None}

    def invoices(self) -> list[dict]:
        return [self.invoice("invoice-elec-2025-001")] if self.durable else list(INVOICES.values())

    def save_intent(self, key, fingerprint, intent, invoice_id):
        if not self.durable: return None
        with self.connection.cursor() as cur:
            cur.execute("SELECT id,request_fingerprint,invoice_id,amount,currency,status,debit_performed,ledger_entry_created FROM utilities_payment_intent WHERE idempotency_key=%s", (key,))
            row = cur.fetchone()
            if row: return self._intent_row(row), row[1]
            cur.execute("INSERT INTO utilities_payment_intent VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)", (intent["id"], key, fingerprint, invoice_id, intent["amount"], intent["currency"], intent["status"], False, False))
            cur.execute("UPDATE utilities_invoice SET payment_intent_id=%s, settlement_state='AWAITING_APPLICATION' WHERE id=%s", (intent["id"], invoice_id))
        return None

    def intent_by_key(self, key):
        if not self.durable: return None
        with self.connection.cursor() as cur:
            cur.execute("SELECT id,request_fingerprint,invoice_id,amount,currency,status,debit_performed,ledger_entry_created FROM utilities_payment_intent WHERE idempotency_key=%s", (key,))
            row = cur.fetchone()
        return (self._intent_row(row), row[1]) if row else None

    def intent_by_id(self, intent_id):
        if not self.durable: return None
        with self.connection.cursor() as cur:
            cur.execute("SELECT id,request_fingerprint,invoice_id,amount,currency,status,debit_performed,ledger_entry_created FROM utilities_payment_intent WHERE id=%s", (intent_id,))
            row = cur.fetchone()
        return self._intent_row(row) if row else None

    def _intent_row(self, row):
        return {"id": row[0], "invoice_id": row[2], "amount": row[3], "currency": row[4], "status": row[5], "debit_performed": row[6], "ledger_entry_created": row[7], "synthetic": True}

    def apply_event(self, event_id, fingerprint, invoice_id, intent_id, receipt):
        if not self.durable: return None
        with self.connection.cursor() as cur:
            cur.execute("SELECT fingerprint,receipt FROM utilities_payment_event WHERE event_id=%s", (event_id,))
            row = cur.fetchone()
            if row: return (row[1], row[0])
            cur.execute("UPDATE utilities_invoice SET payment_intent_id=%s,status='paid',settlement_state='APPLIED' WHERE id=%s", (intent_id, invoice_id))
            cur.execute("UPDATE utilities_payment_intent SET status='settled' WHERE id=%s", (intent_id,))
            cur.execute("INSERT INTO utilities_payment_event VALUES (%s,%s,%s,%s,%s)", (event_id, fingerprint, invoice_id, intent_id, json.dumps(receipt)))
        return None

STORE = DurableStore()


def response_payload(handler, status, payload):
    body = json.dumps(payload, sort_keys=True).encode()
    handler.send_response(status); handler.send_header("Content-Type", "application/json"); handler.send_header("Content-Length", str(len(body))); handler.end_headers(); handler.wfile.write(body)


def _intent_for_id(intent_id):
    if STORE.durable: return STORE.intent_by_id(intent_id)
    return next((record["intent"] for record in _payment_intents.values() if record["intent"]["id"] == intent_id), None)


class UtilitiesHandler(BaseHTTPRequestHandler):
    server_version = "CholoPay-Synthetic-Utilities/1.0"
    def log_message(self, format, *args): return

    def do_GET(self):
        parsed = urlparse(self.path); parts = [p for p in parsed.path.split("/") if p]; query = parse_qs(parsed.query); api_prefix = parts[:1] == ["api"]; route = parts[1:] if api_prefix else parts
        if parsed.path in ("/health", "/health/live", "/health/ready"): return response_payload(self, 200, {"status":"ok", "synthetic":True})
        if route == ["v1", "services", "catalog"]: return response_payload(self, 200, {"services": SERVICES, "provider":"synthetic-multiservice-company"})
        if route == ["v1", "contracts"]: return response_payload(self, 200, {"contracts": list(CONTRACTS.values())})
        if len(route) == 3 and route[:2] == ["v1", "contracts"]:
            item = CONTRACTS.get(route[2]); return response_payload(self, 200, item) if item else response_payload(self, 404, {"error":"contract_not_found"})
        if len(route) == 4 and route[:2] == ["v1", "contracts"] and route[3] == "bills":
            contract = CONTRACTS.get(route[2]);
            if not contract: return response_payload(self, 404, {"error":"contract_not_found"})
            return response_payload(self, 200, {"contract":contract, "bills":[b for b in STORE.invoices() if b["contract_id"] == route[2]] if STORE.durable else [b for b in INVOICES.values() if b["contract_id"] == route[2]]})
        if route == ["v1", "invoices"]:
            invoices = STORE.invoices() if STORE.durable else list(INVOICES.values()); invoices = [i for i in invoices if not query.get("contract_id") or i["contract_id"] == query["contract_id"][0]]; return response_payload(self, 200, {"invoices":invoices})
        if len(route) >= 3 and route[:2] in (["v1","invoices"], ["v1","bills"]):
            item = STORE.invoice(route[2]);
            if not item: return response_payload(self, 404, {"error":"invoice_not_found"})
            if len(route) == 3: return response_payload(self, 200, item)
            if route[3] == "collection-status": return response_payload(self, 200, {"invoice_id":item["id"],"collection_status":"pending","invoice_status":item["status"],"last_attempt":None,"synthetic":True})
            if route[3] == "payment-status": return response_payload(self, 200, {"invoice_id":item["id"],"status":item["status"],"payment_intent_id":item.get("payment_intent_id"),"settlement_state":item.get("settlement_state","NOT_SETTLED"),"synthetic":True})
        return response_payload(self, 404, {"error":"not_found"})

    def do_POST(self):
        parsed = urlparse(self.path); parts = [p for p in parsed.path.split("/") if p]
        if parsed.path == "/api/v1/internal/payment-events": return self.apply_payment_event()
        if parsed.path == "/api/v1/payment-intents": bill_id = None
        elif len(parts) == 4 and parts[:2] == ["v1","bills"] and parts[3] == "payment-intents": bill_id = parts[2]
        else: return response_payload(self, 404, {"error":"not_found"})
        return self.create_payment_intent(bill_id)

    def create_payment_intent(self, bill_id):
        key = self.headers.get("Idempotency-Key", "").strip()
        if not key: return response_payload(self, 400, {"error":"idempotency_key_required"})
        try: payload = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))) or b"{}")
        except (ValueError, json.JSONDecodeError): return response_payload(self, 400, {"error":"invalid_json"})
        invoice_id = bill_id or payload.get("invoice_id"); invoice = STORE.invoice(invoice_id)
        if not invoice: return response_payload(self, 404, {"error":"invoice_not_found"})
        request_payload = {**payload, **({"invoice_id":invoice_id} if bill_id else {})}; fingerprint = hashlib.sha256(json.dumps(request_payload, sort_keys=True).encode()).hexdigest()
        with _payment_lock:
            existing = STORE.intent_by_key(key) if STORE.durable else _payment_intents.get(key)
            if existing:
                old, old_fp = existing if STORE.durable else (existing["intent"], existing["request_fingerprint"])
                if old_fp != fingerprint: return response_payload(self, 409, {"error":"idempotency_key_reused_with_different_request"})
                return response_payload(self, 200, old)
            if bill_id and invoice.get("payment_intent_id") and invoice["status"] != "paid": return response_payload(self, 409, {"error":"payment_already_in_progress"})
            currency = payload.get("currency", invoice["currency"])
            if currency != "USD": return response_payload(self, 400, {"error":"currency_must_be_USD"})
            intent = {"id":"pi_" + uuid.uuid4().hex,"invoice_id":invoice_id,"amount":payload.get("amount",invoice["amount"]),"currency":currency,"status":"pending","debit_performed":False,"ledger_entry_created":False,"synthetic":True}
            if STORE.durable: STORE.save_intent(key, fingerprint, intent, invoice_id)
            else:
                _payment_intents[key] = {"request_fingerprint":fingerprint,"intent":intent}; invoice["payment_intent_id"] = intent["id"]; invoice["settlement_state"] = "AWAITING_APPLICATION"
        return response_payload(self, 201, intent)

    def apply_payment_event(self):
        event_id = self.headers.get("X-CholoPay-Event-Id", "").strip(); signature = self.headers.get("X-CholoPay-Signature", "").strip(); timestamp = self.headers.get("X-CholoPay-Timestamp", "").strip()
        if not event_id or not signature or not timestamp: return response_payload(self, 400, {"error":"event_headers_required"})
        try: raw = self.rfile.read(int(self.headers.get("Content-Length", "0"))); payload = json.loads(raw or b"{}")
        except (ValueError, json.JSONDecodeError): return response_payload(self, 400, {"error":"invalid_json"})
        canonical = json.dumps(payload, sort_keys=True, separators=(",", ":")); expected = hmac.new(_INTERNAL_EVENT_SECRET.encode(), f"{timestamp}.{canonical}".encode(), hashlib.sha256).hexdigest()
        if not hmac.compare_digest(signature, expected): return response_payload(self, 401, {"error":"invalid_event_signature"})
        invoice_id = payload.get("invoice_id") or payload.get("bill_id"); intent_id = payload.get("payment_intent_id")
        if payload.get("event_type") != "payment.settled" or not isinstance(intent_id, str) or not isinstance(invoice_id, str): return response_payload(self, 400, {"error":"invalid_payment_event"})
        with _payment_lock:
            fingerprint = hashlib.sha256(canonical.encode()).hexdigest()
            if STORE.durable:
                with STORE.connection.cursor() as cur:
                    cur.execute("SELECT fingerprint,receipt FROM utilities_payment_event WHERE event_id=%s", (event_id,))
                    existing = cur.fetchone()
                if existing:
                    if existing[0] != fingerprint: return response_payload(self, 409, {"error":"event_id_reused_with_different_payload"})
                    return response_payload(self, 200, existing[1])
            else:
                existing = _payment_events.get(event_id)
                if existing:
                    if existing["fingerprint"] != fingerprint: return response_payload(self, 409, {"error":"event_id_reused_with_different_payload"})
                    return response_payload(self, 200, existing["receipt"])
            invoice = STORE.invoice(invoice_id); intent = _intent_for_id(intent_id)
            if not invoice: return response_payload(self, 404, {"error":"invoice_not_found"})
            if not intent or intent["invoice_id"] != invoice_id: return response_payload(self, 409, {"error":"payment_intent_not_bound_to_invoice"})
            if invoice.get("payment_intent_id") not in (None, intent_id): return response_payload(self, 409, {"error":"invoice_payment_intent_conflict"})
            receipt = {"event_id":event_id,"invoice_id":invoice_id,"payment_intent_id":intent_id,"status":"PAID","debit_performed":False,"ledger_entry_created":False,"synthetic":True}
            if STORE.durable:
                STORE.apply_event(event_id, fingerprint, invoice_id, intent_id, receipt)
            else:
                invoice["payment_intent_id"] = intent_id; invoice["status"] = "paid"; invoice["settlement_state"] = "APPLIED"; intent["status"] = "settled"; _payment_events[event_id] = {"fingerprint":fingerprint,"receipt":receipt}
        return response_payload(self, 200, receipt)


def main():
    ThreadingHTTPServer((os.getenv("HOST", "127.0.0.1"), int(os.getenv("PORT", "8080"))), UtilitiesHandler).serve_forever()
if __name__ == "__main__": main()
