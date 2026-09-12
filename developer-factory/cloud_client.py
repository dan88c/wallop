"""Cloud developer dispatcher.

Default is MOCK_MODE=true: no sockets. A real HTTP call happens only when
MOCK_MODE is false and CLOUD_DEVELOPER_URL is set. The payload must already
have passed privacy_guard.sanitize().
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field
from typing import Any


def mock_mode() -> bool:
    return os.environ.get("MOCK_MODE", "true").strip().lower() in {"1", "true", "yes", "on"}


@dataclass
class Proposal:
    source: str
    name: str
    desc: str
    params: list[dict[str, Any]] = field(default_factory=list)
    notes: str = ""


def propose(abstract: str) -> Proposal:
    """Turn a sanitized abstract into a tool spec.

    Mock path is deterministic and offline. Live path POSTs JSON to
    CLOUD_DEVELOPER_URL and expects {name, desc, params}.
    """
    if not abstract or not abstract.strip():
        raise ValueError("empty abstract")

    if mock_mode() or not os.environ.get("CLOUD_DEVELOPER_URL"):
        return Proposal(
            source="mock",
            name="generated_capability",
            desc=abstract.strip()[:240],
            params=[{"name": "query", "type": "string", "required": True}],
            notes="Deterministic mock; no network.",
        )

    import httpx  # local import: mock path never needs it

    url = os.environ["CLOUD_DEVELOPER_URL"].strip()
    headers = {"Content-Type": "application/json"}
    token = os.environ.get("CLOUD_DEVELOPER_TOKEN", "").strip()
    if token:
        headers["Authorization"] = f"Bearer {token}"
    with httpx.Client(timeout=30.0) as client:
        resp = client.post(url, json={"abstract": abstract}, headers=headers)
        resp.raise_for_status()
        body = resp.json()
    if not isinstance(body, dict):
        raise ValueError("cloud developer returned non-object JSON")
    return Proposal(
        source="cloud",
        name=str(body.get("name") or "generated_capability"),
        desc=str(body.get("desc") or abstract.strip()[:240]),
        params=list(body.get("params") or [{"name": "query", "type": "string", "required": True}]),
        notes=str(body.get("notes") or ""),
    )
