package pointer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type ToolPointer struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Command     string          `json:"command"`
	Args        []string        `json:"args,omitempty"`
	Workdir     string          `json:"workdir,omitempty"`
	Healthcheck []string        `json:"healthcheck,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
}

func LoadAndValidatePointers(configDir string) ([]ToolPointer, error) {
	toolsDir := filepath.Join(configDir, "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tools dir: %w", err)
	}

	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools dir: %w", err)
	}

	var validTools []ToolPointer
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(toolsDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var tool ToolPointer
		if err := json.Unmarshal(data, &tool); err != nil {
			continue
		}
		if ValidateToolName(tool.Name) != nil {
			continue
		}

		if _, err := ResolveCommandTarget(tool.Command, tool.Workdir); err != nil {
			continue
		}

		validTools = append(validTools, tool)
	}

	return validTools, nil
}

func RunHealthcheck(tool ToolPointer) error {
	if len(tool.Healthcheck) == 0 {
		return nil
	}
	resolved, err := ResolveCommandTarget(tool.Command, tool.Workdir)
	if err != nil {
		return err
	}
	spawn := BuildSpawnCommand(resolved, tool.Healthcheck)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, spawn.Binary, spawn.Args...)
	if tool.Workdir != "" {
		cmd.Dir = tool.Workdir
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func ValidatePayload(schemaRaw json.RawMessage, payloadRaw []byte) error {
	if len(schemaRaw) == 0 {
		return nil
	}

	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(schemaRaw, &schema); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return fmt.Errorf("malformed payload JSON: %w", err)
	}

	for _, reqField := range schema.Required {
		if _, exists := payload[reqField]; !exists {
			return fmt.Errorf("missing required parameter: %s", reqField)
		}
	}
	return nil
}
