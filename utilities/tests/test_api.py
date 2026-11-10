import hashlib
import hmac
import json
import threading
import unittest
from http.client import HTTPConnection
from http.server import ThreadingHTTPServer

from app import UtilitiesHandler


class UtilitiesApiContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), UtilitiesHandler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.host, cls.port = cls.server.server_address

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def request(self, method, path, payload=None, headers=None):
        connection = HTTPConnection(self.host, self.port)
        body = json.dumps(payload).encode() if payload is not None else None
        connection.request(method, path, body=body, headers={"Content-Type": "application/json", **(headers or {})})
        response = connection.getresponse()
        result = json.loads(response.read())
        connection.close()
        return response.status, result

    def test_services_catalog_and_contract_bills_are_public(self):
        status, catalog = self.request("GET", "/v1/services/catalog")
        self.assertEqual(status, 200)
        self.assertEqual([service["id"] for service in catalog["services"]], ["electricity", "water", "internet", "mobile"])
        self.assertTrue(all(service["currency"] == "USD" for service in catalog["services"]))

        status, bills = self.request("GET", "/v1/contracts/contract-internet-001/bills")
        self.assertEqual(status, 200)
        self.assertEqual([bill["service"] for bill in bills["bills"]], ["internet"])
        self.assertEqual(bills["bills"][0]["currency"], "USD")

    def test_bill_routes_create_idempotent_intent_and_apply_settlement_once(self):
        status, bill = self.request("GET", "/v1/bills/bill-mobile-2025-001")
        self.assertEqual(status, 200)
        self.assertEqual(bill["service"], "mobile")

        payload = {"amount": bill["amount"], "currency": "USD"}
        headers = {"Idempotency-Key": "bill-mobile-intent-001"}
        status, first = self.request("POST", "/v1/bills/bill-mobile-2025-001/payment-intents", payload, headers)
        self.assertEqual(status, 201)
        status, replay = self.request("POST", "/v1/bills/bill-mobile-2025-001/payment-intents", payload, headers)
        self.assertEqual(status, 200)
        self.assertEqual(replay["id"], first["id"])

        event = {"event_type": "payment.settled", "bill_id": "bill-mobile-2025-001", "payment_intent_id": first["id"]}
        canonical = json.dumps(event, sort_keys=True, separators=(",", ":"))
        timestamp = "1700000002"
        signature = hmac.new(b"local-test-only", f"{timestamp}.{canonical}".encode(), hashlib.sha256).hexdigest()
        event_headers = {"X-CholoPay-Event-Id": "evt-mobile-001", "X-CholoPay-Timestamp": timestamp, "X-CholoPay-Signature": signature}
        status, receipt = self.request("POST", "/api/v1/internal/payment-events", event, event_headers)
        self.assertEqual(status, 200)
        self.assertEqual(receipt["status"], "PAID")
        status, state = self.request("GET", "/v1/bills/bill-mobile-2025-001/payment-status")
        self.assertEqual(status, 200)
        self.assertEqual(state["status"], "paid")
        self.assertEqual(state["settlement_state"], "APPLIED")

    def test_new_bill_rejects_a_second_payment_reservation(self):
        payload = {"amount": "35.00", "currency": "USD"}
        status, first = self.request("POST", "/v1/bills/bill-internet-2025-001/payment-intents", payload, {"Idempotency-Key": "bill-internet-intent-001"})
        self.assertEqual(status, 201)
        status, conflict = self.request("POST", "/v1/bills/bill-internet-2025-001/payment-intents", payload, {"Idempotency-Key": "bill-internet-intent-002"})
        self.assertEqual(status, 409)
        self.assertEqual(conflict["error"], "payment_already_in_progress")
        self.assertNotEqual(first["id"], conflict.get("id"))

    def test_contract_and_invoice_query_contracts_are_public(self):
        status, contract = self.request("GET", "/api/v1/contracts/contract-elec-001")
        self.assertEqual(status, 200)
        self.assertEqual(contract["service"], "electricity")

        status, invoices = self.request("GET", "/api/v1/invoices?contract_id=contract-elec-001")
        self.assertEqual(status, 200)
        self.assertEqual([item["id"] for item in invoices["invoices"]], ["invoice-elec-2025-001"])

    def test_invoice_and_collection_status(self):
        status, invoice = self.request("GET", "/api/v1/invoices/invoice-elec-2025-001")
        self.assertEqual(status, 200)
        self.assertEqual(invoice["status"], "open")

        status, collection = self.request("GET", "/api/v1/invoices/invoice-elec-2025-001/collection-status")
        self.assertEqual(status, 200)
        self.assertEqual(collection["collection_status"], "pending")
        self.assertTrue(collection["synthetic"])

    def test_payment_intent_is_idempotent_and_does_not_cross_boundary(self):
        payload = {"invoice_id": "invoice-elec-2025-001", "amount": "42.50", "currency": "USD"}
        headers = {"Idempotency-Key": "test-payment-001"}
        status, first = self.request("POST", "/api/v1/payment-intents", payload, headers)
        self.assertEqual(status, 201)
        self.assertEqual(first["status"], "pending")
        self.assertFalse(first["debit_performed"])
        self.assertFalse(first["ledger_entry_created"])

        status, second = self.request("POST", "/api/v1/payment-intents", payload, headers)
        self.assertEqual(status, 200)
        self.assertEqual(second["id"], first["id"])

        status, conflict = self.request("POST", "/api/v1/payment-intents", {**payload, "amount": "1.00"}, headers)
        self.assertEqual(status, 409)
        self.assertEqual(conflict["error"], "idempotency_key_reused_with_different_request")

    def test_settled_event_applies_invoice_once_and_replays(self):
        intent_status, intent = self.request("POST", "/api/v1/payment-intents", {"invoice_id": "invoice-elec-2025-001", "amount": "42.50", "currency": "USD"}, {"Idempotency-Key": "settlement-intent-001"})
        self.assertEqual(intent_status, 201)
        payload = {"event_type": "payment.settled", "invoice_id": "invoice-elec-2025-001", "payment_intent_id": intent["id"]}
        canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"))
        timestamp = "1700000000"
        signature = hmac.new(b"local-test-only", f"{timestamp}.{canonical}".encode(), hashlib.sha256).hexdigest()
        headers = {"X-CholoPay-Event-Id": "evt-settled-001", "X-CholoPay-Timestamp": timestamp, "X-CholoPay-Signature": signature}
        status, receipt = self.request("POST", "/api/v1/internal/payment-events", payload, headers)
        self.assertEqual(status, 200)
        self.assertEqual(receipt["status"], "PAID")
        self.assertFalse(receipt["debit_performed"])
        status, replay = self.request("POST", "/api/v1/internal/payment-events", payload, headers)
        self.assertEqual(status, 200)
        self.assertEqual(replay, receipt)
        status, payment_status = self.request("GET", "/api/v1/invoices/invoice-elec-2025-001/payment-status")
        self.assertEqual(status, 200)
        self.assertEqual(payment_status["status"], "paid")

    def test_settled_event_rejects_unbound_intent(self):
        payload = {"event_type": "payment.settled", "invoice_id": "invoice-elec-2025-001", "payment_intent_id": "pi_unknown"}
        canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"))
        timestamp = "1700000001"
        signature = hmac.new(b"local-test-only", f"{timestamp}.{canonical}".encode(), hashlib.sha256).hexdigest()
        status, body = self.request("POST", "/api/v1/internal/payment-events", payload, {"X-CholoPay-Event-Id": "evt-unbound", "X-CholoPay-Timestamp": timestamp, "X-CholoPay-Signature": signature})
        self.assertEqual(status, 409)
        self.assertEqual(body["error"], "payment_intent_not_bound_to_invoice")

    def test_settled_event_rejects_bad_signature(self):
        payload = {"event_type": "payment.settled", "invoice_id": "invoice-water-2025-001", "payment_intent_id": "pi_bad"}
        status, body = self.request("POST", "/api/v1/internal/payment-events", payload, {"X-CholoPay-Event-Id": "evt-bad", "X-CholoPay-Timestamp": "1700000000", "X-CholoPay-Signature": "bad"})
        self.assertEqual(status, 401)
        self.assertEqual(body["error"], "invalid_event_signature")

    def test_payment_intent_requires_key_and_known_invoice(self):
        status, body = self.request("POST", "/api/v1/payment-intents", {"invoice_id": "invoice-elec-2025-001"})
        self.assertEqual(status, 400)
        self.assertEqual(body["error"], "idempotency_key_required")

        status, body = self.request("POST", "/api/v1/payment-intents", {"invoice_id": "missing"}, {"Idempotency-Key": "missing-invoice"})
        self.assertEqual(status, 404)
        self.assertEqual(body["error"], "invoice_not_found")


if __name__ == "__main__":
    unittest.main()
