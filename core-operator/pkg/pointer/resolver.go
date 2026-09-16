package pointer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolvedCommand holds the concrete executable and adjusted arguments for cross-platform spawning.
type ResolvedCommand struct {
	Binary string
	Args   []string
}

// ResolveCommandTarget locates the executable binary across direct paths, workdir, and PATH.
func ResolveCommandTarget(command, workdir string) (string, error) {
	cleanCmd := filepath.Clean(strings.TrimSpace(command))

	// 1. Check relative to Workdir if specified
	if !filepath.IsAbs(cleanCmd) && workdir != "" {
		candidate := filepath.Join(workdir, cleanCmd)
		if info, err := os.Stat(candidate); err == nil && IsExecutable(info) {
			return candidate, nil
		}
	}

	// 2. Check direct absolute or local relative path
	if info, err := os.Stat(cleanCmd); err == nil && IsExecutable(info) {
		return cleanCmd, nil
	}

	// 3. Fallback: Lookup in system PATH
	pathBinary, err := exec.LookPath(cleanCmd)
	if err == nil {
		return pathBinary, nil
	}

	return "", fmt.Errorf("command '%s' not found as valid executable on this system", command)
}

// BuildSpawnCommand wraps script files (.bat, .cmd, .ps1) appropriately on Windows.
func BuildSpawnCommand(resolvedBinary string, originalArgs []string) ResolvedCommand {
	if runtime.GOOS != "windows" {
		return ResolvedCommand{
			Binary: resolvedBinary,
			Args:   originalArgs,
		}
	}

	ext := strings.ToLower(filepath.Ext(resolvedBinary))
	switch ext {
	case ".bat", ".cmd":
		// Windows batch files MUST be executed via cmd.exe /c
		cmdArgs := append([]string{"/c", resolvedBinary}, originalArgs...)
		return ResolvedCommand{
			Binary: "cmd.exe",
			Args:   cmdArgs,
		}
	case ".ps1":
		// PowerShell scripts MUST be executed via powershell.exe
		psArgs := append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", resolvedBinary}, originalArgs...)
		return ResolvedCommand{
			Binary: "powershell.exe",
			Args:   psArgs,
		}
	default:
		return ResolvedCommand{
			Binary: resolvedBinary,
			Args:   originalArgs,
		}
	}
}