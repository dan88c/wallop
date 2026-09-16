package pointer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ValidateToolName(name string) error {
	if !toolNamePattern.MatchString(name) {
		return fmt.Errorf("invalid tool name %q: use only letters, digits, underscore, and hyphen", name)
	}
	return nil
}

func PointerFile(configDir, name string) (string, error) {
	if err := ValidateToolName(name); err != nil {
		return "", err
	}
	toolsDir := filepath.Join(configDir, "tools")
	dest := filepath.Join(toolsDir, name+".json")
	absTools, err := filepath.Abs(toolsDir)
	if err != nil {
		return "", err
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absTools, absDest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("tool path escapes tools directory")
	}
	return dest, nil
}

func RegisterTool(configDir string, tool ToolPointer) error {
	if tool.Name == "" || tool.Command == "" {
		return fmt.Errorf("tool name and command are required")
	}
	if err := ValidateToolName(tool.Name); err != nil {
		return err
	}

	resolved, err := ResolveCommandTarget(tool.Command, tool.Workdir)
	if err != nil {
		return err
	}
	tool.Command = resolved

	toolsDir := filepath.Join(configDir, "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return err
	}

	destPath, err := PointerFile(configDir, tool.Name)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(tool, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(destPath, data, 0644)
}
