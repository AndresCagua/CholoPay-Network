"""Synthetic commerce sandbox with an HTTP-only public surface.

This service intentionally uses process-local memory. It is independent from the
payment gateway and does not call external services or move real money.
"""

from __future__ import annotations

import json
import os
import threading
import uuid
from http import HTTPStatus
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import urlparse


CATALOG = {
    "demo-coffee": {"sku": "demo-coffee", "name": "Demo Coffee", "description": "Synthetic coffee product for sandbox flows", "price_minor": 650, "currency": "USD"},
    "demo-mug": {"sku": "demo-mug", "name": "Demo Mug", "description": "Synthetic mug product for sandbox flows", "price_minor": 1200, "currency": "USD"},
}


class CommerceStore:
    def __init__(self) -> None:
        self._lock = threading.RLock()
        self.inventory = {"demo-coffee": 10, "demo-mug": 5}
        self.carts: dict[str, dict[str, Any]] = {}
        self.orders: dict[str, dict[str, Any]] = {}
        self.order_keys: dict[str, tuple[str, str]] = {}
        self.checkouts: dict[str, dict[str, Any]] = {}
        self.checkout_keys: dict[str, tuple[str, str]] = {}

    def create_cart(self) -> dict[str, Any]:
        with self._lock:
            cart = {"id": str(uuid.uuid4()), "items": []}
            self.carts[cart["id"]] = cart
            return cart

    def get_cart(self, cart_id: str) -> dict[str, Any] | None:
        with self._lock:
            return self.carts.get(cart_id)

    def add_item(self, cart_id: str, sku: str, quantity: int) -> dict[str, Any] | None:
        with self._lock:
            cart = self.carts.get(cart_id)
            if cart is None:
                return None
            for item in cart["items"]:
                if item["sku"] == sku:
                    item["quantity"] += quantity
                    break
            else:
                cart["items"].append({"sku": sku, "quantity": quantity})
            return cart

    def create_order(self, cart_id: str, idempotency_key: str) -> tuple[int, dict[str, Any]]:
        with self._lock:
            cart = self.carts.get(cart_id)
            if cart is None:
                return HTTPStatus.NOT_FOUND, {"error": "cart_not_found"}
            payload_fingerprint = json.dumps(cart["items"], sort_keys=True)
            previous = self.order_keys.get(idempotency_key)
            if previous:
                previous_order_id, previous_fingerprint = previous
                previous_order = self.orders[previous_order_id]
                if previous_order["cart_id"] != cart_id or previous_fingerprint != payload_fingerprint:
                    return HTTPStatus.CONFLICT, {"error": "idempotency_key_payload_mismatch"}
                return HTTPStatus.OK, previous_order
            if not cart["items"]:
                return HTTPStatus.UNPROCESSABLE_ENTITY, {"error": "cart_empty"}
            for item in cart["items"]:
                if item["sku"] not in CATALOG:
                    return HTTPStatus.UNPROCESSABLE_ENTITY, {"error": "unknown_sku", "sku": item["sku"]}
                if item["quantity"] > self.inventory[item["sku"]]:
                    return HTTPStatus.CONFLICT, {"error": "insufficient_inventory", "sku": item["sku"]}
            for item in cart["items"]:
                self.inventory[item["sku"]] -= item["quantity"]
            order_items = [
                {**item, "unit_price_minor": CATALOG[item["sku"]]["price_minor"], "line_total_minor": CATALOG[item["sku"]]["price_minor"] * item["quantity"]}
                for item in cart["items"]
            ]
            order = {
                "id": str(uuid.uuid4()), "cart_id": cart_id, "status": "created", "currency": "USD",
                "total_minor": sum(item["line_total_minor"] for item in order_items), "items": order_items,
            }
            self.orders[order["id"]] = order
            self.order_keys[idempotency_key] = (order["id"], payload_fingerprint)
            return HTTPStatus.CREATED, order

    def get_order(self, order_id: str) -> dict[str, Any] | None:
        with self._lock:
            return self.orders.get(order_id)

    def get_checkout(self, checkout_id: str) -> dict[str, Any] | None:
        with self._lock:
            return self.checkouts.get(checkout_id)


