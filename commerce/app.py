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
        return self._json(HTTPStatus.NOT_FOUND, {"error": "not_found"})

    def log_message(self, format: str, *args: Any) -> None:
        if os.environ.get("ACCESS_LOG", "0") == "1":
            super().log_message(format, *args)


def create_server() -> ThreadingHTTPServer:
    return ThreadingHTTPServer((os.environ.get("HOST", "0.0.0.0"), int(os.environ.get("PORT", "8081"))), CommerceHandler)


if __name__ == "__main__":
    create_server().serve_forever()
