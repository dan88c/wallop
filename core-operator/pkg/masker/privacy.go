package masker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/dan88c/wallop/core-operator/pkg/config"
)

var (
	ipv4Regex  = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	tokenRegex = regexp.MustCompile(`(?i)(bearer\s+[a-z0-9_\-\.]{10,}|ghp_[a-z0-9]{20,}|sk-[a-z0-9]{20,})`)
)

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float32       `json:"temperature"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func MaskContext(cfg *config.Config, rawContext string) (string, error) {
	if !cfg.Confirmed || cfg.MaskerModel == "" {
		return "", fmt.Errorf("local model not confirmed; run 'wallop init' first")
	}

	// 1. Semantic Anonymization via Standard OpenAI Chat Endpoint
	systemPrompt := "You are a data privacy anonymizer. Rewrite the text replacing real personal names, specific titles/boss/colleague relationships, company names, and specific medical facilities with generic placeholders (<MASKED_PERSON>, <MASKED_LOCATION>). Retain all technical logic, programming code, and error messages unaltered. Output strictly the rewritten text without markdown code blocks."

	reqPayload := ChatCompletionRequest{
		Model: cfg.MaskerModel,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: rawContext},
		},
		Temperature: 0.0,
	}

	payloadBytes, _ := json.Marshal(reqPayload)
	reqURL := strings.TrimRight(cfg.LocalEndpoint, "/") + "/chat/completions"

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(reqURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("local anonymization request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("local runtime returned non-200 status: %d", resp.StatusCode)
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil || len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("failed to decode local chat completion output")
	}

	sanitized := chatResp.Choices[0].Message.Content
	if strings.TrimSpace(sanitized) == "" {
		return "", fmt.Errorf("local anonymizer returned empty output")
	}

	// 2. Cross-Platform Deterministic Path & User Masking
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		sanitized = strings.ReplaceAll(sanitized, home, "<MASKED_HOME>")
	}

	username := os.Getenv("USER")
	if runtime.GOOS == "windows" {
		username = os.Getenv("USERNAME")
	}
	if username != "" {
		sanitized = regexp.MustCompile(`\b`+regexp.QuoteMeta(username)+`\b`).ReplaceAllString(sanitized, "<MASKED_USER>")
	}

	sanitized = tokenRegex.ReplaceAllString(sanitized, "<MASKED_TOKEN>")
	sanitized = ipv4Regex.ReplaceAllString(sanitized, "<MASKED_IP>")

	return sanitized, nil
}
