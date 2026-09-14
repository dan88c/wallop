package registry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func ResolveEntry(root, entry string) string {
	return resolvePath(root, entry)
}

func ResolveVenv(root, toolVenv string) string {
	if strings.TrimSpace(toolVenv) != "" {
		return resolvePath(root, toolVenv)
	}
	if env := strings.TrimSpace(os.Getenv("WALLOP_VENV")); env != "" {
		return resolvePath(root, env)
	}
	return filepath.Join(root, ".venv")
}

func VenvPython(venv string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venv, "Scripts", "python.exe")
	}
	return filepath.Join(venv, "bin", "python")
}

func ResolveCommand(tool Tool) string {
	if runtime.GOOS == "windows" {
		if tool.Command.Windows != "" {
			return tool.Command.Windows
		}
	} else if tool.Command.Posix != "" {
		return tool.Command.Posix
	}
	if tool.Runtime == "shell" || tool.Runtime == "exec" {
		return strings.TrimSpace(tool.Entry)
	}
	return ""
}

func resolvePath(root, p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~/"))
		}
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(root, filepath.FromSlash(p))
}

func RuntimeOf(tool Tool) string {
	r := strings.ToLower(strings.TrimSpace(tool.Runtime))
	if r == "" {
		return "python"
	}
	return r
}