GATEWAY_URL = os.environ.get("GATEWAY_URL", "").rstrip("/")
GATEWAY_ADAPTER = os.environ.get("GATEWAY_ADAPTER", "synthetic")


def gateway_create_intent(amount_minor: int, order_id: str, idempotency_key: str) -> tuple[int, dict[str, Any]]:
    if GATEWAY_ADAPTER == "synthetic":
        return HTTPStatus.CREATED, {"id": f"pi_synthetic_{uuid.uuid4().hex}", "status": "requires_confirmation", "amount_minor": amount_minor, "currency": "USD", "ledger_boundary": "Commerce synthetic adapter never moves money."}
    if not GATEWAY_URL:
        return HTTPStatus.SERVICE_UNAVAILABLE, {"error": "gateway_not_configured"}
    payload = json.dumps({"amount_minor": amount_minor, "currency": "USD", "merchant_order_id": order_id}).encode()
    request = Request(f"{GATEWAY_URL}/v1/payment-intents", data=payload, method="POST", headers={"Content-Type": "application/json", "X-Api-Key": os.environ.get("GATEWAY_API_KEY", "sk_sandbox_merchant_rw"), "Idempotency-Key": idempotency_key})
    try:
        with urlopen(request, timeout=2) as response:
            return response.status, json.loads(response.read())
    except (HTTPError, URLError, TimeoutError, json.JSONDecodeError):
        return HTTPStatus.BAD_GATEWAY, {"error": "gateway_unavailable"}


def gateway_confirm_intent(intent_id: str, idempotency_key: str) -> tuple[int, dict[str, Any]]:
    if GATEWAY_ADAPTER == "synthetic":
        return HTTPStatus.OK, {"id": intent_id, "status": "succeeded"}
    payload = b"{}"
    request = Request(f"{GATEWAY_URL}/v1/payment-intents/{intent_id}/confirm", data=payload, method="POST", headers={"Content-Type": "application/json", "X-Api-Key": os.environ.get("GATEWAY_API_KEY", "sk_sandbox_merchant_rw"), "Idempotency-Key": idempotency_key})
    try:
        with urlopen(request, timeout=2) as response:
            return response.status, json.loads(response.read())
    except (HTTPError, URLError, TimeoutError, json.JSONDecodeError):
        return HTTPStatus.BAD_GATEWAY, {"error": "gateway_unavailable"}


STORE = CommerceStore()


