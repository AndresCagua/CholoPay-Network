"""Synthetic transit authority API for Docker-only CholoPay demos.

This process-local service is not a real authority. It uses synthetic records only,
never accesses accounts, debits money, or writes a payment ledger.
"""
from __future__ import annotations

import hashlib
import json
import os
import threading
import uuid
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import parse_qs, urlparse


class TransitStore:
    def __init__(self) -> None:
        self._lock = threading.RLock()
        self.registrations: dict[str, dict[str, Any]] = {}
        self.fines: dict[str, dict[str, Any]] = {}
        self.obligations: dict[str, dict[str, Any]] = {}
        self.payment_intents: dict[str, dict[str, Any]] = {}
        self.payment_keys: dict[str, tuple[str, str]] = {}
        self.constancias: dict[str, dict[str, Any]] = {}

    def register(self, body: dict[str, Any]) -> tuple[int, dict[str, Any]]:
        plate = body.get("plate")
        if not isinstance(plate, str) or not plate.strip():
            return HTTPStatus.UNPROCESSABLE_ENTITY, {"error": "plate_required"}
        plate = plate.strip().upper()
        with self._lock:
            existing = next((r for r in self.registrations.values() if r["plate"] == plate), None)
            if existing:
                return HTTPStatus.CONFLICT, {"error": "registration_already_exists", "registration": existing}
            record = {"id": str(uuid.uuid4()), "plate": plate,
                      "vehicle_type": body.get("vehicle_type", "synthetic-car"),
                      "owner_ref": body.get("owner_ref", "synthetic-owner"), "status": "active"}
            self.registrations[record["id"]] = record
            return HTTPStatus.CREATED, record

    def create_fine(self, body: dict[str, Any]) -> tuple[int, dict[str, Any]]:
        registration_id = body.get("registration_id")
        reason = body.get("reason")
        amount = body.get("amount_minor")
        with self._lock:
            if registration_id not in self.registrations:
                return HTTPStatus.NOT_FOUND, {"error": "registration_not_found"}
            if not isinstance(reason, str) or not reason.strip() or not isinstance(amount, int) or isinstance(amount, bool) or amount <= 0:
                return HTTPStatus.UNPROCESSABLE_ENTITY, {"error": "reason_and_positive_amount_required"}
            fine = {"id": str(uuid.uuid4()), "registration_id": registration_id, "reason": reason.strip(),
                    "amount_minor": amount, "currency": "SYN", "status": "unpaid"}
            obligation = {"id": str(uuid.uuid4()), "type": "fine", "source_id": fine["id"],
                          "registration_id": registration_id, "amount_minor": amount, "currency": "SYN", "status": "unpaid"}
            self.fines[fine["id"]] = fine
            self.obligations[obligation["id"]] = obligation
            return HTTPStatus.CREATED, {"fine": fine, "obligation": obligation}

    def payment_intent(self, body: dict[str, Any], key: str | None) -> tuple[int, dict[str, Any]]:
        obligation_id = body.get("obligation_id")
        if not key:
            return HTTPStatus.BAD_REQUEST, {"error": "idempotency_key_required"}
        with self._lock:
            obligation = self.obligations.get(obligation_id)
            if not obligation:
                return HTTPStatus.NOT_FOUND, {"error": "obligation_not_found"}
            fingerprint = hashlib.sha256(json.dumps(body, sort_keys=True).encode()).hexdigest()
            previous = self.payment_keys.get(key)
            if previous:
                intent_id, previous_fingerprint = previous
                if previous_fingerprint != fingerprint:
                    return HTTPStatus.CONFLICT, {"error": "idempotency_key_payload_mismatch"}
                return HTTPStatus.OK, self.payment_intents[intent_id]
            intent = {"id": str(uuid.uuid4()), "obligation_id": obligation_id,
                      "amount_minor": obligation["amount_minor"], "currency": obligation["currency"],
                      "status": "pending", "debit_attempted": False, "ledger_written": False}
            self.payment_intents[intent["id"]] = intent
            self.payment_keys[key] = (intent["id"], fingerprint)
            return HTTPStatus.CREATED, intent

    def constancia(self, body: dict[str, Any]) -> tuple[int, dict[str, Any]]:
        registration_id = body.get("registration_id")
        with self._lock:
            if registration_id not in self.registrations:
                return HTTPStatus.NOT_FOUND, {"error": "registration_not_found"}
            obligations = [o for o in self.obligations.values() if o["registration_id"] == registration_id]
            record = {"id": str(uuid.uuid4()), "registration_id": registration_id,
                      "obligation_status": "unpaid" if any(o["status"] == "unpaid" for o in obligations) else "clear",
                      "obligation_ids": [o["id"] for o in obligations], "synthetic": True}
            self.constancias[record["id"]] = record
            return HTTPStatus.CREATED, record


