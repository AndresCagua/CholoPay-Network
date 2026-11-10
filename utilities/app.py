"""Synthetic utilities biller API; standard library only."""
from __future__ import annotations

import hashlib
import json
import os
import threading
import uuid
from datetime import date
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

CONTRACTS = {
    "contract-elec-001": {
        "id": "contract-elec-001",
        "service": "electricity",
        "customer_ref": "synthetic-customer-001",
        "status": "active",
    },
    "contract-water-001": {
        "id": "contract-water-001",
        "service": "water",
        "customer_ref": "synthetic-customer-001",
        "status": "active",
    },
}

INVOICES = {
    "invoice-elec-2025-001": {
        "id": "invoice-elec-2025-001",
        "contract_id": "contract-elec-001",
        "service": "electricity",
        "amount": "42.50",
        "currency": "USD",
        "due_date": "2025-12-31",
        "status": "open",
    },
    "invoice-water-2025-001": {
        "id": "invoice-water-2025-001",
        "contract_id": "contract-water-001",
        "service": "water",
        "amount": "18.00",
        "currency": "USD",
        "due_date": "2025-12-31",
        "status": "open",
    },
}

_payment_intents: dict[str, dict] = {}
_payment_lock = threading.Lock()


def response_payload(handler: BaseHTTPRequestHandler, status: int, payload: dict) -> None:
    body = json.dumps(payload, sort_keys=True).encode()
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json")
    handler.send_header("Content-Length", str(len(body)))
    handler.end_headers()
    handler.wfile.write(body)


class UtilitiesHandler(BaseHTTPRequestHandler):
    server_version = "CholoPay-Synthetic-Utilities/1.0"

    def log_message(self, format: str, *args) -> None:
        return

    def do_GET(self) -> None:
        parsed = urlparse(self.path)
        parts = [part for part in parsed.path.split("/") if part]
        query = parse_qs(parsed.query)

        if parsed.path == "/health":
            return response_payload(self, 200, {"status": "ok", "synthetic": True})
        if parts == ["api", "v1", "contracts"]:
            contract_list = list(CONTRACTS.values())
            return response_payload(self, 200, {"contracts": contract_list})
        if len(parts) == 4 and parts[:3] == ["api", "v1", "contracts"]:
            contract = CONTRACTS.get(parts[3])
            return response_payload(self, 200, contract or {"error": "contract_not_found"}) if contract else response_payload(self, 404, {"error": "contract_not_found"})
        if parts == ["api", "v1", "invoices"]:
            invoices = list(INVOICES.values())
            if "contract_id" in query:
                invoices = [i for i in invoices if i["contract_id"] == query["contract_id"][0]]
            return response_payload(self, 200, {"invoices": invoices})
        if len(parts) == 4 and parts[:3] == ["api", "v1", "invoices"]:
            invoice = INVOICES.get(parts[3])
            return response_payload(self, 200, invoice) if invoice else response_payload(self, 404, {"error": "invoice_not_found"})
        if len(parts) == 5 and parts[:3] == ["api", "v1", "invoices"] and parts[4] == "collection-status":
            invoice = INVOICES.get(parts[3])
            if not invoice:
                return response_payload(self, 404, {"error": "invoice_not_found"})
            return response_payload(self, 200, {
                "invoice_id": invoice["id"],
                "collection_status": "pending",
                "invoice_status": invoice["status"],
                "last_attempt": None,
                "synthetic": True,
            })
        return response_payload(self, 404, {"error": "not_found"})

    def do_POST(self) -> None:
        parsed = urlparse(self.path)
        if parsed.path != "/api/v1/payment-intents":
            return response_payload(self, 404, {"error": "not_found"})
        key = self.headers.get("Idempotency-Key", "").strip()
        if not key:
            return response_payload(self, 400, {"error": "idempotency_key_required"})
        try:
            length = int(self.headers.get("Content-Length", "0"))
            payload = json.loads(self.rfile.read(length) or b"{}")
        except (ValueError, json.JSONDecodeError):
            return response_payload(self, 400, {"error": "invalid_json"})
        invoice_id = payload.get("invoice_id")
        if invoice_id not in INVOICES:
            return response_payload(self, 404, {"error": "invoice_not_found"})
        fingerprint = hashlib.sha256(json.dumps(payload, sort_keys=True).encode()).hexdigest()
        with _payment_lock:
            existing = _payment_intents.get(key)
            if existing and existing["request_fingerprint"] != fingerprint:
                return response_payload(self, 409, {"error": "idempotency_key_reused_with_different_request"})
            if existing:
                return response_payload(self, 200, existing["intent"])
            intent = {
                "id": "pi_" + uuid.uuid4().hex,
                "invoice_id": invoice_id,
                "amount": payload.get("amount", INVOICES[invoice_id]["amount"]),
                "currency": payload.get("currency", INVOICES[invoice_id]["currency"]),
                "status": "pending",
                "debit_performed": False,
                "ledger_entry_created": False,
                "synthetic": True,
            }
            _payment_intents[key] = {"request_fingerprint": fingerprint, "intent": intent}
        return response_payload(self, 201, intent)


def main() -> None:
    host = os.getenv("HOST", "127.0.0.1")
    port = int(os.getenv("PORT", "8080"))
    ThreadingHTTPServer((host, port), UtilitiesHandler).serve_forever()


if __name__ == "__main__":
    main()
