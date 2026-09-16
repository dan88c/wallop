from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))
from doctor import check_duplicates, check_env, run  # noqa: E402


def test_duplicate_names_detected():
    assert check_duplicates([{"name": "a"}, {"name": "a"}]) == [
        "duplicate tool name 'a' (2 times)"
    ]


def test_unique_names_ok():
    assert check_duplicates([{"name": "a"}, {"name": "b"}]) == []


def test_env_defaults_ok(monkeypatch):
    monkeypatch.delenv("HARNESS_TZ", raising=False)
    monkeypatch.delenv("VAULT_PATH", raising=False)
    errors, warns = check_env(ROOT, False)
    assert errors == []
    assert any("HARNESS_TZ unset" in w for w in warns)
    assert any("VAULT_PATH unset" in w for w in warns)


def test_bad_timezone_is_warning_unless_strict(monkeypatch):
    monkeypatch.setenv("HARNESS_TZ", "Not/A_Zone")
    monkeypatch.setenv("VAULT_PATH", str(ROOT / "sandbox" / "vault"))
    errors, warns = check_env(ROOT, False)
    assert errors == []
    assert any("HARNESS_TZ" in w for w in warns)
    errors, _ = check_env(ROOT, True)
    assert errors and "HARNESS_TZ" in errors[0]


def test_missing_vault_is_error(monkeypatch, tmp_path):
    monkeypatch.setenv("HARNESS_TZ", "Asia/Hong_Kong")
    monkeypatch.setenv("VAULT_PATH", str(tmp_path / "no-such-vault"))
    errors, _ = check_env(ROOT, False)
    assert errors and "VAULT_PATH" in errors[0]


def test_run_missing_registry_exits_3(tmp_path):
    code = run(tmp_path / "missing.yaml", ROOT, False)
    assert code == 3
