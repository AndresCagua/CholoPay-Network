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

def status(url: str) -> int:
    try:
        with urllib.request.urlopen(url, timeout=3) as response:
            return response.status
    except urllib.error.HTTPError as error:
        return error.code
    except urllib.error.URLError:
        return 599


def wait_for(url: str, expected: int) -> None:
    for _ in range(30):
        if status(url) == expected:
            return
        time.sleep(1)
    raise SystemExit(f"{url} no respondió {expected}")

mode = os.environ.get("P8_MODE", "healthy")
for name, (base, _) in services.items():
    wait_for(base + "/health/live", 200)
    expected = 503 if mode == "network-db-down" and name == "network" else 200
    wait_for(base + "/health/ready", expected)
print(f"P8 HEALTH PASS: mode={mode}, live/ready verificados dentro de Docker")
