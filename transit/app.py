"""API sintética de autoridad de tránsito para demostraciones Docker de CholoPay.

Este servicio local al proceso no es una autoridad real. Solo usa registros sintéticos;
nunca accede a cuentas, debita dinero ni escribe en un libro mayor de pagos.
"""
from __future__ import annotations

import hashlib
import hmac
import json
import os
import threading
import uuid
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import parse_qs, urlparse


EVENT_SECRET = os.environ.get("TRANSIT_EVENT_SECRET", "synthetic-transit-event-secret")


class TransitStore:
    def __init__(self) -> None:
        self._lock = threading.RLock()
        self.registrations: dict[str, dict[str, Any]] = {}
        self.fines: dict[str, dict[str, Any]] = {}
        self.obligations: dict[str, dict[str, Any]] = {}
        self.payment_intents: dict[str, dict[str, Any]] = {}
        self.payment_keys: dict[str, tuple[str, str]] = {}
        self.constancias: dict[str, dict[str, Any]] = {}
        self.events: dict[str, dict[str, Any]] = {}

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
            if obligation["status"] != "unpaid":
                return HTTPStatus.CONFLICT, {"error": "obligation_not_payable", "status": obligation["status"]}
            intent = {"id": str(uuid.uuid4()), "obligation_id": obligation_id,
                      "amount_minor": obligation["amount_minor"], "currency": obligation["currency"],
                      "status": "pending", "debit_attempted": False, "ledger_written": False}
            self.payment_intents[intent["id"]] = intent
            self.payment_keys[key] = (intent["id"], fingerprint)
            return HTTPStatus.CREATED, intent

    def obligations_for_plate(self, plate: str) -> tuple[int, dict[str, Any]]:
        with self._lock:
            registration = next((r for r in self.registrations.values() if r["plate"] == plate.strip().upper()), None)
            if not registration:
                return HTTPStatus.NOT_FOUND, {"error": "vehicle_not_found"}
            return HTTPStatus.OK, {"vehicle": registration, "items": [o for o in self.obligations.values() if o["registration_id"] == registration["id"]]}

    def apply_event(self, body: dict[str, Any], signature: str | None) -> tuple[int, dict[str, Any]]:
        raw = json.dumps(body, separators=(",", ":")).encode()
        expected = hmac.new(EVENT_SECRET.encode(), raw, hashlib.sha256).hexdigest()
        if not signature or not hmac.compare_digest(signature.removeprefix("sha256="), expected):
            return HTTPStatus.UNAUTHORIZED, {"error": "invalid_event_signature"}
        event_id = body.get("event_id")
        if not isinstance(event_id, str) or not event_id:
            return HTTPStatus.UNPROCESSABLE_ENTITY, {"error": "event_id_required"}
        with self._lock:
            if event_id in self.events:
                return HTTPStatus.OK, self.events[event_id]
            intent = self.payment_intents.get(body.get("payment_intent_id"))
            obligation = self.obligations.get(body.get("obligation_id"))
            if not intent or not obligation or intent["obligation_id"] != obligation["id"]:
                return HTTPStatus.NOT_FOUND, {"error": "payment_intent_or_obligation_not_found"}
            if body.get("type") not in ("payment.final.confirmed", "payment.confirmed"):
                return HTTPStatus.UNPROCESSABLE_ENTITY, {"error": "unsupported_event_type"}
            if body.get("application_available", True) is False:
                result = {"event_id": event_id, "status": "SETTLED_AWAITING_APPLICATION", "payment_intent_id": intent["id"], "obligation_id": obligation["id"]}
                intent["status"] = "settled_awaiting_application"
            else:
                intent["status"] = "confirmed"
                intent["debit_attempted"] = True
                obligation["status"] = "paid"
                result = {"event_id": event_id, "status": "APPLIED", "payment_intent_id": intent["id"], "obligation_id": obligation["id"]}
            self.events[event_id] = result
            return HTTPStatus.OK, result

    def constancia(self, body: dict[str, Any]) -> tuple[int, dict[str, Any]]:
        registration_id = body.get("registration_id")
        with self._lock:
            if registration_id not in self.registrations:
                return HTTPStatus.NOT_FOUND, {"error": "registration_not_found"}
            obligations = [o for o in self.obligations.values() if o["registration_id"] == registration_id]
            if not obligations or any(o["status"] != "paid" for o in obligations):
                return HTTPStatus.CONFLICT, {"error": "constancia_not_available_until_final_confirmation"}
            record = {"id": str(uuid.uuid4()), "registration_id": registration_id,
                      "obligation_status": "clear", "obligation_ids": [o["id"] for o in obligations], "synthetic": True}
            self.constancias[record["id"]] = record
            return HTTPStatus.CREATED, record

    def receipt(self, obligation_id: str) -> tuple[int, dict[str, Any]]:
        with self._lock:
            obligation = self.obligations.get(obligation_id)
            if not obligation:
                return HTTPStatus.NOT_FOUND, {"error": "obligation_not_found"}
            if obligation["status"] != "paid":
                return HTTPStatus.CONFLICT, {"error": "receipt_not_available_until_final_confirmation"}
            existing = next((r for r in self.constancias.values() if r.get("obligation_id") == obligation_id), None)
            if existing:
                return HTTPStatus.OK, existing
            record = {"id": str(uuid.uuid4()), "obligation_id": obligation_id, "amount_minor": obligation["amount_minor"], "currency": obligation["currency"], "synthetic": True}
            self.constancias[record["id"]] = record
            return HTTPStatus.OK, record


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
        if len(parts) == 5 and parts[1:3] == ["v1", "vehicles"] and parts[4] == "obligations":
            status, result = STORE.obligations_for_plate(parts[3])
            return self._json(status, result)
        if len(parts) == 5 and parts[1:3] == ["v1", "obligations"] and parts[4] == "receipt":
            status, result = STORE.receipt(parts[3])
            return self._json(status, result)
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
        elif len(path.split("/")) == 5 and path.split("/")[1:3] == ["v1", "obligations"] and path.split("/")[4] == "payment-intents":
            body["obligation_id"] = path.split("/")[3]
            status, result = STORE.payment_intent(body, self.headers.get("Idempotency-Key"))
        elif path == "/v1/internal/events":
            status, result = STORE.apply_event(body, self.headers.get("X-Transit-Signature") or self.headers.get("X-Event-Signature"))
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
