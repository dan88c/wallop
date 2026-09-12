from importlib.machinery import SourceFileLoader
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
mod = SourceFileLoader(
    "calendar_gateway", str(ROOT / "tools" / "calendar_gateway.py")
).load_module()


def test_mock_create_never_opens_socket(monkeypatch):
    monkeypatch.setenv("MOCK_MODE", "true")

    class Boom:
        def __init__(self, *args, **kwargs):
            raise AssertionError("httpx must not run in MOCK_MODE")

    monkeypatch.setattr(mod.httpx, "Client", Boom)
    out = mod.run(
        mod.CalendarGatewayInput(action="create", start="2026-09-13T10:00:00+08:00", title="demo")
    )
    assert out.ok
    assert out.data["mock"] is True
    assert out.data["id"] == "evt_demo_1"


def test_list_is_mock_by_default(monkeypatch):
    monkeypatch.delenv("MOCK_MODE", raising=False)
    out = mod.run(mod.CalendarGatewayInput(action="list"))
    assert out.ok
    assert out.data["mock"] is True
