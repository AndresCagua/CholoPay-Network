"""Docker-only P9 acceptance: public fixture creation followed by financial reads."""
import hashlib
import json
import os
import sys
import urllib.error
import urllib.request

sys.path.insert(0, "/chololab")
from runner import run  # noqa: E402


def request_json(url, method="GET", token=None, body=None):
    headers = {"Accept": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    data = None
    if body is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(body).encode()
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    with urllib.request.urlopen(request, timeout=10) as response:
        return response.status, json.loads(response.read())


def identity(balance):
    return {key: balance.get(key) for key in ("account_id", "currency", "posted_balance_minor", "available_balance_minor")}


def main():
    base = os.getenv("BANK_URL", "http://bank-core:8080")
    bank_token = os.getenv("BANK_API_TOKEN", "bank-local-test-only")
    seed = os.getenv("LOGICAL_SEED", "p9-financial-read-v1")
    commit = os.getenv("GIT_COMMIT", "unknown")
    target = f"{base}/v1/accounts/{{account_id}}/balance"
    for _ in range(60):
        try:
            if request_json(f"{base}/health/ready")[0] == 200:
                break
        except urllib.error.URLError:
            pass
        import time
        time.sleep(1)
    else:
        raise AssertionError("bank-core readiness did not become 200")

    status, fixture = request_json(
        f"{base}/v1/customers", method="POST", token=bank_token,
        body={"synthetic_reference": f"chololab-{seed}"},
    )
    assert status == 201, f"fixture creation status={status}"
    customer_token = fixture["access_token"]
    account_id = fixture["accounts"][0]["account_id"]
    read_url = target.format(account_id=account_id)
    before_status, before = request_json(read_url, token=customer_token)
    assert before_status == 200, f"fixture read status={before_status}"

    summary = run(
        read_url, int(os.getenv("REQUESTS", "20")), int(os.getenv("CONCURRENCY", "4")), 0,
        expected_status=200, headers={"Authorization": f"Bearer {customer_token}"},
        logical_seed=seed, commit=commit, scenario="p9_financial_read_fixture",
    )
    after_status, after = request_json(read_url, token=customer_token)
    ready_status, _ = request_json(f"{base}/health/ready")
    before_id, after_id = identity(before), identity(after)
    assert before_id == after_id, f"fixture changed: before={before_id} after={after_id}"
    assert after_status == 200 and ready_status == 200
    assert summary["successful"] == summary["requests"] and summary["red"]["errors"] == 0
    print(json.dumps({"event": "p9_acceptance", "fixture_identity": before_id,
                      "summary": summary, "readiness_status": ready_status,
                      "published_ports": False}, sort_keys=True))
    print("P9 PERFORMANCE PASS: private financial read fixture preserved, expected responses received, errors=0, readiness=200")


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, KeyError, urllib.error.URLError) as exc:
        print(f"P9 PERFORMANCE FAIL: {exc}", file=sys.stderr)
        raise
