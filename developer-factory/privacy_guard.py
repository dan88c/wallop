"""Information Wall.

Sanitizes a local task before any payload is allowed to leave the machine.
Private calendar titles, notes, tokens, emails, and file paths must not cross
this boundary. The cloud developer model receives only an abstract functional
requirement.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from typing import Any

_SECRET = re.compile(
    r"(?i)\b(api[_-]?key|secret|token|password|passwd|bearer|authorization)\b"
    r"\s*[:=]\s*\S+"
)
_BEARER = re.compile(r"(?i)\bbearer\s+[A-Za-z0-9\-._~+/]+=*")
_EMAIL = re.compile(r"\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b")
_ISO_DT = re.compile(r"\b\d{4}-\d{2}-\d{2}(?:[T ]\d{2}:\d{2}(?::\d{2})?(?:Z|[+\-]\d{2}:?\d{2})?)?\b")
_PATH = re.compile(r"(?i)(?:[A-Za-z]:\\|\\\\|/(?:Users|home|Wiki|LocalWiki|documents)/)[^\s\"']+")
_WIKILINK = re.compile(r"\[\[[^\]]+\]\]")
_QUOTED = re.compile(
    r"(?i)(?:title|titled|event|note|notes|summary|subject)\s*[:=]?\s*[\"'][^\"']+[\"']"
)

_BLOCK_HINTS = (
    "calendar title",
    "meeting with",
    "personal note",
    "oauth",
    "refresh_token",
    "client_secret",
)

_CALENDAR_KEYS = {"title", "start", "end", "event_id", "action", "attendees"}
_WRITE_ACTIONS = {"create", "update", "delete", "write", "set"}


@dataclass
class SanitizeResult:
    ok: bool
    abstract: str
    redactions: list[str] = field(default_factory=list)
    blocked: bool = False
    reason: str | None = None
    spec: dict[str, Any] | None = None

    def cloud_payload(self) -> str:
        if self.blocked:
            raise PermissionError(self.reason or "blocked by information wall")
        return self.abstract

    def cloud_spec(self) -> dict[str, Any]:
        if self.blocked:
            raise PermissionError(self.reason or "blocked by information wall")
        if self.spec is not None:
            return self.spec
        return {"intent": "generic_requirement", "entities": [], "abstract": self.abstract}


def sanitize(task: str, *, allow_examples: bool = True) -> SanitizeResult:
    if not task or not task.strip():
        return SanitizeResult(ok=False, abstract="", blocked=True, reason="empty task")

    parsed = _try_json_object(task)
    if parsed is not None:
        return sanitize_payload(parsed)

    lower = task.lower()
    redactions: list[str] = []
    text = task

    def _sub(pattern: re.Pattern[str], label: str, repl: str) -> None:
        nonlocal text
        found = pattern.findall(text)
        if found:
            redactions.append(f"{label}:{len(found)}")
            text = pattern.sub(repl, text)

    _sub(_BEARER, "bearer", "[REDACTED_TOKEN]")
    _sub(_SECRET, "secret", "[REDACTED_SECRET]")
    _sub(_EMAIL, "email", "[REDACTED_EMAIL]")
    _sub(_PATH, "path", "[REDACTED_PATH]")
    _sub(_WIKILINK, "wikilink", "[NOTE_REF]")
    _sub(_QUOTED, "quoted-pii", "[REDACTED_TITLE]")
    _sub(_ISO_DT, "datetime", "[TIMESTAMP]")

    for hint in _BLOCK_HINTS:
        if hint in lower and "write a" not in lower:
            if hint in {"oauth", "refresh_token", "client_secret"}:
                return SanitizeResult(
                    ok=False,
                    abstract="",
                    redactions=redactions + [hint],
                    blocked=True,
                    reason=f"information wall blocked credential hint: {hint}",
                )

    abstract = _to_abstract(text, allow_examples=allow_examples)
    return SanitizeResult(ok=True, abstract=abstract, redactions=redactions, blocked=False)


def sanitize_payload(payload: dict[str, Any]) -> SanitizeResult:
    """Drop private field values. Cloud sees intent only, never titles or names."""
    keys = {str(k).lower() for k in payload}
    redactions: list[str] = []
    if payload.get("title") not in (None, ""):
        redactions.append("title:1")
    for key in ("start", "end", "event_id", "attendees", "notes", "note", "summary"):
        if payload.get(key) not in (None, "", [], {}):
            redactions.append(f"{key}:1")

    action = str(payload.get("action") or "").lower()
    if keys & _CALENDAR_KEYS:
        intent = "calendar_write" if action in _WRITE_ACTIONS else "calendar_check"
    else:
        intent = "generic_requirement"

    spec = {"intent": intent, "entities": []}
    abstract = _to_abstract(json.dumps(spec, ensure_ascii=False), allow_examples=False)
    return SanitizeResult(
        ok=True,
        abstract=abstract,
        redactions=redactions,
        blocked=False,
        spec=spec,
    )


def _try_json_object(task: str) -> dict[str, Any] | None:
    raw = task.strip()
    if not raw.startswith("{") or not raw.endswith("}"):
        return None
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return None
    return data if isinstance(data, dict) else None


def _to_abstract(text: str, *, allow_examples: bool) -> str:
    compact = re.sub(r"\s+", " ", text).strip()
    if len(compact) > 400:
        compact = compact[:400].rstrip() + "…"
    prefix = "Abstract functional requirement (no private data): "
    return prefix + compact


def assert_safe_for_cloud(task: str) -> str:
    result = sanitize(task)
    return result.cloud_payload()
