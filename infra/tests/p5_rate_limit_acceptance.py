"""Docker-only acceptance for the gateway's local synthetic rate limiter."""
import json
import os
import urllib.error
import urllib.request

base = os.environ.get("GATEWAY_URL", "http://gateway:8090")
write_key = os.environ.get("GATEWAY_API_KEY", "sk_sandbox_merchant_rw")
read_key = os.environ.get("READ_ONLY_API_KEY", "sk_sandbox_merchant_ro")


def request(path, key=None, idempotency=None, body=None):
    headers = {}
    if key is not None:
        headers["X-Api-Key"] = key
    if idempotency is not None:
        headers["Idempotency-Key"] = idempotency
    data = None
    if body is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(body).encode()
    try:
        with urllib.request.urlopen(urllib.request.Request(base + path, headers=headers, data=data, method="POST" if body is not None else "GET"), timeout=5) as response:
            return response.status, dict(response.headers), json.loads(response.read())
    except urllib.error.HTTPError as error:
        return error.code, dict(error.headers), json.loads(error.read())


def require(condition, message):
    if not condition:
        raise AssertionError(message)


body = {"amount_minor": 100, "currency": "USD", "merchant_order_id": "p5-rate-limit"}
results = [request("/v1/payment-intents", write_key, f"p5-{index}", body) for index in range(1, 4)]
require([item[0] for item in results] == [201, 201, 429], f"unexpected statuses: {results}")
retry_after = next((value for name, value in results[2][1].items() if name.lower() == "retry-after"), "")
require(retry_after.isdigit() and int(retry_after) > 0, f"invalid Retry-After: {retry_after!r}")
require(results[2][2].get("error") == "rate_limited", f"unexpected 429 body: {results[2][2]}")
require(request("/v1/payment-intents/pi_sandbox_00000003", read_key)[0] == 404, "limited request created an intent")
require(request("/v1/payment-intents", read_key, "p5-read-only", body)[0] == 403, "read-only key bypassed authorization")
require(request("/health/live")[0] == 200, "gateway health is not ready")
print("P5 RATE LIMIT PASS: 201,201,429; retry-after; no limited mutation; 403; health=200")
