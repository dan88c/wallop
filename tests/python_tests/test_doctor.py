from pathlib import Path
import os
import sys

import yaml

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))
from doctor import check_duplicates, check_env, check_tool, run  # noqa: E402


def test_no_duplicate_names_in_repo_registry():
    data = yaml.safe_load((ROOT / "config" / "tool_registry.yaml").read_text(encoding="utf-8"))
    assert check_duplicates(data["tools"]) == []


def test_repo_tools_import_and_align():
    data = yaml.safe_load((ROOT / "config" / "tool_registry.yaml").read_text(encoding="utf-8"))
    errors: list[str] = []
    for tool in data["tools"]:
        e, _w = check_tool(ROOT, tool)
        errors.extend(e)
    assert errors == []


def test_duplicate_names_detected():
    assert check_duplicates([{"name": "a"}, {"name": "a"}]) == [
        "duplicate tool name 'a' (2 times)"
    ]


def test_env_defaults_ok(monkeypatch):
    monkeypatch.delenv("HARNESS_TZ", raising=False)
    monkeypatch.delenv("VAULT_PATH", raising=False)
    errors, warns = check_env(ROOT)
    assert errors == []
    assert any("HARNESS_TZ unset" in w for w in warns)
    assert any("VAULT_PATH unset" in w for w in warns)


def test_bad_timezone_is_error(monkeypatch):
    monkeypatch.setenv("HARNESS_TZ", "Not/A_Zone")
    monkeypatch.setenv("VAULT_PATH", str(ROOT / "sandbox" / "vault"))
    errors, _ = check_env(ROOT)
    assert errors and "HARNESS_TZ" in errors[0]


def test_missing_vault_is_error(monkeypatch, tmp_path):
    monkeypatch.setenv("HARNESS_TZ", "Asia/Hong_Kong")
    monkeypatch.setenv("VAULT_PATH", str(tmp_path / "no-such-vault"))
    errors, _ = check_env(ROOT)
    assert errors and "VAULT_PATH" in errors[0]


def test_run_repo_catalog_exits_zero(monkeypatch):
    monkeypatch.setenv("HARNESS_TZ", "Asia/Hong_Kong")
    monkeypatch.setenv("VAULT_PATH", str(ROOT / "sandbox" / "vault"))
    code = run(ROOT / "config" / "tool_registry.yaml", ROOT)
    assert code == 0
