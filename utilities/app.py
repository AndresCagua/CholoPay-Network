"""API sintética de facturación; solo biblioteca estándar y ejecución Docker-only."""
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


def response_payload(handler: BaseHTTPRequestHandler, status: int, payload: dict) -> None:
    body = json.dumps(payload, sort_keys=True).encode()
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json")
    handler.send_header("Content-Length", str(len(body)))
    handler.end_headers()
    handler.wfile.write(body)


def _intent_for_id(intent_id: str) -> dict | None:
    return next((record["intent"] for record in _payment_intents.values() if record["intent"]["id"] == intent_id), None)


class UtilitiesHandler(BaseHTTPRequestHandler):
    server_version = "CholoPay-Synthetic-Utilities/1.0"

    def log_message(self, format: str, *args) -> None:
        return

    def do_GET(self) -> None:
        parsed = urlparse(self.path)
        parts = [part for part in parsed.path.split("/") if part]
        query = parse_qs(parsed.query)
        api_prefix = parts[:1] == ["api"]
        route = parts[1:] if api_prefix else parts

        if parsed.path == "/health":
            return response_payload(self, 200, {"status": "ok", "synthetic": True})
        if route == ["v1", "services", "catalog"]:
            return response_payload(self, 200, {"services": SERVICES, "provider": "synthetic-multiservice-company"})
        if route == ["v1", "contracts"]:
            return response_payload(self, 200, {"contracts": list(CONTRACTS.values())})
        if len(route) == 3 and route[:2] == ["v1", "contracts"]:
            contract = CONTRACTS.get(route[2])
            return response_payload(self, 200, contract) if contract else response_payload(self, 404, {"error": "contract_not_found"})
        if len(route) == 4 and route[:2] == ["v1", "contracts"] and route[3] == "bills":
            contract = CONTRACTS.get(route[2])
            if not contract:
                return response_payload(self, 404, {"error": "contract_not_found"})
            return response_payload(self, 200, {"contract": contract, "bills": [bill for bill in INVOICES.values() if bill["contract_id"] == route[2]]})
        if route == ["v1", "invoices"]:
            invoices = list(INVOICES.values())
            if "contract_id" in query:
                invoices = [i for i in invoices if i["contract_id"] == query["contract_id"][0]]
            return response_payload(self, 200, {"invoices": invoices})
        if len(route) == 3 and route[:2] in (["v1", "invoices"], ["v1", "bills"]):
            item = INVOICES.get(route[2])
            return response_payload(self, 200, item) if item else response_payload(self, 404, {"error": "invoice_not_found"})
        if len(route) == 4 and route[:2] in (["v1", "invoices"], ["v1", "bills"]):
            item = INVOICES.get(route[2])
            if not item:
                return response_payload(self, 404, {"error": "invoice_not_found"})
            if route[3] == "collection-status":
                return response_payload(self, 200, {"invoice_id": item["id"], "collection_status": "pending", "invoice_status": item["status"], "last_attempt": None, "synthetic": True})
            if route[3] == "payment-status":
                return response_payload(self, 200, {"invoice_id": item["id"], "status": item["status"], "payment_intent_id": item.get("payment_intent_id"), "settlement_state": item.get("settlement_state", "NOT_SETTLED"), "synthetic": True})
        return response_payload(self, 404, {"error": "not_found"})

    def do_POST(self) -> None:
        parsed = urlparse(self.path)
        parts = [part for part in parsed.path.split("/") if part]
        if parsed.path == "/api/v1/internal/payment-events":
            return self.apply_payment_event()
        if parsed.path == "/api/v1/payment-intents":
            bill_id = None
        elif len(parts) == 4 and parts[:2] == ["v1", "bills"] and parts[3] == "payment-intents":
            bill_id = parts[2]
        else:
            return response_payload(self, 404, {"error": "not_found"})
        return self.create_payment_intent(bill_id)

    def create_payment_intent(self, bill_id: str | None) -> None:
        key = self.headers.get("Idempotency-Key", "").strip()
        if not key:
            return response_payload(self, 400, {"error": "idempotency_key_required"})
        try:
            length = int(self.headers.get("Content-Length", "0"))
            payload = json.loads(self.rfile.read(length) or b"{}")
        except (ValueError, json.JSONDecodeError):
            return response_payload(self, 400, {"error": "invalid_json"})
        invoice_id = bill_id or payload.get("invoice_id")
        if invoice_id not in INVOICES:
            return response_payload(self, 404, {"error": "invoice_not_found"})
        request_payload = {**payload, **({"invoice_id": invoice_id} if bill_id else {})}
        fingerprint = hashlib.sha256(json.dumps(request_payload, sort_keys=True).encode()).hexdigest()
        with _payment_lock:
            existing = _payment_intents.get(key)
            if existing:
                if existing["request_fingerprint"] != fingerprint:
                    return response_payload(self, 409, {"error": "idempotency_key_reused_with_different_request"})
                return response_payload(self, 200, existing["intent"])
            invoice = INVOICES[invoice_id]
            if bill_id and invoice.get("payment_intent_id") and invoice["status"] != "paid":
                return response_payload(self, 409, {"error": "payment_already_in_progress"})
            currency = payload.get("currency", invoice["currency"])
            if currency != "USD":
                return response_payload(self, 400, {"error": "currency_must_be_USD"})
            intent = {"id": "pi_" + uuid.uuid4().hex, "invoice_id": invoice_id, "amount": payload.get("amount", invoice["amount"]), "currency": currency, "status": "pending", "debit_performed": False, "ledger_entry_created": False, "synthetic": True}
            _payment_intents[key] = {"request_fingerprint": fingerprint, "intent": intent}
            if bill_id:
                invoice["payment_intent_id"] = intent["id"]
            invoice["settlement_state"] = "AWAITING_APPLICATION"
        return response_payload(self, 201, intent)

    def apply_payment_event(self) -> None:
        event_id = self.headers.get("X-CholoPay-Event-Id", "").strip()
        signature = self.headers.get("X-CholoPay-Signature", "").strip()
        timestamp = self.headers.get("X-CholoPay-Timestamp", "").strip()
        if not event_id or not signature or not timestamp:
            return response_payload(self, 400, {"error": "event_headers_required"})
        try:
            length = int(self.headers.get("Content-Length", "0"))
            raw = self.rfile.read(length)
            payload = json.loads(raw or b"{}")
        except (ValueError, json.JSONDecodeError):
            return response_payload(self, 400, {"error": "invalid_json"})
        canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"))
        expected = hmac.new(_INTERNAL_EVENT_SECRET.encode(), f"{timestamp}.{canonical}".encode(), hashlib.sha256).hexdigest()
        if not hmac.compare_digest(signature, expected):
            return response_payload(self, 401, {"error": "invalid_event_signature"})
        invoice_id = payload.get("invoice_id") or payload.get("bill_id")
        intent_id = payload.get("payment_intent_id")
        if payload.get("event_type") != "payment.settled" or not isinstance(intent_id, str) or not isinstance(invoice_id, str):
            return response_payload(self, 400, {"error": "invalid_payment_event"})
        with _payment_lock:
            fingerprint = hashlib.sha256(canonical.encode()).hexdigest()
            existing = _payment_events.get(event_id)
            if existing:
                if existing["fingerprint"] != fingerprint:
                    return response_payload(self, 409, {"error": "event_id_reused_with_different_payload"})
                return response_payload(self, 200, existing["receipt"])
            invoice = INVOICES.get(invoice_id)
            intent = _intent_for_id(intent_id)
            if not invoice:
                return response_payload(self, 404, {"error": "invoice_not_found"})
            if not intent or intent["invoice_id"] != invoice_id:
                return response_payload(self, 409, {"error": "payment_intent_not_bound_to_invoice"})
            if invoice.get("payment_intent_id") not in (None, intent_id):
                return response_payload(self, 409, {"error": "invoice_payment_intent_conflict"})
            invoice["payment_intent_id"] = intent_id
            invoice["status"] = "paid"
            invoice["settlement_state"] = "APPLIED"
            intent["status"] = "settled"
            receipt = {"event_id": event_id, "invoice_id": invoice_id, "payment_intent_id": intent_id, "status": "PAID", "debit_performed": False, "ledger_entry_created": False, "synthetic": True}
            _payment_events[event_id] = {"fingerprint": fingerprint, "receipt": receipt}
        return response_payload(self, 200, receipt)


def main() -> None:
    host = os.getenv("HOST", "127.0.0.1")
    port = int(os.getenv("PORT", "8080"))
    ThreadingHTTPServer((host, port), UtilitiesHandler).serve_forever()


if __name__ == "__main__":
    main()
