from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "developer-factory"))
from privacy_guard import sanitize  # noqa: E402


def test_strips_email_and_token():
    raw = (
        "Create an event titled 'Dinner with Alice' "
        "token: sk-live-abcdef email me@home.example "
        "at C:\\Users\\Daniel\\LocalWiki\\notes.md"
    )
    out = sanitize(raw)
    assert out.ok
    payload = out.cloud_payload()
    assert "sk-live" not in payload
    assert "me@home.example" not in payload
    assert "Dinner with Alice" not in payload
    assert "LocalWiki" not in payload
    assert "Abstract functional requirement" in payload


def test_blocks_oauth_hint():
    out = sanitize("Use the oauth refresh_token from the desktop app")
    assert out.blocked
    try:
        out.cloud_payload()
        raise AssertionError("should have blocked")
    except PermissionError:
        pass