STORE = TransitStore()


class TransitHandler(BaseHTTPRequestHandler):
    server_version = "CholoPay-Transit-Synthetic/0.1"

    def _json(self, status: int, body: Any) -> None:
        encoded = json.dumps(body, separators=(",", ":")).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def _body(self) -> dict[str, Any]:
        length = int(self.headers.get("Content-Length", "0"))
        value = json.loads(self.rfile.read(length) or b"{}")
        return value if isinstance(value, dict) else {}

    def do_GET(self) -> None:  # noqa: N802
        parsed = urlparse(self.path)
        path = parsed.path.rstrip("/") or "/"
        if path == "/health/live":
            return self._json(HTTPStatus.OK, {"status": "ok", "service": "transit-synthetic", "real_authority": False})
        parts = path.split("/")
        if len(parts) == 4 and parts[1:3] == ["v1", "registrations"]:
            with STORE._lock:
                item = STORE.registrations.get(parts[3]) or next((r for r in STORE.registrations.values() if r["plate"] == parts[3].upper()), None)
            return self._json(HTTPStatus.OK if item else HTTPStatus.NOT_FOUND, item or {"error": "registration_not_found"})
        collections = {"fines": STORE.fines, "obligations": STORE.obligations, "payment-intents": STORE.payment_intents, "constancias": STORE.constancias}
        if len(parts) == 4 and parts[1:3] in (["v1", key] for key in collections):
            item = collections[parts[2]].get(parts[3])
            return self._json(HTTPStatus.OK if item else HTTPStatus.NOT_FOUND, item or {"error": f"{parts[2].replace('-', '_')}_not_found"})
        if path == "/v1/obligations":
            query = parse_qs(parsed.query)
            registration_id = query.get("registration_id", [None])[0]
            status = query.get("status", [None])[0]
            with STORE._lock:
                items = [o for o in STORE.obligations.values() if (not registration_id or o["registration_id"] == registration_id) and (not status or o["status"] == status)]
            return self._json(HTTPStatus.OK, {"items": items})
        if path == "/v1/status":
            registration_id = parse_qs(parsed.query).get("registration_id", [None])[0]
            if not registration_id or registration_id not in STORE.registrations:
                return self._json(HTTPStatus.NOT_FOUND, {"error": "registration_not_found"})
            with STORE._lock:
                return self._json(HTTPStatus.OK, {"registration": STORE.registrations[registration_id], "obligations": [o for o in STORE.obligations.values() if o["registration_id"] == registration_id]})
        return self._json(HTTPStatus.NOT_FOUND, {"error": "not_found"})

    def do_POST(self) -> None:  # noqa: N802
        path = urlparse(self.path).path.rstrip("/")
        try:
            body = self._body()
        except (ValueError, json.JSONDecodeError):
            return self._json(HTTPStatus.BAD_REQUEST, {"error": "invalid_json"})
        if path == "/v1/registrations":
            status, result = STORE.register(body)
        elif path == "/v1/fines":
            status, result = STORE.create_fine(body)
        elif path == "/v1/payment-intents":
            status, result = STORE.payment_intent(body, self.headers.get("Idempotency-Key"))
        elif path == "/v1/constancias":
            status, result = STORE.constancia(body)
        else:
            return self._json(HTTPStatus.NOT_FOUND, {"error": "not_found"})
        return self._json(status, result)

    def log_message(self, format: str, *args: Any) -> None:
        if os.environ.get("ACCESS_LOG", "0") == "1":
            super().log_message(format, *args)


def create_server() -> ThreadingHTTPServer:
    return ThreadingHTTPServer((os.environ.get("HOST", "0.0.0.0"), int(os.environ.get("PORT", "8082"))), TransitHandler)


if __name__ == "__main__":
    create_server().serve_forever()
