import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "developer-factory"))

from cloud_client import propose  # noqa: E402


def test_mock_propose_is_offline(monkeypatch):
    monkeypatch.setenv("MOCK_MODE", "true")
    monkeypatch.setenv("CLOUD_DEVELOPER_URL", "http://127.0.0.1:9/should-not-hit")
    spec = propose("Abstract functional requirement: keyword search over a vault")
    assert spec.source == "mock"
    assert spec.name == "generated_capability"
    assert spec.params[0]["name"] == "query"


def test_empty_abstract_rejected():
    try:
        propose("  ")
    except ValueError:
        return
    raise AssertionError("expected ValueError")
