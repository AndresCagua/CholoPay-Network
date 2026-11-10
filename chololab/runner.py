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
    injected = sum(result.error == "injected_failure" for result in results)
    summary = {
        "target_url": target_url,
        "requests": requests,
        "concurrency": concurrency,
        "failure_every": failure_every,
        "successful": successful,
        "failed": requests - successful,
        "injected_failures": injected,
        "elapsed_ms": round(elapsed_ms, 3),
        "throughput_per_second": round(requests / (elapsed_ms / 1000), 3) if elapsed_ms else 0,
        "latency_ms": {
            "min": round(min(latencies), 3),
            "median": round(statistics.median(latencies), 3),
            "max": round(max(latencies), 3),
        },
    }
    return summary


def main() -> None:
    target_url = os.getenv("TARGET_URL", "http://network:8080/health/live")
    requests = int(os.getenv("REQUESTS", "100"))
    concurrency = int(os.getenv("CONCURRENCY", "4"))
    failure_every = int(os.getenv("FAILURE_EVERY", "0"))
    print(json.dumps(run(target_url, requests, concurrency, failure_every), sort_keys=True))


if __name__ == "__main__":
    main()
