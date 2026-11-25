"""Docker-only P8.2 checks for private Prometheus/Grafana provisioning."""
import json
import os
import time
import urllib.request

prometheus = os.environ.get("PROMETHEUS_URL", "http://prometheus:9090")
grafana = os.environ.get("GRAFANA_URL", "http://grafana:3000")


def get(url):
    with urllib.request.urlopen(url, timeout=5) as response:
        return response.status, json.loads(response.read())


def wait(url, predicate, label):
    last = None
    for _ in range(45):
        try:
            status, payload = get(url)
            last = (status, payload)
            if predicate(payload):
                return payload
        except Exception as error:
            last = error
        time.sleep(1)
    raise SystemExit(f"{label} no estuvo listo: {last}")


targets = wait(
    prometheus + "/api/v1/targets",
    lambda body: all(t["health"] == "up" for t in body["data"]["activeTargets"])
    and {t["labels"]["job"] for t in body["data"]["activeTargets"]}
    >= {"cholonet-network", "cholonet-bank-core"},
    "targets Prometheus",
)
query = wait(
    prometheus + "/api/v1/query?query=cholonet_synthetic_acceptance_info",
    lambda body: len(body["data"]["result"]) >= 2,
    "métrica scrapeada",
)
datasources = wait(
    grafana + "/api/datasources",
    lambda body: any(d["name"] == "Prometheus" and d["type"] == "prometheus" for d in body),
    "datasource Grafana",
)
dashboards = wait(
    grafana + "/api/search?query=CholoNet",
    lambda body: any(d.get("uid") == "cholonet-local" for d in body),
    "dashboard Grafana",
)
print("P8.2 OBSERVABILITY PASS: Prometheus targets up y scrapeados; datasource y dashboard Grafana provisionados en red privada")
