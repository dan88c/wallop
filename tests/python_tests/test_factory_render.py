from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "developer-factory"))

import factory_agent as factory  # noqa: E402


def test_render_contains_pydantic_models():
    src = factory.render_tool(
        "demo_tool",
        "Demo",
        [{"name": "title", "type": "string", "required": True, "annotation": "str", "default": None}],
    )
    assert "class DemoToolInput" in src
    assert "class DemoToolOutput" in src
    assert "def run(" in src
