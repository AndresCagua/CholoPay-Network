"""Deterministic Docker-only HTTP load and fault-injection runner."""
from __future__ import annotations

import concurrent.futures
import json
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


def request_once(target_url: str, index: int, failure_every: int) -> Result:
    started = time.perf_counter()
    if failure_every > 0 and (index + 1) % failure_every == 0:
        return Result(index, None, (time.perf_counter() - started) * 1000, "injected_failure")
    try:
        with urllib.request.urlopen(target_url, timeout=5) as response:
            response.read()
            return Result(index, response.status, (time.perf_counter() - started) * 1000)
    except (urllib.error.URLError, TimeoutError, OSError) as exc:
        return Result(index, None, (time.perf_counter() - started) * 1000, type(exc).__name__)


def run(target_url: str, requests: int, concurrency: int, failure_every: int) -> dict[str, object]:
    if requests < 1 or concurrency < 1 or failure_every < 0:
        raise ValueError("requests and concurrency must be positive; failure_every cannot be negative")
    started = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as pool:
        results = list(pool.map(lambda i: request_once(target_url, i, failure_every), range(requests)))
    elapsed_ms = (time.perf_counter() - started) * 1000
    latencies = [result.latency_ms for result in results]
    successful = sum(result.status is not None and 200 <= result.status < 300 for result in results)
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
        },
    }
    summary["red"] = {
        "rate_requests_per_second": rate,
        "errors": failed,
        "error_rate": round(failed / requests, 3),
        "duration_ms": summary["elapsed_ms"],
    }
    return summary


def emit_log(event: str, **fields: object) -> None:
    """Emit one machine-readable record without adding credentials or request data."""
    print(json.dumps({"event": event, **fields}, sort_keys=True))


def main() -> None:
    target_url = os.getenv("TARGET_URL", "http://network:8080/health/live")
    requests = int(os.getenv("REQUESTS", "100"))
    concurrency = int(os.getenv("CONCURRENCY", "4"))
    failure_every = int(os.getenv("FAILURE_EVERY", "0"))
    emit_log(
        "run_started",
        scenario="synthetic_http_load",
        requests=requests,
        concurrency=concurrency,
        failure_every=failure_every,
    )
    summary = run(target_url, requests, concurrency, failure_every)
    emit_log("run_summary", scenario="synthetic_http_load", summary=summary)


if __name__ == "__main__":
    main()
