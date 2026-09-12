from importlib.machinery import SourceFileLoader
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
mod = SourceFileLoader(
    "time_ops_reader", str(ROOT / "tools" / "time_ops_reader.py")
).load_module()


def test_reads_week_window(tmp_path):
    log = tmp_path / "time_ops.log"
    log.write_text(
        "2026-09-07,holiday,mid-autumn\n2026-09-20,override,skip\n",
        encoding="utf-8",
    )
    out = mod.run(mod.TimeOpsReaderInput(week_of="2026-09-09", path=str(log)))
    assert out.ok
    assert out.week_start == "2026-09-07"
    assert out.week_end == "2026-09-13"
    assert len(out.lines) == 1
    assert "mid-autumn" in out.lines[0]


def test_missing_file_is_empty_ok(tmp_path):
    missing = tmp_path / "nope.log"
    out = mod.run(mod.TimeOpsReaderInput(week_of="2026-09-09", path=str(missing)))
    assert out.ok
    assert out.lines == []
