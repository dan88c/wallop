"""Information Wall.

Sanitizes a local task before any payload is allowed to leave the machine.
Private calendar titles, notes, tokens, emails, and file paths must not cross
this boundary. The cloud developer model receives only an abstract functional
requirement.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field

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


@dataclass
class SanitizeResult:
    ok: bool
    abstract: str
    redactions: list[str] = field(default_factory=list)
    blocked: bool = False
    reason: str | None = None

    def cloud_payload(self) -> str:
        if self.blocked:
            raise PermissionError(self.reason or "blocked by information wall")
        return self.abstract


def sanitize(task: str, *, allow_examples: bool = True) -> SanitizeResult:
    if not task or not task.strip():
        return SanitizeResult(ok=False, abstract="", blocked=True, reason="empty task")

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


def _to_abstract(text: str, *, allow_examples: bool) -> str:
    compact = re.sub(r"\s+", " ", text).strip()
    if len(compact) > 400:
        compact = compact[:400].rstrip() + "…"
    prefix = "Abstract functional requirement (no private data): "
    return prefix + compact


def assert_safe_for_cloud(task: str) -> str:
    result = sanitize(task)
    return result.cloud_payload()
