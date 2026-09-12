"""Read the weekly exception-time log (holidays, overrides, on-call)."""

from __future__ import annotations

import json
import os
import sys
from datetime import date, datetime, timedelta
from pathlib import Path
from typing import Any
from zoneinfo import ZoneInfo

from pydantic import BaseModel, Field


class TimeOpsReaderInput(BaseModel):
    week_of: str | None = None
    path: str | None = None


class TimeOpsReaderOutput(BaseModel):
    ok: bool
    week_start: str
    week_end: str
    lines: list[str] = Field(default_factory=list)
    path: str
    error: str | None = None


def _tz() -> ZoneInfo:
    return ZoneInfo(os.environ.get("HARNESS_TZ", "Asia/Hong_Kong"))


def _week_bounds(week_of: str | None) -> tuple[date, date]:
    if week_of:
        d = date.fromisoformat(week_of)
    else:
        d = datetime.now(_tz()).date()
    start = d - timedelta(days=d.weekday())
    return start, start + timedelta(days=6)


def default_log_path() -> Path:
    env = os.environ.get("TIME_OPS_LOG_PATH")
    if env:
        return Path(env)
    return Path(__file__).resolve().parents[1] / "data" / "time_ops.log"


def run(params: TimeOpsReaderInput) -> TimeOpsReaderOutput:
    start, end = _week_bounds(params.week_of)
    path = Path(params.path) if params.path else default_log_path()
    if not path.exists():
        return TimeOpsReaderOutput(
            ok=True,
            week_start=start.isoformat(),
            week_end=end.isoformat(),
            lines=[],
            path=str(path),
            error=None,
        )
    matched: list[str] = []
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        token = line.split(",", 1)[0].strip()
        try:
            day = date.fromisoformat(token[:10])
        except ValueError:
            matched.append(line)
            continue
        if start <= day <= end:
            matched.append(line)
    return TimeOpsReaderOutput(
        ok=True,
        week_start=start.isoformat(),
        week_end=end.isoformat(),
        lines=matched,
        path=str(path),
    )


if __name__ == "__main__":
    raw: dict[str, Any] = json.loads(sys.stdin.read() or "{}")
    print(run(TimeOpsReaderInput.model_validate(raw)).model_dump_json())
