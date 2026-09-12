#!/usr/bin/env python3
"""Catalog health check.

1. Duplicate tool names in config/tool_registry.yaml
2. Each tools/<name>.py imports and its Input BaseModel schema matches YAML params
3. HARNESS_TZ is a real IANA zone and VAULT_PATH exists
"""

from __future__ import annotations

import argparse
import importlib.util
import json
import os
import sys
import typing
from collections import Counter
from pathlib import Path
from typing import Any, Literal

import yaml
from pydantic import BaseModel

YAML_TO_JSON = {
    "string": "string",
    "path": "string",
    "datetime": "string",
    "date": "string",
    "time": "string",
    "timestamp": "string",
    "enum": "string",
    "int": "integer",
    "integer": "integer",
    "bool": "boolean",
    "boolean": "boolean",
    "number": "number",
    "float": "number",
}


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


def import_tool(entry: Path, name: str) -> Any:
    spec = importlib.util.spec_from_file_location(f"wallop_tool_{name}", entry)
    if spec is None or spec.loader is None:
        raise ImportError(f"cannot load {entry}")
    mod = importlib.util.module_from_spec(spec)
    # Future annotations leave Literal unresolved under spec_from_file_location.
    mod.__dict__.setdefault("Literal", Literal)
    mod.__dict__.setdefault("Any", Any)
    mod.__dict__.setdefault("typing", typing)
    spec.loader.exec_module(mod)
    return mod


def input_model(mod: Any) -> type[BaseModel] | None:
    found: list[type[BaseModel]] = []
    for attr in vars(mod).values():
        if isinstance(attr, type) and issubclass(attr, BaseModel) and attr is not BaseModel:
            if attr.__name__.endswith("Input"):
                found.append(attr)
    if found:
        return found[0]
    return None


def json_type(schema_prop: dict[str, Any]) -> str:
    if "anyOf" in schema_prop:
        types = [p.get("type") for p in schema_prop["anyOf"] if p.get("type") and p.get("type") != "null"]
        return types[0] if types else "string"
    if "enum" in schema_prop:
        return "string"
    return str(schema_prop.get("type") or "string")


def schema_for(model: type[BaseModel]) -> dict[str, Any]:
    try:
        model.model_rebuild(_types_namespace={"Literal": Literal, "Any": Any, "typing": typing})
    except Exception:
        pass
    return model.model_json_schema()


def check_duplicates(tools: list[dict[str, Any]]) -> list[str]:
    names = [str(t.get("name") or "") for t in tools]
    counts = Counter(names)
    return [f"duplicate tool name {name!r} ({n} times)" for name, n in counts.items() if name and n > 1]


def check_tool(root: Path, tool: dict[str, Any]) -> tuple[list[str], list[str]]:
    errors: list[str] = []
    warns: list[str] = []
    name = str(tool.get("name") or "")
    entry_rel = str(tool.get("entry") or f"tools/{name}.py")
    entry = (root / entry_rel).resolve()
    if not entry.is_file():
        return [f"{name}: missing entry {entry_rel}"], warns
    try:
        mod = import_tool(entry, name or entry.stem)
    except Exception as exc:  # noqa: BLE001 — doctor must surface any import failure
        return [f"{name}: import failed: {exc}"], warns
    model = input_model(mod)
    if model is None:
        return [f"{name}: no pydantic *Input BaseModel in {entry_rel}"], warns
    try:
        schema = schema_for(model)
    except Exception as exc:  # noqa: BLE001
        return [f"{name}: model_json_schema() failed: {exc}"], warns
    props: dict[str, Any] = schema.get("properties") or {}
    required = set(schema.get("required") or [])
    yaml_params = tool.get("params") or []
    yaml_names = [str(p.get("name")) for p in yaml_params if p.get("name")]
    yaml_set = set(yaml_names)
    if len(yaml_names) != len(yaml_set):
        errors.append(f"{name}: duplicate param names in YAML: {yaml_names}")

    for p in yaml_params:
        pname = str(p.get("name") or "")
        if not pname:
            errors.append(f"{name}: YAML param missing name")
            continue
        if pname not in props:
            errors.append(f"{name}: YAML param {pname!r} not in {model.__name__}.model_json_schema()")
            continue
        ytype = YAML_TO_JSON.get(str(p.get("type") or "string").lower(), "string")
        stype = json_type(props[pname])
        if ytype != stype:
            errors.append(
                f"{name}: param {pname!r} type YAML {p.get('type')} ~ {ytype} vs schema {stype}"
            )
        yreq = bool(p.get("required"))
        if yreq and pname not in required:
            errors.append(f"{name}: YAML marks {pname!r} required but schema does not")
        if not yreq and pname in required:
            errors.append(f"{name}: schema requires {pname!r} but YAML does not")

    extras = sorted(set(props) - yaml_set)
    if extras:
        warns.append(f"{name}: schema fields not in YAML: {', '.join(extras)}")
    return errors, warns


def check_env(root: Path) -> tuple[list[str], list[str]]:
    errors: list[str] = []
    warns: list[str] = []
    tz = os.environ.get("HARNESS_TZ") or "Asia/Hong_Kong"
    if not os.environ.get("HARNESS_TZ"):
        warns.append("HARNESS_TZ unset; using default Asia/Hong_Kong")
    try:
        from zoneinfo import ZoneInfo

        ZoneInfo(tz)
    except Exception as exc:  # noqa: BLE001
        errors.append(f"HARNESS_TZ={tz!r} is not a usable IANA zone: {exc}")

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


def run(registry_path: Path, root: Path) -> int:
    errors: list[str] = []
    warns: list[str] = []
    if not registry_path.is_file():
        print(f"ERROR registry missing: {registry_path}", file=sys.stderr)
        return 3
    data = load_registry(registry_path)
    tools = list(data.get("tools") or [])
    errors.extend(check_duplicates(tools))
    for tool in tools:
        e, w = check_tool(root, tool)
        errors.extend(e)
        warns.extend(w)
    e, w = check_env(root)
    errors.extend(e)
    warns.extend(w)

    report = {
        "ok": not errors,
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
    p = argparse.ArgumentParser(description="Check wallop registry, tool schemas, and env paths")
    p.add_argument(
        "--registry",
        default=str(root / "config" / "tool_registry.yaml"),
        help="path to tool_registry.yaml",
    )
    p.add_argument("--root", default=str(root), help="repo root")
    args = p.parse_args(argv)
    return run(Path(args.registry), Path(args.root).resolve())


if __name__ == "__main__":
    raise SystemExit(main())
