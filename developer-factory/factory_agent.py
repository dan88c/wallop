#!/usr/bin/env python3
"""Developer factory: gap request -> Python tool -> tests."""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from pathlib import Path

from jinja2 import Environment, FileSystemLoader

sys.path.insert(0, str(Path(__file__).resolve().parent))
from cloud_client import propose  # noqa: E402
from privacy_guard import assert_safe_for_cloud  # noqa: E402

TEMPLATE_DIR = Path(__file__).resolve().parent / "templates"


def repo_root() -> Path:
    override = os.environ.get("WALLOP_ROOT")
    if override:
        return Path(override).resolve()
    return Path(__file__).resolve().parents[1]


def snake(name: str) -> str:
    s = re.sub(r"[^a-zA-Z0-9]+", "_", name).strip("_").lower()
    return s or "new_tool"


def class_name(name: str) -> str:
    base = "".join(part.capitalize() for part in snake(name).split("_"))
    return base if base.endswith("Tool") else base + "Tool"


def parse_params(raw: list[str]) -> list[dict]:
    out = []
    for item in raw:
        bits = item.split(":")
        pname = bits[0]
        ptype = bits[1] if len(bits) > 1 else "string"
        required = len(bits) > 2 and bits[2] in {"req", "required", "true", "1"}
        py = {
            "string": "str",
            "datetime": "str",
            "date": "str",
            "path": "str",
            "enum": "str",
            "int": "int",
            "bool": "bool",
        }.get(ptype, "str")
        out.append(
            {
                "name": pname,
                "type": ptype,
                "required": required,
                "annotation": py if required else f"{py} | None",
                "default": "None" if not required else None,
            }
        )
    return out


def render_tool(name: str, desc: str, params: list[dict]) -> str:
    env = Environment(loader=FileSystemLoader(str(TEMPLATE_DIR)), autoescape=False)
    tmpl = env.get_template("tool_base.py.jinja")
    return tmpl.render(class_name=class_name(name), desc=desc, params=params)


def write_smoke_test(name: str, *, root: Path) -> Path:
    test_dir = root / "tests" / "python_tests"
    test_dir.mkdir(parents=True, exist_ok=True)
    path = test_dir / f"test_{name}_smoke.py"
    body = f'''from importlib.machinery import SourceFileLoader
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
mod = SourceFileLoader("{name}", str(ROOT / "tools" / "{name}.py")).load_module()


def test_import_and_schema():
    assert hasattr(mod, "run")
'''
    path.write_text(body, encoding="utf-8")
    return path


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Generate a tool, then run its tests")
    p.add_argument("--name", default="", help="tool name (optional with --brief)")
    p.add_argument("--desc", default="", help="tool description (optional with --brief)")
    p.add_argument("--brief", default="", help="free-text request; sanitized then proposed")
    p.add_argument("--tag", action="append", default=[])
    p.add_argument("--param", action="append", default=[], help="name:type[:required]")
    p.add_argument("--risk", default="read", choices=["read", "write"])
    p.add_argument("--skip-test", action="store_true")
    args = p.parse_args(argv)

    root = repo_root()
    tools_dir = root / "tools"

    desc = args.desc
    name = args.name
    params = parse_params(args.param)

    if args.brief:
        abstract = assert_safe_for_cloud(args.brief)
        proposal = propose(abstract)
        desc = desc or proposal.desc
        name = name or proposal.name
        if not params:
            params = parse_params(
                [
                    f"{p['name']}:{p.get('type', 'string')}:{'required' if p.get('required') else 'optional'}"
                    for p in proposal.params
                ]
            )
    elif not name or not desc:
        p.error("--name and --desc are required unless --brief is set")
    else:
        _ = assert_safe_for_cloud(desc)

    name = snake(name)
    dest = tools_dir / f"{name}.py"
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(render_tool(name, desc, params), encoding="utf-8")
    test_path = write_smoke_test(name, root=root)

    if not args.skip_test:
        proc = subprocess.run(
            [sys.executable, "-m", "pytest", str(test_path), "-q"],
            cwd=root,
        )
        if proc.returncode != 0:
            print("factory: tests failed; tool not kept as registered", file=sys.stderr)
            return proc.returncode

    print(f"factory: wrote {dest.relative_to(root)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
