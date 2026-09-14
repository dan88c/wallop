#!/usr/bin/env python3
"""Catalog health check.

Default (operator): names unique; entry file exists; optional venv path exists.
--strict: also import tool, require pydantic *Input, match YAML params.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from collections import Counter
from pathlib import Path
from typing import Any

import yaml


def repo_root() -> Path:
    override = os.environ.get("WALLOP_ROOT")
    if override:
        return Path(override).resolve()
    return Path(__file__).resolve().parents[1]


def load_registry(path: Path) -> dict[str, Any]:
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    if not isinstance(data, dict):
        raise ValueError(f"registry is not a mapping: {path}")
    return data


def resolve_entry(root: Path, entry: str) -> Path:
    p = Path(entry)
    if p.is_absolute():
        return p.resolve()
    return (root / p).resolve()


def check_duplicates(tools: list[dict[str, Any]]) -> list[str]:
    names = [str(t.get("name") or "") for t in tools]
    counts = Counter(names)
    return [f"duplicate tool name {name!r} ({n} times)" for name, n in counts.items() if name and n > 1]


def check_tool_exists(root: Path, tool: dict[str, Any]) -> tuple[list[str], list[str]]:
    errors: list[str] = []
    warns: list[str] = []
    name = str(tool.get("name") or "")
    raw = str(tool.get("entry") or f"tools/{name}.py")
    entry = resolve_entry(root, raw)
    if not entry.is_file():
        errors.append(f"{name}: missing entry {raw}")
    venv = tool.get("venv") or os.environ.get("WALLOP_VENV")
    if venv:
        vp = Path(str(venv))
        if not vp.is_absolute():
            vp = (root / vp).resolve()
        if not vp.exists():
            errors.append(f"{name}: venv path missing {vp}")
    return errors, warns


def check_env(root: Path, strict: bool) -> tuple[list[str], list[str]]:
    errors: list[str] = []
    warns: list[str] = []
    tz = os.environ.get("HARNESS_TZ") or "Asia/Hong_Kong"
    if not os.environ.get("HARNESS_TZ"):
        warns.append("HARNESS_TZ unset; using default Asia/Hong_Kong")
    try:
        from zoneinfo import ZoneInfo

        ZoneInfo(tz)
    except Exception as exc:  # noqa: BLE001
        msg = f"HARNESS_TZ={tz!r} unusable ({exc}). On Windows: py -3 -m pip install tzdata"
        if strict:
            errors.append(msg)
        else:
            warns.append(msg)

    raw_vault = os.environ.get("VAULT_PATH") or str(root / "sandbox" / "vault")
    vault = Path(raw_vault)
    if not vault.is_absolute():
        vault = (root / vault).resolve()
    if not os.environ.get("VAULT_PATH"):
        warns.append(f"VAULT_PATH unset; using {vault}")
    if not vault.exists():
        errors.append(f"VAULT_PATH does not exist: {vault}")
    elif not vault.is_dir():
        errors.append(f"VAULT_PATH is not a directory: {vault}")
    return errors, warns


def run(registry_path: Path, root: Path, strict: bool) -> int:
    errors: list[str] = []
    warns: list[str] = []
    if not registry_path.is_file():
        print(f"ERROR registry missing: {registry_path}", file=sys.stderr)
        return 3
    data = load_registry(registry_path)
    tools = list(data.get("tools") or [])
    errors.extend(check_duplicates(tools))
    for tool in tools:
        e, w = check_tool_exists(root, tool)
        errors.extend(e)
        warns.extend(w)
    e, w = check_env(root, strict)
    errors.extend(e)
    warns.extend(w)

    report = {
        "ok": not errors,
        "mode": "strict" if strict else "exists",
        "registry": str(registry_path),
        "tools": len(tools),
        "errors": errors,
        "warnings": warns,
    }
    print(json.dumps(report, indent=2))
    for item in warns:
        print(f"WARN  {item}", file=sys.stderr)
    for item in errors:
        print(f"ERROR {item}", file=sys.stderr)
    if errors:
        print(f"doctor: {len(errors)} error(s), {len(warns)} warning(s)", file=sys.stderr)
        return 1
    print(f"doctor: ok ({len(tools)} tools, {len(warns)} warning(s))", file=sys.stderr)
    return 0


def main(argv: list[str] | None = None) -> int:
    root = repo_root()
    p = argparse.ArgumentParser(description="Check wallop registry paths")
    p.add_argument("--registry", default=str(root / "config" / "tool_registry.yaml"))
    p.add_argument("--root", default=str(root))
    p.add_argument("--strict", action="store_true", help="import tools and match pydantic Input")
    args = p.parse_args(argv)
    return run(Path(args.registry), Path(args.root).resolve(), args.strict)


if __name__ == "__main__":
    raise SystemExit(main())
