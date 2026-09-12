from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))
from bootstrap import dist_binary_name  # noqa: E402


def test_linux_amd64_maps_to_dist_name():
    assert dist_binary_name("Linux", "x86_64") == "wallop-linux-amd64"


def test_darwin_arm64_maps_to_dist_name():
    assert dist_binary_name("Darwin", "arm64") == "wallop-darwin-arm64"


def test_windows_amd64_maps_to_dist_exe():
    assert dist_binary_name("Windows", "AMD64") == "wallop-windows-amd64.exe"
