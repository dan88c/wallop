#!/usr/bin/env python3
"""Credential-free end-to-end demo. No APIs, no private hardware."""

from __future__ import annotations

import os
import subprocess
import sys
from datetime import datetime, timedelta
from pathlib import Path
from zoneinfo import ZoneInfo

ROOT = Path(__file__).resolve().parents[1]
os.chdir(ROOT)
os.environ.setdefault("MOCK_MODE", "true")
os.environ.setdefault("VAULT_PATH", str(ROOT / "sandbox" / "vault"))
os.environ.setdefault("HARNESS_TZ", "Asia/Hong_Kong")
os.environ.setdefault("TIME_OPS_LOG_PATH", str(ROOT / "data" / "time_ops.log"))
sys.path.insert(0, str(ROOT / "tools"))
sys.path.insert(0, str(ROOT / "developer-factory"))

from calendar_gateway import CalendarGatewayInput, run as cal_run  # noqa: E402
from privacy_guard import sanitize  # noqa: E402
from time_ops_reader import TimeOpsReaderInput, run as time_run  # noqa: E402


def step(title: str) -> None:
    print(f"\n== {title} ==")


def wallop_bin() -> Path:
    name = "wallop.exe" if os.name == "nt" else "wallop"
    return ROOT / "bin" / name


def main() -> int:
    print("wallop demo (MOCK_MODE=true, no credentials)")

    step("1. Information Wall")
    dirty = (
        "Schedule dinner titled 'Private standup' token: sk-live-demo "
        "email me@home.example at C:\\Users\\Demo\\LocalWiki\\notes.md. "
        "Need a parser for ISO time ranges."
    )
    wall = sanitize(dirty)
    print(wall.abstract)
    print("redactions:", wall.redactions, "blocked:", wall.blocked)
    assert wall.ok and "sk-live-demo" not in wall.abstract

    step("2. time_ops_reader")
    ops = time_run(TimeOpsReaderInput(week_of="2026-09-09"))
    print(ops.model_dump_json(indent=2))

    step("3. calendar_gateway mock create")
    tz = ZoneInfo(os.environ["HARNESS_TZ"])
    start = (datetime.now(tz) + timedelta(days=1)).replace(minute=0, second=0, microsecond=0)
    cal = cal_run(
        CalendarGatewayInput(
            action="create",
            start=start.isoformat(),
            title="demo-block",
        )
    )
    print(cal.model_dump_json(indent=2))
    assert cal.ok and cal.data.get("mock") is True

    step("4. operator TOC + graph (if wallop binary exists)")
    binary = wallop_bin()
    if not binary.exists():
        subprocess.run(
            ["go", "build", "-o", str(binary), "./cmd/harness"],
            cwd=ROOT / "core-operator",
            check=False,
        )
    if binary.exists():
        print(subprocess.check_output([str(binary), "toc"], text=True))
        graph = subprocess.check_output(
            [str(binary), "graph", "--vault", os.environ["VAULT_PATH"]],
            text=True,
        )
        print(graph[:1200])
    else:
        print("Go toolchain not found; skipped compiled operator. Python path still succeeded.")

    print("\nDEMO OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
