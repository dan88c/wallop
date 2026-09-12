#!/usr/bin/env python3
"""Clone-and-run helper for macOS, Linux, and Windows (no Make required).

  python scripts/bootstrap.py
  python scripts/bootstrap.py --download   # fetch wallop from GitHub Releases
  py -3 scripts\\bootstrap.py

If Go is missing: use a local dist/ file if present, otherwise download the
nightly release asset for this platform into bin/.
"""

from __future__ import annotations

import argparse
import json
import os
import platform
import shutil
import stat
import subprocess
import sys
import urllib.error
import urllib.request
import venv
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VENV = ROOT / ".venv"
REQ = ROOT / "developer-factory" / "requirements.txt"
RELEASE_TAG = os.environ.get("WALLOP_RELEASE", "nightly")
RELEASE_BASE = f"https://github.com/dan88c/wallop/releases/download/{RELEASE_TAG}"

_DIST_BINARIES = {
    ("linux", "x86_64"): "wallop-linux-amd64",
    ("linux", "amd64"): "wallop-linux-amd64",
    ("darwin", "arm64"): "wallop-darwin-arm64",
    ("darwin", "aarch64"): "wallop-darwin-arm64",
    ("windows", "amd64"): "wallop-windows-amd64.exe",
    ("windows", "x86_64"): "wallop-windows-amd64.exe",
}


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
        print("Go not on PATH; skipping Go tests")
        return 0
    code = run([go, "test", "./..."], cwd=ROOT / "core-operator", check=False)
    if code != 0:
        return code
    return run([go, "test", "./..."], cwd=ROOT / "tests" / "go_tests", check=False)


def dist_binary_name(system: str | None = None, machine: str | None = None) -> str | None:
    system = (system or platform.system()).lower()
    machine = (machine or platform.machine()).lower()
    return _DIST_BINARIES.get((system, machine))


def dest_binary_name(asset: str) -> str:
    return "wallop.exe" if asset.endswith(".exe") else "wallop"


def make_executable(path: Path) -> None:
    if os.name != "nt":
        path.chmod(path.stat().st_mode | stat.S_IEXEC | stat.S_IXGRP | stat.S_IXOTH)


def copy_into_bin(src: Path, asset: str) -> Path:
    dest = ROOT / "bin" / dest_binary_name(asset)
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, dest)
    make_executable(dest)
    return dest


def download_asset(asset: str) -> Path:
    url = f"{RELEASE_BASE}/{asset}"
    dest = ROOT / "bin" / dest_binary_name(asset)
    dest.parent.mkdir(parents=True, exist_ok=True)
    print(f"[INFO] downloading {url}")
    try:
        urllib.request.urlretrieve(url, dest)
    except urllib.error.URLError as exc:
        raise SystemExit(f"download failed: {exc}") from exc
    make_executable(dest)
    return dest


def ensure_binary(*, force_download: bool) -> tuple[str, str]:
    """Return (source_label, bin_path_rel)."""
    go = shutil.which("go")
    asset = dist_binary_name()
    if go:
        name = dest_binary_name(asset or "wallop")
        out_root = ROOT / "bin"
        out_go = ROOT / "core-operator" / "bin"
        out_root.mkdir(parents=True, exist_ok=True)
        out_go.mkdir(parents=True, exist_ok=True)
        code = run([go, "build", "-o", str(out_root / name), "./cmd/harness"], cwd=ROOT / "core-operator", check=False)
        if code != 0:
            raise SystemExit(code)
        legacy = "harness.exe" if os.name == "nt" else "harness"
        run([go, "build", "-o", str(out_go / legacy), "./cmd/harness"], cwd=ROOT / "core-operator", check=False)
        return "go build", f"bin/{name}"

    if not asset:
        print(
            f"[INFO] Go not found and no prebuilt asset for "
            f"{platform.system()} {platform.machine()}"
        )
        return "none", ""

    local = ROOT / "dist" / asset
    if local.is_file() and not force_download:
        print("[INFO] Go not found, using prebuilt binary from dist/")
        dest = copy_into_bin(local, asset)
        return f"dist/{asset}", dest.relative_to(ROOT).as_posix()

    dest = download_asset(asset)
    return f"release {RELEASE_TAG}/{asset}", dest.relative_to(ROOT).as_posix()


def demo(py: Path) -> int:
    env = os.environ.copy()
    env.setdefault("MOCK_MODE", "true")
    env.setdefault("VAULT_PATH", str(ROOT / "sandbox" / "vault"))
    env.setdefault("HARNESS_TZ", "Asia/Hong_Kong")
    env.setdefault("TIME_OPS_LOG_PATH", str(ROOT / "data" / "time_ops.log"))
    return run([str(py), str(ROOT / "scripts" / "demo.py")], env=env, check=False)


def run_doctor(py: Path) -> tuple[int, dict]:
    proc = subprocess.run(
        [str(py), str(ROOT / "scripts" / "doctor.py"), "--registry", str(ROOT / "config" / "tool_registry.yaml"), "--root", str(ROOT)],
        cwd=ROOT,
        check=False,
        capture_output=True,
        text=True,
    )
    if proc.stderr:
        sys.stderr.write(proc.stderr)
    report: dict = {}
    raw = (proc.stdout or "").strip()
    if raw:
        try:
            report = json.loads(raw)
        except json.JSONDecodeError:
            print(raw)
    return proc.returncode, report


def print_status(*, doctor_code: int, report: dict, binary_src: str, binary_path: str) -> None:
    errors = report.get("errors") or []
    warns = report.get("warnings") or []
    tools = report.get("tools", "?")
    go = "present" if shutil.which("go") else "missing"
    healthy = doctor_code == 0
    print()
    print("Catalog health")
    print(f"  doctor        exit {doctor_code} ({'healthy' if healthy else 'unhealthy'})")
    print(f"  tools         {tools}")
    print(f"  errors        {len(errors)}")
    print(f"  warnings      {len(warns)}")
    print(f"  go            {go}")
    print(f"  binary        {binary_path or '(none)'} <- {binary_src}")
    print()
    if healthy:
        print("Doctor exits 0 when the catalog, tool schemas, timezone, and vault path are healthy.")
    else:
        print("Doctor did not exit 0. Fix the catalog before adding another tool.")


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Install deps, test, demo, and report catalog health")
    p.add_argument("--deps", action="store_true", help="venv + pip only")
    p.add_argument("--test", action="store_true", help="run tests only (implies --deps)")
    p.add_argument("--demo", action="store_true", help="run demo only (implies --deps)")
    p.add_argument("--build", action="store_true", help="compile wallop if go is installed")
    p.add_argument(
        "--download",
        action="store_true",
        help="fetch the nightly (or WALLOP_RELEASE) binary from GitHub Releases into bin/",
    )
    args = p.parse_args(argv)

    py = ensure_venv()
    install_deps(py)
    selected = args.test or args.demo or args.build or args.download
    if args.deps and not selected:
        print("deps ready:", py)
        return 0
    do_all = not selected and not args.deps

    binary_src, binary_path = "skipped", ""
    if do_all or args.build or args.download:
        binary_src, binary_path = ensure_binary(force_download=args.download)
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

    doctor_code, report = run_doctor(py)
    print_status(doctor_code=doctor_code, report=report, binary_src=binary_src, binary_path=binary_path)
    return doctor_code


if __name__ == "__main__":
    raise SystemExit(main())
