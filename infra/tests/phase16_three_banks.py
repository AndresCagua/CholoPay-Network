"""Aceptación Docker-only de identidad y aislamiento sintético entre bancos."""

import json
import os
import time
import urllib.error
import urllib.request


BANKS = (
    ("costa", "http://bank-core:8080", "BANCO_COSTA", os.environ["COSTA_BANK_API_TOKEN"]),
    ("sierra", "http://bank-core-sierra:8080", "BANCO_SIERRA", os.environ["SIERRA_BANK_API_TOKEN"]),
    ("oriente", "http://bank-core-oriente:8080", "BANCO_ORIENTE", os.environ["ORIENTE_BANK_API_TOKEN"]),
)


def request_identity(url: str, token: str) -> tuple[int, dict]:
    request = urllib.request.Request(
        f"{url}/internal/v1/bank-identity",
        headers={"Authorization": f"Bearer {token}"},
    )
    try:
        with urllib.request.urlopen(request, timeout=2) as response:
            return response.status, json.loads(response.read())
    except urllib.error.HTTPError as error:
        body = error.read().decode("utf-8")
        return error.code, json.loads(body) if body else {}
    except urllib.error.URLError:
        return 0, {}


def wait_for_banks() -> None:
    deadline = time.time() + 90
    while time.time() < deadline:
        if all(request_identity(url, token)[0] == 200 for _, url, _, token in BANKS):
            return
        time.sleep(2)
    raise AssertionError("los tres bancos no estuvieron disponibles dentro del tiempo límite")


def main() -> None:
    wait_for_banks()
    identities = {}
    for name, url, participant, token in BANKS:
        status, body = request_identity(url, token)
        assert status == 200, (name, status, body)
        assert body == {"participantId": participant}, (name, body)
        identities[name] = body["participantId"]

    assert identities == {
        "costa": "BANCO_COSTA",
        "sierra": "BANCO_SIERRA",
        "oriente": "BANCO_ORIENTE",
    }

    for name, url, _, _ in BANKS:
        for other_name, _, _, other_token in BANKS:
            if name == other_name:
                continue
            status, body = request_identity(url, other_token)
            assert status == 401, (name, other_name, status, body)

    print("PHASE16 PASS: tres identidades bancarias aisladas y tokens cruzados rechazados")


if __name__ == "__main__":
    main()
