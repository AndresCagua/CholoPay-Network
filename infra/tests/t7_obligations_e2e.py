"""Docker-only public HTTP acceptance for six synthetic obligations.

This runner never opens a database connection. Every request uses a Compose service
alias and a public HTTP API; all amounts and resulting states remain synthetic.
"""
import json
import os
import time
from urllib.error import HTTPError
from urllib.request import Request, urlopen

GATEWAY = "http://gateway:8090"
TRANSIT = "http://transit:8082"
UTILITIES = "http://utilities:8080"
BANK = "http://bank-core:8080"
PARTICIPANT = "BANCO_COSTA"
API_KEY = os.environ.get("GATEWAY_API_KEY", "sk_sandbox_merchant_rw")
BANK_TOKEN = os.environ.get("BANK_API_TOKEN", "bank-local-test-only")
SOURCE = "00000000-0000-4000-8000-000000000001"


def call(method, url, body=None, headers=None):
    data = json.dumps(body).encode() if body is not None else None
    request = Request(url, data=data, method=method, headers={"Content-Type": "application/json", **(headers or {})})
    try:
        with urlopen(request, timeout=3) as response:
            return response.status, json.loads(response.read())
    except HTTPError as error:
        return error.code, json.loads(error.read())


def wait_for(url, method="GET", headers=None, accepted=(200,)):
    for _ in range(120):
        try:
            status, body = call(method, url, headers=headers)
            if status in accepted:
                return body
        except Exception:
            pass
        time.sleep(1)
    raise AssertionError(f"service unavailable: {url}")


wait_for(f"{GATEWAY}/health/live")
# bank-core has no health route; an authentication/idempotency rejection from a
# public controller proves that its HTTP server and Spring context are ready
# without creating a financial record.
wait_for(
    f"{BANK}/v1/obligations/t7-readiness/payment-intents",
    method="POST",
    headers={"Authorization": f"Bearer {BANK_TOKEN}"},
    accepted=(400,),
)

status, registration = call("POST", f"{TRANSIT}/v1/registrations", {"plate": "T7-SYN-001"})
assert status == 201, (status, registration)
status, fine = call("POST", f"{TRANSIT}/v1/fines", {"registration_id": registration["id"], "reason": "T7 synthetic", "amount_minor": 1100})
assert status == 201, (status, fine)

status, projected = call("GET", f"{GATEWAY}/v1/participants/{PARTICIPANT}/obligations", headers={"X-Api-Key": API_KEY})
assert status == 200, (status, projected)
items = projected["obligations"]
assert {item["kind"] for item in items} >= {"electricity", "water", "internet", "mobile", "registration", "fine"}
assert len(items) >= 6

for item in items:
    key = f"t7-{item['id']}"
    is_transit = item["kind"] in {"registration", "fine"}
    payload = {
        "participantId": PARTICIPANT,
        "participant_id": PARTICIPANT,
        "sourceAccountId": SOURCE,
        "source_account_id": SOURCE,
        "biller": "transit" if is_transit else "utilities",
        "service": item["kind"],
        "invoiceId": None if is_transit else item["id"],
        "invoice_id": None if is_transit else item["id"],
        "amount": f"{item['amountMinor'] / 100:.2f}",
        "currency": item["currency"],
    }
    headers = {"Authorization": f"Bearer {BANK_TOKEN}", "Idempotency-Key": key}
    path = f"{BANK}/v1/participants/{PARTICIPANT}/obligations/{item['id']}/payment-intents"
    first_status, first = call("POST", path, payload, headers)
    assert first_status in (201, 202), (item, first_status, first)
    replay_status, replay = call("POST", path, payload, headers)
    assert replay_status in (201, 202) and replay == first, (item, replay_status, replay)
    conflict = dict(payload, amount=f"{item['amountMinor'] / 100 + 1:.2f}")
    conflict_status, _ = call("POST", path, conflict, headers)
    assert conflict_status == 409, (item, conflict_status)

for item in items:
    if item["kind"] in {"registration", "fine"}:
        status, state = call("GET", f"{TRANSIT}/v1/obligations/{item['id']}")
    else:
        status, state = call("GET", f"{UTILITIES}/api/v1/invoices/{item['id']}")
    assert status == 200 and state.get("status") in {"paid", "PAID"}, (item, status, state)

print("T7 PASS: six synthetic obligations queried and paid through gateway -> bank-core -> synthetic billers")
print("BOUNDARY: no real settlement, ledger movement, or official transit certificate is claimed")
