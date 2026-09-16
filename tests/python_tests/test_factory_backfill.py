import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
FACTORY = ROOT / "developer-factory" / "factory_agent.py"


def _skeleton(tmp: Path) -> None:
    (tmp / "tools").mkdir()
    (tmp / "config").mkdir()
    (tmp / "tests" / "python_tests").mkdir(parents=True)


def test_factory_writes_tool_without_yaml_catalog(tmp_path):
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
            "--skip-test",
        ],
        cwd=str(tmp_path),
        env=env,
        capture_output=True,
        text=True,
    )
    assert proc.returncode == 0, proc.stdout + proc.stderr
    assert (tmp_path / "tools" / "wiki_search.py").exists()
    assert not (tmp_path / "config" / "tool_registry.yaml").exists()
    assert not (ROOT / "config" / "tool_registry.yaml").exists()


def test_brief_uses_mock_client(tmp_path):
    _skeleton(tmp_path)
    env = os.environ.copy()
    env["WALLOP_ROOT"] = str(tmp_path)
    env["MOCK_MODE"] = "true"
    proc = subprocess.run(
        [
            sys.executable,
            str(FACTORY),
            "--brief",
            "Keyword search over a local markdown vault",
            "--skip-test",
        ],
        cwd=str(tmp_path),
        env=env,
        capture_output=True,
        text=True,
    )
    assert proc.returncode == 0, proc.stdout + proc.stderr
    assert (tmp_path / "tools" / "generated_capability.py").exists()
