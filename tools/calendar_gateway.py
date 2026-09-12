"""Call the edge calendar webhook. No OAuth credentials live in this process."""

from __future__ import annotations

import json
import os
import sys
from typing import Any, Literal

import httpx
from pydantic import BaseModel, Field


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
            data={"mock": True, "echo": payload, "id": "evt_demo_1"},
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
        return CalendarGatewayOutput(
            ok=resp.is_success,
            status_code=resp.status_code,
            data=body,
            error=None if resp.is_success else f"webhook HTTP {resp.status_code}",
        )
    except httpx.HTTPError as exc:
        return CalendarGatewayOutput(ok=False, error=str(exc))


if __name__ == "__main__":
    raw = json.loads(sys.stdin.read() or "{}")
    print(run(CalendarGatewayInput.model_validate(raw)).model_dump_json())
