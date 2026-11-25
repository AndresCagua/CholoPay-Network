"""Docker-only P8.2 checks for private Prometheus/Grafana provisioning and recovery."""
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request

prometheus = os.environ.get("PROMETHEUS_URL", "http://prometheus:9090")
grafana = os.environ.get("GRAFANA_URL", "http://grafana:3000")
bank_core = os.environ.get("BANK_CORE_URL", "http://bank-core:8080")
mode = os.environ.get("P8_MODE", "baseline")


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
    raise SystemExit(f"P8.2 ERROR: {label} no estuvo listo; último diagnóstico: {last}")


def http_status(url, timeout=5):
    try:
        with urllib.request.urlopen(url, timeout=timeout) as response:
            return response.status
    except urllib.error.HTTPError as error:
        return error.code
    except Exception:
        return 599


def wait_status(url, expected, label, timeout=5):
    last = None
    for _ in range(45):
        last = http_status(url, timeout=timeout)
        if last == expected:
            return
        time.sleep(1)
    raise SystemExit(f"P8.2 ERROR: {label} esperaba HTTP {expected}; último estado: {last}")


def query(metric, label):
    encoded = urllib.parse.quote(metric, safe="")
    return wait(
        prometheus + "/api/v1/query?query=" + encoded,
        lambda body: body.get("status") == "success" and len(body["data"]["result"]) >= 1,
        label,
    )


def assert_targets(label):
    def ready(body):
        targets = body["data"]["activeTargets"]
        by_job = {t["labels"].get("job"): t for t in targets}
        return all(
            by_job.get(job, {}).get("health") == "up"
            and by_job.get(job, {}).get("lastError", "") == ""
            for job in ("cholonet-network", "cholonet-bank-core")
        )

    return wait(prometheus + "/api/v1/targets", ready, label)


def assert_scrape(label):
    assert_targets(label + ": targets")
    query('up{job="cholonet-bank-core"}', label + ": target up")
    query('cholonet_process_info{service="bank-core"}', label + ": process metric")
    query('cholonet_synthetic_acceptance_info{scope="local"}', label + ": acceptance metric")


if mode not in {"baseline", "bank-db-down", "recovered"}:
    raise SystemExit(f"P8.2 ERROR: P8_MODE inválido: {mode!r}")

if mode == "baseline":
    assert_targets("baseline Prometheus")
    query("cholonet_synthetic_acceptance_info", "baseline métrica scrapeada")
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
    print("P8.2 BASELINE PASS: targets, métrica sintética, datasource y dashboard verificados")
elif mode == "bank-db-down":
    wait_status(bank_core + "/health/live", 200, "bank-core live durante outage")
    wait_status(bank_core + "/health/ready", 503, "bank-core ready durante outage", timeout=35)
    assert_scrape("outage")
    print("P8.2 OUTAGE PASS: bank-core live=200, ready=503 y scrape/métricas consultables con costa-db detenido")
else:
    wait_status(bank_core + "/health/live", 200, "bank-core live tras recovery")
    wait_status(bank_core + "/health/ready", 200, "bank-core ready tras recovery")
    assert_scrape("recovery")
    print("P8.2 RECOVERY PASS: bank-core ready=200 y scrape/métricas consultables tras recuperar costa-db")
