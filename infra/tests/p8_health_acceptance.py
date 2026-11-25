"""HTTP checks run from an internal Compose container; no host ports are needed."""
import json
import os
import time
import urllib.error
import urllib.request

services = {
    "network": ("http://network:8080", True),
    "bank-core": ("http://bank-core:8080", True),
    "gateway": ("http://gateway:8090", False),
    "transit": ("http://transit:8082", False),
    "utilities": ("http://utilities:8080", False),
}

def probe(url: str, timeout: int = 3) -> tuple[int, str]:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as response:
            return response.status, response.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as error:
        return error.code, error.read().decode("utf-8", errors="replace")
    except (urllib.error.URLError, TimeoutError) as error:
        return 599, f"<sin respuesta: {error}>"


def wait_for(
    url: str, expected_http: int, expected_status: str, request_timeout: int = 3
) -> None:
    last_http = None
    last_body = None
    for _ in range(30):
        last_http, last_body = probe(url, timeout=request_timeout)
        try:
            payload = json.loads(last_body)
            observed_status = payload.get("status") if isinstance(payload, dict) else None
        except json.JSONDecodeError as error:
            observed_status = None
            parse_error = f"JSON inválido ({error.msg})"
        else:
            parse_error = ""
        if last_http == expected_http and observed_status == expected_status:
            return
        time.sleep(1)
    raise SystemExit(
        f"{url} contrato health inválido; esperado HTTP {expected_http} y "
        f"status={expected_status!r}; observado HTTP {last_http}, "
        f"status={observed_status!r}, body={last_body!r}"
        + (f", {parse_error}" if parse_error else "")
    )


mode = os.environ.get("P8_MODE", "healthy")
expected_modes = {"healthy", "network-db-down", "bank-db-down"}
if mode not in expected_modes:
    raise SystemExit(f"P8_MODE inválido: {mode!r}; esperado uno de {sorted(expected_modes)}")

for name, (base, _) in services.items():
    healthy_status = "UP" if name in {"network", "bank-core"} else "ok"
    wait_for(base + "/health/live", 200, healthy_status)
    if mode == "network-db-down" and name == "network":
        expected_http, expected_status = 503, "DOWN"
    elif mode == "bank-db-down" and name == "bank-core":
        expected_http, expected_status = 503, "DOWN"
    else:
        expected_http, expected_status = 200, healthy_status
    request_timeout = 35 if mode == "bank-db-down" and name == "bank-core" else 3
    wait_for(
        base + "/health/ready",
        expected_http,
        expected_status,
        request_timeout=request_timeout,
    )

print(f"P8 HEALTH PASS: mode={mode}, live/ready verificados dentro de Docker")
