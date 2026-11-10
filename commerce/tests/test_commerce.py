import json
import threading
import unittest
from urllib.error import HTTPError
from urllib.request import Request, urlopen

import app


class CommerceFlowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        app.STORE = app.CommerceStore()
        cls.server = app.ThreadingHTTPServer(("127.0.0.1", 0), app.CommerceHandler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.base = f"http://127.0.0.1:{cls.server.server_port}"

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join(timeout=2)

    def request(self, method, path, body=None, headers=None):
        data = json.dumps(body).encode() if body is not None else None
        request = Request(self.base + path, data=data, method=method, headers={"Content-Type": "application/json", **(headers or {})})
        try:
            with urlopen(request) as response:
                return response.status, json.loads(response.read())
        except HTTPError as error:
            return error.code, json.loads(error.read())

    def test_catalog_cart_order_inventory_flow_is_observable(self):
        status, catalog = self.request("GET", "/v1/catalog")
        self.assertEqual(status, 200)
        self.assertEqual({item["sku"] for item in catalog["items"]}, {"demo-coffee", "demo-mug"})

        status, cart = self.request("POST", "/v1/carts", {})
        self.assertEqual(status, 201)
        cart_id = cart["id"]
        status, cart = self.request("POST", f"/v1/carts/{cart_id}/items", {"sku": "demo-coffee", "quantity": 2})
        self.assertEqual(status, 200)
        self.assertEqual(cart["items"], [{"sku": "demo-coffee", "quantity": 2}])

        status, before = self.request("GET", "/v1/inventory")
        self.assertEqual(before["items"]["demo-coffee"], 10)
        status, order = self.request("POST", "/v1/orders", {"cart_id": cart_id}, {"Idempotency-Key": "order-flow-1"})
        self.assertEqual(status, 201)
        self.assertEqual(order["total_minor"], 1300)
        self.assertEqual(order["status"], "created")

        status, after = self.request("GET", "/v1/inventory")
        self.assertEqual(status, 200)
        self.assertEqual(after["items"]["demo-coffee"], 8)
        status, fetched = self.request("GET", f"/v1/orders/{order['id']}")
        self.assertEqual(status, 200)
        self.assertEqual(fetched, order)

    def test_order_idempotency_does_not_decrement_inventory_twice(self):
        _, cart = self.request("POST", "/v1/carts", {})
        self.request("POST", f"/v1/carts/{cart['id']}/items", {"sku": "demo-mug", "quantity": 1})
        first_status, first = self.request("POST", "/v1/orders", {"cart_id": cart["id"]}, {"Idempotency-Key": "same-order"})
        second_status, second = self.request("POST", "/v1/orders", {"cart_id": cart["id"]}, {"Idempotency-Key": "same-order"})
        self.assertEqual(first_status, 201)
        self.assertEqual(second_status, 200)
        self.assertEqual(second, first)
        _, inventory = self.request("GET", "/v1/inventory")
        self.assertEqual(inventory["items"]["demo-mug"], 4)

    def test_invalid_item_does_not_change_cart(self):
        _, cart = self.request("POST", "/v1/carts", {})
        status, error = self.request("POST", f"/v1/carts/{cart['id']}/items", {"sku": "missing", "quantity": 1})
        self.assertEqual(status, 422)
        self.assertEqual(error["error"], "invalid_item")
        _, unchanged = self.request("GET", f"/v1/carts/{cart['id']}")
        self.assertEqual(unchanged["items"], [])


if __name__ == "__main__":
    unittest.main()
