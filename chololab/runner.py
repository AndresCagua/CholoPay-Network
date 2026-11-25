"""Ejecutor Docker de carga HTTP e inyección de fallos determinista."""
from __future__ import annotations

import concurrent.futures
import json
import math
import os
import statistics
import time
import urllib.error
import urllib.request
from dataclasses import dataclass


@dataclass(frozen=True)
class Result:
    index: int
    status: int | None
    latency_ms: float
    error: str | None = None


def request_once(target_url: str, index: int, failure_every: int, headers: dict[str, str] | None = None) -> Result:
    started = time.perf_counter()
    if failure_every > 0 and (index + 1) % failure_every == 0:
        return Result(index, None, (time.perf_counter() - started) * 1000, "injected_failure")
    try:
        request = urllib.request.Request(target_url, headers=headers or {})
        with urllib.request.urlopen(request, timeout=5) as response:
            response.read()
            return Result(index, response.status, (time.perf_counter() - started) * 1000)
    except (urllib.error.URLError, TimeoutError, OSError) as exc:
        return Result(index, None, (time.perf_counter() - started) * 1000, type(exc).__name__)


def percentile(values: list[float], percentile_value: int) -> float:
    ordered = sorted(values)
    rank = max(0, math.ceil(percentile_value * len(ordered) / 100) - 1)
    return round(ordered[rank], 3)


def run(target_url: str, requests: int, concurrency: int, failure_every: int,
        *, expected_status: int | None = None, headers: dict[str, str] | None = None,
        logical_seed: str = "unspecified", commit: str = "unknown",
        scenario: str = "synthetic_http_load") -> dict[str, object]:
    if requests < 1 or concurrency < 1 or failure_every < 0:
        raise ValueError("requests y concurrency deben ser positivos; failure_every no puede ser negativo")
    started = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as pool:
        results = list(pool.map(lambda i: request_once(target_url, i, failure_every, headers), range(requests)))
    elapsed_ms = (time.perf_counter() - started) * 1000
    latencies = [result.latency_ms for result in results]
    successful = sum(
        result.status == expected_status if expected_status is not None
        else result.status is not None and 200 <= result.status < 300
        for result in results
    )
    failed = requests - successful
    injected = sum(result.error == "injected_failure" for result in results)
    elapsed = elapsed_ms / 1000
    rate = round(requests / elapsed, 3) if elapsed else 0
    summary = {
        "target_url": target_url,
        "requests": requests,
        "concurrency": concurrency,
        "failure_every": failure_every,
        "successful": successful,
        "failed": failed,
        "injected_failures": injected,
        "elapsed_ms": round(elapsed_ms, 3),
        "throughput_per_second": rate,
        "latency_ms": {
            "min": round(min(latencies), 3),
            "median": round(statistics.median(latencies), 3),
            "max": round(max(latencies), 3),
            "p95": percentile(latencies, 95),
            "p99": percentile(latencies, 99),
        },
    }
    summary["metadata"] = {
        "target": target_url,
        "requests": requests,
        "concurrency": concurrency,
        "duration_ms": summary["elapsed_ms"],
        "commit": commit,
        "logical_seed": logical_seed,
        "expected_status": expected_status if expected_status is not None else "2xx",
        "errors": failed,
    }
    summary["red"] = {
        "rate_requests_per_second": rate,
        "errors": failed,
        "error_rate": round(failed / requests, 3),
        "duration_ms": summary["elapsed_ms"],
        "target": target_url,
        "requests": requests,
        "concurrency": concurrency,
        "commit": commit,
        "logical_seed": logical_seed,
        "expected_status": expected_status if expected_status is not None else "2xx",
    }
    return summary


def emit_log(event: str, **fields: object) -> None:
    """Emite un registro legible por máquina sin agregar credenciales ni datos de solicitudes."""
    print(json.dumps({"event": event, **fields}, sort_keys=True))


def main() -> None:
    target_url = os.getenv("TARGET_URL", "http://network:8080/health/live")
    requests = int(os.getenv("REQUESTS", "100"))
    concurrency = int(os.getenv("CONCURRENCY", "4"))
    failure_every = int(os.getenv("FAILURE_EVERY", "0"))
    expected_status_raw = os.getenv("EXPECTED_STATUS")
    expected_status = int(expected_status_raw) if expected_status_raw else None
    logical_seed = os.getenv("LOGICAL_SEED", "unspecified")
    commit = os.getenv("GIT_COMMIT", "unknown")
    scenario = os.getenv("SCENARIO", "synthetic_http_load")
    emit_log(
        "run_started",
        scenario=scenario,
        requests=requests,
        concurrency=concurrency,
        failure_every=failure_every,
    )
    summary = run(target_url, requests, concurrency, failure_every,
                  expected_status=expected_status, logical_seed=logical_seed,
                  commit=commit, scenario=scenario)
    emit_log("run_summary", scenario=scenario, summary=summary)


if __name__ == "__main__":
    main()
