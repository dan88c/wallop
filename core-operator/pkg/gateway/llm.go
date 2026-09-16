package gateway

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dan88c/wallop/core-operator/pkg/pointer"
)

func LoadSecrets(configDir string) (map[string]string, error) {
	secretPath := filepath.Join(configDir, "secrets.env")
	info, err := os.Stat(secretPath)
	if err != nil {
		return nil, fmt.Errorf("secrets file missing: %w", err)
	}

	if !pointer.IsStrictSecretPerm(info) {
		return nil, fmt.Errorf("insecure permissions on %s: must be 0600", secretPath)
	}

	file, err := os.Open(secretPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	secrets := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			secrets[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return secrets, scanner.Err()
}

func SynthesizeTool(apiURL, token, reason, sanitizedContext string) (string, error) {
	payload := map[string]any{
		"model": "claude-3-5-sonnet-20241022",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are Wallop Tool Synthesizer. Output strictly executable standalone Python code. No markdown fences.",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("Reason: %s\nContext:\n%s", reason, sanitizedContext),
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloud synthesis network failure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cloud synthesis API error: status %d", resp.StatusCode)
	}

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || len(res.Choices) == 0 {
		return "", fmt.Errorf("failed to decode cloud response")
	}

	code := strings.TrimSpace(res.Choices[0].Message.Content)
	if code == "" {
		return "", fmt.Errorf("cloud synthesized an empty tool script")
	}

	return code, nil
}

// WriteSynthesizedToolSafely writes code atomically to prevent corrupt scripts on disk.
func WriteSynthesizedToolSafely(shareDir, scriptContent string) (string, error) {
	genDir := filepath.Join(shareDir, "generated")
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return "", err
	}

	finalPath := filepath.Join(genDir, fmt.Sprintf("tool_%d.py", time.Now().Unix()))
	tmpPath := finalPath + ".tmp"

	// 1. Write to temporary file first
	if err := os.WriteFile(tmpPath, []byte(scriptContent), 0755); err != nil {
		return "", fmt.Errorf("failed to write temporary tool file: %w", err)
	}

	// 2. Atomic swap to final executable target
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath) // Cleanup on failure
		return "", fmt.Errorf("failed to commit synthesized tool: %w", err)
	}

	return finalPath, nil
}
