package pointer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveCommandTarget(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Current running test executable should always resolve directly
	currentExe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to resolve current test executable: %v", err)
	}

	resolved, err := ResolveCommandTarget(currentExe, "")
	if err != nil {
		t.Fatalf("expected direct executable to resolve, got error: %v", err)
	}
	if resolved != currentExe {
		t.Errorf("expected %s, got %s", currentExe, resolved)
	}

	// 2. Relative to Workdir resolution
	exeName := filepath.Base(currentExe)
	resolvedWithWorkdir, err := ResolveCommandTarget(exeName, filepath.Dir(currentExe))
	if err != nil {
		t.Errorf("expected workdir-relative target to resolve, got error: %v", err)
	}
	if filepath.Clean(resolvedWithWorkdir) != filepath.Clean(currentExe) {
		t.Errorf("expected %s, got %s", currentExe, resolvedWithWorkdir)
	}

	// 3. System PATH resolution (check for "go" binary which is guaranteed present during test)
	goBinary, err := ResolveCommandTarget("go", "")
	if err != nil {
		t.Errorf("expected system PATH binary 'go' to resolve, got error: %v", err)
	}
	if goBinary == "" {
		t.Errorf("expected non-empty resolved path for 'go'")
	}

	// 4. Non-existent command must return error
	_, err = ResolveCommandTarget("non_existent_command_12345", tempDir)
	if err == nil {
		t.Errorf("expected error for non-existent command, got nil")
	}
}

func TestBuildSpawnCommand(t *testing.T) {
	originalArgs := []string{"run", "--flag", "value"}

	if runtime.GOOS == "windows" {
		// Test .bat wrapping on Windows
		batSpawn := BuildSpawnCommand("C:\\tools\\run.bat", originalArgs)
		if batSpawn.Binary != "cmd.exe" {
			t.Errorf("expected cmd.exe, got %s", batSpawn.Binary)
		}
		if len(batSpawn.Args) < 2 || batSpawn.Args[0] != "/c" || batSpawn.Args[1] != "C:\\tools\\run.bat" {
			t.Errorf("unexpected bat spawn args: %v", batSpawn.Args)
		}

		// Test .ps1 wrapping on Windows
		psSpawn := BuildSpawnCommand("C:\\tools\\run.ps1", originalArgs)
		if psSpawn.Binary != "powershell.exe" {
			t.Errorf("expected powershell.exe, got %s", psSpawn.Binary)
		}
		if !strings.Contains(strings.Join(psSpawn.Args, " "), "-ExecutionPolicy Bypass") {
			t.Errorf("expected execution policy bypass in ps1 args: %v", psSpawn.Args)
		}
	} else {
		// On non-Windows platforms, spawn command should pass through unaltered
		nativeSpawn := BuildSpawnCommand("/usr/local/bin/mytool", originalArgs)
		if nativeSpawn.Binary != "/usr/local/bin/mytool" {
			t.Errorf("expected binary passthrough on POSIX, got %s", nativeSpawn.Binary)
		}
		if len(nativeSpawn.Args) != len(originalArgs) {
			t.Errorf("args length mismatch: got %v, expected %v", nativeSpawn.Args, originalArgs)
		}
	}
}