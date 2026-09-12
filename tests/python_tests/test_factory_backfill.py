import os
import subprocess
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]
FACTORY = ROOT / "developer-factory" / "factory_agent.py"


def _skeleton(tmp: Path) -> None:
    (tmp / "tools").mkdir()
    (tmp / "config").mkdir()
    (tmp / "tests" / "python_tests").mkdir(parents=True)
    (tmp / "config" / "tool_registry.yaml").write_text(
        "version: 1\ntimezone: Asia/Hong_Kong\ntools: []\n",
        encoding="utf-8",
    )


def test_factory_registers_only_after_tests(tmp_path):
    _skeleton(tmp_path)
    env = os.environ.copy()
    env["WALLOP_ROOT"] = str(tmp_path)
    env["MOCK_MODE"] = "true"
    proc = subprocess.run(
        [
            sys.executable,
            str(FACTORY),
            "--name",
            "wiki_search",
            "--desc",
            "Keyword search over a vault",
            "--param",
            "query:string:required",
        ],
        cwd=ROOT,
        env=env,
        capture_output=True,
        text=True,
    )
    assert proc.returncode == 0, proc.stdout + proc.stderr
    assert (tmp_path / "tools" / "wiki_search.py").exists()
    data = yaml.safe_load((tmp_path / "config" / "tool_registry.yaml").read_text(encoding="utf-8"))
    names = [t["name"] for t in data["tools"]]
    assert "wiki_search" in names


def test_brief_uses_mock_client(tmp_path):
    _skeleton(tmp_path)
    env = os.environ.copy()
    env["WALLOP_ROOT"] = str(tmp_path)
    env["MOCK_MODE"] = "true"
    proc = subprocess.run(
        [sys.executable, str(FACTORY), "--brief", "Keyword search over a local markdown vault"],
        cwd=ROOT,
        env=env,
        capture_output=True,
        text=True,
    )
    assert proc.returncode == 0, proc.stdout + proc.stderr
    assert (tmp_path / "tools" / "generated_capability.py").exists()
