import hashlib
import hmac
import json
import threading
import unittest
from urllib.error import HTTPError
from urllib.request import Request, urlopen

import app


class TransitHttpTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        app.STORE = app.TransitStore()
        cls.server = app.ThreadingHTTPServer(("127.0.0.1", 0), app.TransitHandler)
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
        request = Request(self.base + path, data=data, method=method,
                          headers={"Content-Type": "application/json", **(headers or {})})
        try:
            with urlopen(request) as response:
                return response.status, json.loads(response.read())
        except HTTPError as error:
            return error.code, json.loads(error.read())

    def setUp(self):
        app.STORE = app.TransitStore()

    def registration(self):
        status, result = self.request("POST", "/v1/registrations", {"plate": "SYN-001", "vehicle_type": "synthetic-bus"})
        self.assertEqual(status, 201)
        return result

    def test_registration_fine_obligation_status_and_constancia_are_publicly_observable(self):
        registration = self.registration()
        self.assertFalse((self.request("GET", f"/v1/registrations/{registration['plate']}")[1])["owner_ref"].startswith("real"))
        status, created = self.request("POST", "/v1/fines", {"registration_id": registration["id"], "reason": "synthetic violation", "amount_minor": 2500})
        self.assertEqual(status, 201)
        fine, obligation = created["fine"], created["obligation"]
        self.assertEqual(self.request("GET", f"/v1/fines/{fine['id']}")[1], fine)
        status, result = self.request("GET", f"/v1/obligations/{obligation['id']}")
        self.assertEqual(status, 200)
        self.assertEqual(result["status"], "unpaid")
        status, result = self.request("GET", f"/v1/status?registration_id={registration['id']}")
        self.assertEqual(status, 200)
        self.assertEqual(result["obligations"], [obligation])
        status, constancia = self.request("POST", "/v1/constancias", {"registration_id": registration["id"]})
        self.assertEqual(status, 409)
        self.assertEqual(constancia["error"], "constancia_not_available_until_final_confirmation")

    def test_obligations_are_queryable_by_synthetic_plate_and_receipt_waits_for_final_confirmation(self):
        registration = self.registration()
        _, created = self.request("POST", "/v1/fines", {"registration_id": registration["id"], "reason": "synthetic", "amount_minor": 100})
        obligation = created["obligation"]
        status, result = self.request("GET", f"/v1/vehicles/{registration['plate']}/obligations")
        self.assertEqual(status, 200)
        self.assertEqual(result["items"], [obligation])
        status, result = self.request("GET", f"/v1/obligations/{obligation['id']}/receipt")
        self.assertEqual(status, 409)
        self.assertEqual(result["error"], "receipt_not_available_until_final_confirmation")

    def test_signed_final_application_event_is_idempotent_and_emits_receipt(self):
        registration = self.registration()
        _, created = self.request("POST", "/v1/fines", {"registration_id": registration["id"], "reason": "synthetic", "amount_minor": 100})
        obligation = created["obligation"]
        headers = {"Idempotency-Key": "event-intent-1"}
        _, intent = self.request("POST", f"/v1/obligations/{obligation['id']}/payment-intents", {}, headers)
        event = {"event_id": "evt-1", "type": "payment.final.confirmed", "payment_intent_id": intent["id"], "obligation_id": obligation["id"]}
        raw = json.dumps(event, separators=(",", ":")).encode()
        signature = hmac.new(app.EVENT_SECRET.encode(), raw, hashlib.sha256).hexdigest()
        event_headers = {"X-Transit-Signature": signature}
        status, applied = self.request("POST", "/v1/internal/events", event, event_headers)
        self.assertEqual(status, 200)
        self.assertEqual(applied["status"], "APPLIED")
        self.assertEqual(self.request("POST", "/v1/internal/events", event, event_headers)[1], applied)
        self.assertEqual(self.request("GET", f"/v1/obligations/{obligation['id']}")[1]["status"], "paid")
        status, receipt = self.request("GET", f"/v1/obligations/{obligation['id']}/receipt")
        self.assertEqual(status, 200)
        self.assertTrue(receipt["synthetic"])

    def test_payment_intent_is_idempotent_and_never_marks_obligation_paid(self):
        registration = self.registration()
        _, created = self.request("POST", "/v1/fines", {"registration_id": registration["id"], "reason": "synthetic", "amount_minor": 100})
        obligation = created["obligation"]
        headers = {"Idempotency-Key": "transit-intent-1"}
        first_status, first = self.request("POST", "/v1/payment-intents", {"obligation_id": obligation["id"]}, headers)
        second_status, second = self.request("POST", "/v1/payment-intents", {"obligation_id": obligation["id"]}, headers)
        self.assertEqual(first_status, 201)
        self.assertEqual(second_status, 200)
        self.assertEqual(second, first)
        self.assertEqual(first["status"], "pending")
        self.assertFalse(first["debit_attempted"])
        self.assertFalse(first["ledger_written"])
        self.assertEqual(self.request("GET", f"/v1/obligations/{obligation['id']}")[1]["status"], "unpaid")

    def test_idempotency_mismatch_and_invalid_inputs_do_not_create_records(self):
        registration = self.registration()
        status, result = self.request("POST", "/v1/fines", {"registration_id": registration["id"], "reason": "synthetic", "amount_minor": 100})
        obligation = result["obligation"]
        key = {"Idempotency-Key": "mismatch"}
        self.assertEqual(self.request("POST", "/v1/payment-intents", {"obligation_id": obligation["id"]}, key)[0], 201)
        status, error = self.request("POST", "/v1/payment-intents", {"obligation_id": obligation["id"], "extra": "different"}, key)
        self.assertEqual(status, 409)
        self.assertEqual(error["error"], "idempotency_key_payload_mismatch")
        status, error = self.request("POST", "/v1/fines", {"registration_id": registration["id"], "reason": "", "amount_minor": 0})
        self.assertEqual(status, 422)
        self.assertEqual(error["error"], "reason_and_positive_amount_required")


if __name__ == "__main__":
    unittest.main()
