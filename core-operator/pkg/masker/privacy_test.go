package masker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/dan88c/wallop/core-operator/pkg/config"
)

func TestMaskContext(t *testing.T) {
	// Mock local OpenAI-compatible inference engine
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}

		resp := ChatCompletionResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						// Simulated model response masking informal relations and locations
						Content: "Please send <MASKED_PERSON> to <MASKED_LOCATION> at 10.0.0.1 with sk-1234567890abcdef1234",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		LocalEndpoint: mockServer.URL,
		MaskerModel:   "test-model",
		Confirmed:     true,
	}

	homeDir, _ := os.UserHomeDir()
	rawInput := "Please send my boss to Central Hospital at 10.0.0.1 with sk-1234567890abcdef1234 stored in " + homeDir

	result, err := MaskContext(cfg, rawInput)
	if err != nil {
		t.Fatalf("MaskContext failed: %v", err)
	}

	// 1. Verify semantic replacement
	if strings.Contains(result, "Central Hospital") || strings.Contains(result, "boss") {
		t.Errorf("failed to mask semantic entities: %s", result)
	}

	// 2. Verify IP deterministic masking
	if strings.Contains(result, "10.0.0.1") {
		t.Errorf("IP address was not masked: %s", result)
	}
	if !strings.Contains(result, "<MASKED_IP>") {
		t.Errorf("expected <MASKED_IP> in output: %s", result)
	}

	// 3. Verify API token masking
	if strings.Contains(result, "sk-1234567890abcdef1234") {
		t.Errorf("secret token was not masked: %s", result)
	}
	if !strings.Contains(result, "<MASKED_TOKEN>") {
		t.Errorf("expected <MASKED_TOKEN> in output: %s", result)
	}

	// 4. Verify home path masking
	if homeDir != "" && strings.Contains(result, homeDir) {
		t.Errorf("user home directory was not masked: %s", result)
	}
}

func TestMaskContextUnconfirmed(t *testing.T) {
	cfg := &config.Config{
		Confirmed: false,
	}

	_, err := MaskContext(cfg, "sample input")
	if err == nil {
		t.Errorf("expected error when model confirmation is false, got nil")
	}
}

func TestMaskContextEmptyOutput(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "   "}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		LocalEndpoint: mockServer.URL,
		MaskerModel:   "test-model",
		Confirmed:     true,
	}
	if _, err := MaskContext(cfg, "secret context"); err == nil {
		t.Fatal("expected empty anonymizer output to fail closed")
	}
}
