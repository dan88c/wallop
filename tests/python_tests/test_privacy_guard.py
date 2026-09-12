from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "developer-factory"))
from privacy_guard import sanitize, sanitize_payload  # noqa: E402


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


def test_calendar_title_becomes_intent_only():
    raw = {"title": "go to the hospital with the boss"}
    out = sanitize_payload(raw)
    assert out.ok
    assert out.cloud_spec() == {"intent": "calendar_check", "entities": []}
    leaked = out.cloud_payload() + str(out.cloud_spec())
    assert "hospital" not in leaked.lower()
    assert "boss" not in leaked.lower()
    assert "title" not in out.cloud_spec()


def test_sanitize_parses_json_string_title():
    out = sanitize('{"title": "go to the hospital with the boss"}')
    assert out.cloud_spec() == {"intent": "calendar_check", "entities": []}
    payload = out.cloud_payload().lower()
    assert "hospital" not in payload
    assert "boss" not in payload
