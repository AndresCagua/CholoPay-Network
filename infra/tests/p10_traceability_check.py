#!/usr/bin/env python3
"""Docker-only structural checks for the P10 documentation slice."""
from __future__ import annotations

import argparse
import re
from pathlib import Path
from urllib.parse import unquote, urlparse

ROOT = Path(__file__).resolve().parents[2]
MATRIX = ROOT / "docs/traceability-matrix.md"
ADR_DIR = ROOT / "adr"
EXPECTED_OPENAPI = {
    "api/bank.openapi.yaml",
    "api/network.openapi.yaml",
    "api/gateway.openapi.yaml",
}
REQUIRED_COLUMNS = [
    "spec.md",
    "tarea ODD",
    "ADR",
    "código",
    "prueba",
    "evidencia Docker",
    "commit",
    "estado",
    "limitación",
]


def fail(message: str) -> None:
    raise SystemExit(f"P10 TRACEABILITY FAIL: {message}")


def links_in(markdown: str) -> list[str]:
    return re.findall(r"\[[^\]]+\]\(([^)]+)\)", markdown)


def check_links(markdown: str) -> None:
    for raw in links_in(markdown):
        parsed = urlparse(raw)
        if parsed.scheme or parsed.netloc:
            continue
        target = unquote(parsed.path)
        if not target:
            continue
        path = (MATRIX.parent / target).resolve()
        if not path.exists():
            fail(f"referencia inexistente en matriz: {raw}")


def check_compose(compose_config: Path | None) -> None:
    if compose_config is None:
        # The shell runner supplies docker compose config. This fallback keeps the
        # Python checker useful in a container while still checking the source.
        compose_config = ROOT / "infra/compose.yaml"
    text = compose_config.read_text(encoding="utf-8")
    if re.search(r"(?m)^\s+(ports|published):", text):
        fail(f"la configuración Compose contiene puertos publicados: {compose_config}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--compose-config", type=Path)
    args = parser.parse_args()

    if not MATRIX.is_file():
        fail("falta docs/traceability-matrix.md")
    matrix = MATRIX.read_text(encoding="utf-8")
    header = next((line for line in matrix.splitlines() if line.startswith("| Punto / requisito")), "")
    for column in REQUIRED_COLUMNS:
        if column.lower() not in header.lower():
            fail(f"falta columna requerida: {column}")
    rows = re.findall(r"^\| \*\*P([1-9])\*\*", matrix, re.MULTILINE)
    if sorted(rows) != list("123456789"):
        fail(f"cobertura P1-P9 incompleta: {rows}")
    check_links(matrix)
    required_current_evidence = {
        "P7": ["odd/tasks/phase-7-http-outbox-recovery.md", "infra/tests/p7_http_outbox_recovery.sh", "runbooks/p7-http-outbox-recovery.md"],
        "P8": ["odd/tasks/phase-8-bank-core-readiness-recovery.md", "odd/tasks/phase-8-observability-resilience.md", "odd/tasks/phase-8-health-contract.md", "infra/tests/p8_health_acceptance.sh", "infra/tests/p8_observability_acceptance.sh", "runbooks/p8-observability.md", "runbooks/p8-metrics.md"],
    }
    for point, references in required_current_evidence.items():
        row = next((line for line in matrix.splitlines() if line.startswith(f"| **{point}**")), "")
        for reference in references:
            if reference not in row:
                fail(f"{point} no referencia evidencia actual: {reference}")

    actual_adrs = {p.name for p in ADR_DIR.glob("[0-9][0-9][0-9][0-9]-*.md")}
    index = (ADR_DIR / "README.md").read_text(encoding="utf-8")
    for name in sorted(actual_adrs):
        if f"]({name})" not in index:
            fail(f"ADR no indexado: {name}")
    for name in EXPECTED_OPENAPI:
        if not (ROOT / name).is_file():
            fail(f"falta OpenAPI: {name}")
        if name not in (ROOT / "README.md").read_text(encoding="utf-8"):
            fail(f"OpenAPI no enlazado desde README: {name}")

    required_identifiers = ["event_id", "transfer_id", "correlation_id", "causation_id", "trace_id"]
    for identifier in required_identifiers:
        if identifier not in matrix:
            fail(f"no se documenta el metadato sintético: {identifier}")
    exclusions = [
        "no equivalen a spans",
        "no tracing distribuido",
        "OpenTelemetry",
        "W3C `traceparent`",
        "collectors",
        "Jaeger",
        "SLO/SLI",
        "monitoreo productivo",
    ]
    for phrase in exclusions:
        if phrase.lower() not in matrix.lower():
            fail(f"falta exclusión explícita: {phrase}")
    check_compose(args.compose_config)
    print(
        "P10 TRACEABILITY PASS: P1-P9, "
        f"{len(actual_adrs)} ADRs, {len(EXPECTED_OPENAPI)} OpenAPI, "
        "límites de tracing y Compose privado verificados"
    )


if __name__ == "__main__":
    main()
