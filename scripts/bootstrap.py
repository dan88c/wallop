#!/usr/bin/env python3
"""Clone-and-run helper for macOS, Linux, and Windows (no Make required).

  python scripts/bootstrap.py
  py -3 scripts\\bootstrap.py          # Windows
"""

from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
import venv
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VENV = ROOT / ".venv"
REQ = ROOT / "developer-factory" / "requirements.txt"


def venv_python() -> Path:
    if os.name == "nt":
        return VENV / "Scripts" / "python.exe"
    return VENV / "bin" / "python"


def run(cmd: list[str], *, cwd: Path | None = None, env: dict[str, str] | None = None, check: bool = True) -> int:
    print("+", " ".join(cmd))
    proc = subprocess.run(cmd, cwd=cwd or ROOT, env=env, check=False)
    if check and proc.returncode != 0:
        raise SystemExit(proc.returncode)
    return proc.returncode


def ensure_venv() -> Path:
    py = venv_python()
    if not py.exists():
        print(f"creating venv at {VENV}")
        venv.EnvBuilder(with_pip=True).create(VENV)
    return py


def install_deps(py: Path) -> None:
    run([str(py), "-m", "pip", "install", "--upgrade", "pip"], check=True)
    run([str(py), "-m", "pip", "install", "-r", str(REQ)], check=True)


def test_py(py: Path) -> int:
    return run(
        [str(py), "-m", "pytest", "tests/python_tests", "tests/test_factory_pipeline.py", "-q"],
        check=False,
    )


def test_go() -> int:
    go = shutil.which("go")
    if not go:
        print("Go not on PATH; skipping Go tests (install Go 1.22+ for the wallop CLI)")
        return 0
    code = run([go, "test", "./..."], cwd=ROOT / "core-operator", check=False)
    if code != 0:
        return code
    return run([go, "test", "./..."], cwd=ROOT / "tests" / "go_tests", check=False)


def build_go() -> int:
    go = shutil.which("go")
    if not go:
        print("Go not on PATH; skipped wallop binary")
        return 0
    name = "wallop.exe" if os.name == "nt" else "wallop"
    out_root = ROOT / "bin"
    out_go = ROOT / "core-operator" / "bin"
    out_root.mkdir(parents=True, exist_ok=True)
    out_go.mkdir(parents=True, exist_ok=True)
    code = run([go, "build", "-o", str(out_root / name), "./cmd/harness"], cwd=ROOT / "core-operator", check=False)
    if code != 0:
        return code
    legacy = "harness.exe" if os.name == "nt" else "harness"
    return run([go, "build", "-o", str(out_go / legacy), "./cmd/harness"], cwd=ROOT / "core-operator", check=False)


def demo(py: Path) -> int:
    env = os.environ.copy()
    env.setdefault("MOCK_MODE", "true")
    env.setdefault("VAULT_PATH", str(ROOT / "sandbox" / "vault"))
    env.setdefault("HARNESS_TZ", "Asia/Hong_Kong")
    env.setdefault("TIME_OPS_LOG_PATH", str(ROOT / "data" / "time_ops.log"))
    return run([str(py), str(ROOT / "scripts" / "demo.py")], env=env, check=False)


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Install deps, test, and demo wallop")
    p.add_argument("--deps", action="store_true", help="venv + pip only")
    p.add_argument("--test", action="store_true", help="run tests only (implies --deps)")
    p.add_argument("--demo", action="store_true", help="run demo only (implies --deps)")
    p.add_argument("--build", action="store_true", help="compile wallop if go is installed")
    args = p.parse_args(argv)

    py = ensure_venv()
    install_deps(py)
    selected = args.test or args.demo or args.build
    if args.deps and not selected:
        print("deps ready:", py)
        return 0
    do_all = not selected and not args.deps

    if do_all or args.build:
        code = build_go()
        if code != 0:
            return code
    if do_all or args.test:
        code = test_py(py)
        if code != 0:
            return code
        code = test_go()
        if code != 0:
            return code
    if do_all or args.demo:
        code = demo(py)
        if code != 0:
            return code
    print("bootstrap OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
