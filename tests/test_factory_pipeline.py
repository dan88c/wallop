"""End-to-end factory pipeline: wall -> render -> registry shape."""

from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "developer-factory"))

from privacy_guard import sanitize  # noqa: E402
import factory_agent as factory  # noqa: E402


def test_pipeline_sanitizes_then_renders():
    brief = "Write a parser for ISO time ranges. Ignore title='Private standup'."
    wall = sanitize(brief)
    assert wall.ok
    src = factory.render_tool(
        "iso_range_parser",
        wall.abstract,
        [{"name": "text", "type": "string", "required": True, "annotation": "str", "default": None}],
    )
    assert "class IsoRangeParserToolInput" in src
    assert "Private standup" not in src