class CommerceHandler(BaseHTTPRequestHandler):
    server_version = "CholoPay-Commerce/0.1"

    def _json(self, status: int, body: Any) -> None:
        encoded = json.dumps(body, separators=(",", ":")).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def _body(self) -> dict[str, Any]:
        length = int(self.headers.get("Content-Length", "0"))
        return json.loads(self.rfile.read(length) or b"{}")

    def do_GET(self) -> None:  # noqa: N802
        path = urlparse(self.path).path.rstrip("/") or "/"
        if path == "/health/live":
            return self._json(HTTPStatus.OK, {"status": "ok", "service": "commerce-sandbox"})
        if path == "/v1/catalog":
            return self._json(HTTPStatus.OK, {"items": list(CATALOG.values())})
        if path == "/v1/inventory":
            with STORE._lock:
                return self._json(HTTPStatus.OK, {"items": STORE.inventory.copy()})
        parts = path.split("/")
        if len(parts) == 4 and parts[1:3] == ["v1", "carts"]:
            cart = STORE.get_cart(parts[3])
            return self._json(HTTPStatus.OK if cart else HTTPStatus.NOT_FOUND, cart or {"error": "cart_not_found"})
        if len(parts) == 4 and parts[1:3] == ["v1", "orders"]:
            order = STORE.get_order(parts[3])
            return self._json(HTTPStatus.OK if order else HTTPStatus.NOT_FOUND, order or {"error": "order_not_found"})
        return self._json(HTTPStatus.NOT_FOUND, {"error": "not_found"})

    def do_POST(self) -> None:  # noqa: N802
        path = urlparse(self.path).path.rstrip("/")
        try:
            body = self._body()
        except (ValueError, json.JSONDecodeError):
            return self._json(HTTPStatus.BAD_REQUEST, {"error": "invalid_json"})
        if path == "/v1/carts":
            return self._json(HTTPStatus.CREATED, STORE.create_cart())
        parts = path.split("/")
        if len(parts) == 5 and parts[1:3] == ["v1", "carts"] and parts[4] == "items":
            sku, quantity = body.get("sku"), body.get("quantity")
            if sku not in CATALOG or not isinstance(quantity, int) or isinstance(quantity, bool) or quantity < 1:
                return self._json(HTTPStatus.UNPROCESSABLE_ENTITY, {"error": "invalid_item"})
            cart = STORE.add_item(parts[3], sku, quantity)
            return self._json(HTTPStatus.OK if cart else HTTPStatus.NOT_FOUND, cart or {"error": "cart_not_found"})
        if path == "/v1/orders":
            cart_id, key = body.get("cart_id"), self.headers.get("Idempotency-Key")
            if not cart_id or not key:
                return self._json(HTTPStatus.BAD_REQUEST, {"error": "cart_id_and_idempotency_key_required"})
            status, order = STORE.create_order(cart_id, key)
            return self._json(status, order)
        if path == "/v1/checkout":
            cart_id, key = body.get("cart_id"), self.headers.get("Idempotency-Key")
            if not cart_id or not key:
                return self._json(HTTPStatus.BAD_REQUEST, {"error": "cart_id_and_idempotency_key_required"})
            cart = STORE.get_cart(cart_id)
            fingerprint = json.dumps(cart["items"], sort_keys=True) if cart else ""
            previous = STORE.checkout_keys.get(key)
            if previous:
                if previous[1] != fingerprint or previous[0] not in STORE.checkouts:
                    return self._json(HTTPStatus.CONFLICT, {"error": "idempotency_key_payload_mismatch"})
                return self._json(HTTPStatus.OK, STORE.checkouts[previous[0]])
            status, order = STORE.create_order(cart_id, f"order:{key}")
            if status not in (HTTPStatus.CREATED, HTTPStatus.OK):
                return self._json(status, order)
            gateway_status, intent = gateway_create_intent(order["total_minor"], order["id"], f"checkout:{key}")
            if gateway_status >= 400:
                return self._json(HTTPStatus.BAD_GATEWAY, {"error": "gateway_unavailable"})
            checkout = {"id": f"co_sandbox_{uuid.uuid4().hex}", "order": order, "payment_intent_id": intent["id"], "payment_status": "succeeded" if intent.get("status") == "succeeded" else "pending", "synthetic": GATEWAY_ADAPTER == "synthetic"}
            with STORE._lock:
                STORE.checkouts[checkout["id"]] = checkout
                STORE.checkout_keys[key] = (checkout["id"], fingerprint)
            return self._json(HTTPStatus.CREATED, checkout)
        if len(parts) == 5 and parts[1:3] == ["v1", "checkouts"] and parts[4] == "confirm":
            checkout = STORE.get_checkout(parts[3])
            key = self.headers.get("Idempotency-Key")
            if not checkout:
                return self._json(HTTPStatus.NOT_FOUND, {"error": "checkout_not_found"})
            if not key:
                return self._json(HTTPStatus.BAD_REQUEST, {"error": "idempotency_key_required"})
            status, result = gateway_confirm_intent(checkout["payment_intent_id"], f"checkout-confirm:{key}")
            if status >= 400:
                return self._json(HTTPStatus.BAD_GATEWAY, {"error": "gateway_unavailable"})
            with STORE._lock:
                checkout["payment_status"] = "succeeded" if result.get("status") == "succeeded" else "pending"
                checkout["synthetic"] = GATEWAY_ADAPTER == "synthetic"
            return self._json(HTTPStatus.OK, checkout)
        return self._json(HTTPStatus.NOT_FOUND, {"error": "not_found"})

    def log_message(self, format: str, *args: Any) -> None:
        if os.environ.get("ACCESS_LOG", "0") == "1":
            super().log_message(format, *args)


def create_server() -> ThreadingHTTPServer:
    return ThreadingHTTPServer((os.environ.get("HOST", "0.0.0.0"), int(os.environ.get("PORT", "8081"))), CommerceHandler)


if __name__ == "__main__":
    create_server().serve_forever()
