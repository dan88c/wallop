package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	LocalEndpoint string `json:"local_endpoint"`
	MaskerModel   string `json:"masker_model"`
	Confirmed     bool   `json:"confirmed"`
}

type OpenAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "wallop", "config.json")
}

func LoadConfig() (*Config, error) {
	cfgPath := GetConfigPath()
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return &Config{
			LocalEndpoint: "http://localhost:11434/v1",
			Confirmed:     false,
		}, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	cfgPath := GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}

func ProbeLocalModels(endpoint string) ([]string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	reqURL := strings.TrimRight(endpoint, "/") + "/models"
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query models at %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("endpoint returned unexpected status: %d", resp.StatusCode)
	}

	var res OpenAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode models payload: %w", err)
	}

	var models []string
	for _, item := range res.Data {
		models = append(models, item.ID)
	}
	return models, nil
}

func SelectBestModel(models []string) string {
	preferred := []string{
		"qwen2.5:32b", "qwen2.5:14b", "qwen2.5-32b", "qwen2.5-14b",
		"mistral-small", "llama-3.1-8b", "llama3.1:8b", "qwen2.5:7b",
	}
	for _, pref := range preferred {
		for _, m := range models {
			if strings.Contains(strings.ToLower(m), pref) {
				return m
			}
		}
	}
	if len(models) > 0 {
		return models[0]
	}
	return ""
}

func RunInteractiveSetup(cfg *Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("[Wallop Setup] Current local endpoint: [%s]\n", cfg.LocalEndpoint)
	fmt.Print("Enter endpoint URL (press Enter to keep current): ")
	customEndpoint, _ := reader.ReadString('\n')
	customEndpoint = strings.TrimSpace(customEndpoint)
	if customEndpoint != "" {
		cfg.LocalEndpoint = customEndpoint
	}

	fmt.Printf("[Wallop Setup] Probing models from %s...\n", cfg.LocalEndpoint)
	models, err := ProbeLocalModels(cfg.LocalEndpoint)
	if err != nil {
		return fmt.Errorf("endpoint probe failed: %w", err)
	}

	if len(models) == 0 {
		return fmt.Errorf("no models detected. Please load a model into your runtime")
	}

	recommended := SelectBestModel(models)
	fmt.Println("Available models:", strings.Join(models, ", "))
	fmt.Printf("Recommended de-identification model: [%s]\n", recommended)
	fmt.Printf("Confirm using '%s' for privacy masking? [Y/n]: ", recommended)

	confirmInput, _ := reader.ReadString('\n')
	confirmInput = strings.TrimSpace(strings.ToLower(confirmInput))

	if confirmInput == "" || confirmInput == "y" || confirmInput == "yes" {
		cfg.MaskerModel = recommended
		cfg.Confirmed = true
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to write configuration: %w", err)
		}
		fmt.Println("[Wallop Setup] Configuration saved successfully.")
		return nil
	}

	return fmt.Errorf("configuration aborted by user")
}