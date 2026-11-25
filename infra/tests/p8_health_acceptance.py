"""HTTP checks run from an internal Compose container; no host ports are needed."""
import os
import sys
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

def status(url: str, timeout: int = 3) -> int:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as response:
            return response.status
    except urllib.error.HTTPError as error:
        return error.code
    except (urllib.error.URLError, TimeoutError):
        return 599


def wait_for(url: str, expected: int, request_timeout: int = 3) -> None:
    last_status = None
    for _ in range(30):
        last_status = status(url, timeout=request_timeout)
        if last_status == expected:
            return
        time.sleep(1)
    raise SystemExit(
        f"{url} no respondió {expected}; último estado observado: {last_status}"
    )


mode = os.environ.get("P8_MODE", "healthy")
expected_modes = {"healthy", "network-db-down", "bank-db-down"}
if mode not in expected_modes:
    raise SystemExit(f"P8_MODE inválido: {mode!r}; esperado uno de {sorted(expected_modes)}")

for name, (base, _) in services.items():
    wait_for(base + "/health/live", 200)
    if mode == "network-db-down" and name == "network":
        expected = 503
    elif mode == "bank-db-down" and name == "bank-core":
        expected = 503
    else:
        expected = 200
    request_timeout = 35 if mode == "bank-db-down" and name == "bank-core" else 3
    wait_for(base + "/health/ready", expected, request_timeout=request_timeout)

print(f"P8 HEALTH PASS: mode={mode}, live/ready verificados dentro de Docker")
