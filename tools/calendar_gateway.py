"""DEMO ONLY — pipeline sample, not a live calendar client.

Shows registry → guard → stdin JSON → Pydantic → MOCK_MODE.
Do not use this script against a real calendar. OAuth and live writes
belong on an edge webhook outside this repo.
"""

from __future__ import annotations

import json
import os
import sys
from typing import Any, Literal

import httpx
from pydantic import BaseModel, Field

DEMO_ONLY = True


class CalendarGatewayInput(BaseModel):
    action: Literal["list", "create", "update", "delete"]
    start: str | None = None
    end: str | None = None
    title: str | None = None
    event_id: str | None = None
    extra: dict[str, Any] = Field(default_factory=dict)


class CalendarGatewayOutput(BaseModel):
    ok: bool
    status_code: int | None = None
    data: dict[str, Any] = Field(default_factory=dict)
    error: str | None = None
    demo: bool = True


def mock_mode() -> bool:
    return os.environ.get("MOCK_MODE", "true").strip().lower() in {"1", "true", "yes", "on"}


def gateway_url() -> str:
    return os.environ.get(
        "EDGE_CALENDAR_WEBHOOK",
        os.environ.get("CALENDAR_GATEWAY_URL", "http://127.0.0.1:8088/webhook/calendar"),
    )


def run(params: CalendarGatewayInput) -> CalendarGatewayOutput:
    payload = params.model_dump(exclude_none=True)
    if mock_mode():
        return CalendarGatewayOutput(
            ok=True,
            status_code=200,
            data={"mock": True, "demo": True, "echo": payload, "id": "evt_demo_1"},
            demo=True,
        )
    url = gateway_url()
    try:
        with httpx.Client(timeout=15.0) as client:
            resp = client.post(url, json=payload)
        body: dict[str, Any]
        try:
            parsed = resp.json()
            body = parsed if isinstance(parsed, dict) else {"result": parsed}
        except ValueError:
            body = {"text": resp.text}
        body = {**body, "demo": True}
        return CalendarGatewayOutput(
            ok=resp.is_success,
            status_code=resp.status_code,
            data=body,
            error=None if resp.is_success else f"webhook HTTP {resp.status_code}",
            demo=True,
        )
    except httpx.HTTPError as exc:
        return CalendarGatewayOutput(ok=False, error=str(exc), demo=True)


if __name__ == "__main__":
    raw = json.loads(sys.stdin.read() or "{}")
    print(run(CalendarGatewayInput.model_validate(raw)).model_dump_json())
