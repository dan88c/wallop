package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSelectBestModel(t *testing.T) {
	candidates := []string{
		"llama3:latest",
		"qwen2.5:7b",
		"qwen2.5:14b-instruct",
		"phi3:mini",
	}

	selected := SelectBestModel(candidates)
	if selected != "qwen2.5:14b-instruct" {
		t.Errorf("expected 'qwen2.5:14b-instruct', got '%s'", selected)
	}

	fallbackCandidates := []string{"unknown-model:v1"}
	fallbackSelected := SelectBestModel(fallbackCandidates)
	if fallbackSelected != "unknown-model:v1" {
		t.Errorf("expected 'unknown-model:v1', got '%s'", fallbackSelected)
	}
}

func TestProbeLocalModels(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}

		resp := OpenAIModelsResponse{
			Data: []struct {
				ID string `json:"id"`
			}{
				{ID: "qwen2.5:14b"},
				{ID: "llama3.1:8b"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	models, err := ProbeLocalModels(mockServer.URL)
	if err != nil {
		t.Fatalf("ProbeLocalModels failed: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != "qwen2.5:14b" || models[1] != "llama3.1:8b" {
		t.Errorf("unexpected models returned: %v", models)
	}
}