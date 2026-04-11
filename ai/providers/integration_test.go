//go:build integration

package providers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/control/skills"
)

func getTestURL() string {
	return os.Getenv("OBS_TEST_LLM_URL")
}

func getTestAPIKey() string {
	return os.Getenv("OBS_TEST_LLM_API_KEY")
}

func TestIntegration_OpenAICompatible_Sync(t *testing.T) {
	baseURL := getTestURL()
	if baseURL == "" {
		t.Skip("OBS_TEST_LLM_URL not set, skipping integration test")
	}

	cfg := ProviderConfig{
		BaseURL: baseURL,
		APIKey:  getTestAPIKey(),
		Model:   os.Getenv("OBS_TEST_LLM_MODEL"),
	}
	if cfg.Model == "" {
		cfg.Model = "default"
	}

	provider, err := NewProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewProviderFromConfig failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := provider.Generate(ctx, skills.GenerateRequest{
		Messages:    []skills.Message{{Role: "user", Content: "Say hello in one word."}},
		MaxTokens:   10,
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.Content == "" {
		t.Error("Expected non-empty content")
	}
	t.Logf("Response: %s (tokens: %d)", resp.Content, resp.Usage.TotalTokens)
}

func TestIntegration_OpenAICompatible_Stream(t *testing.T) {
	baseURL := getTestURL()
	if baseURL == "" {
		t.Skip("OBS_TEST_LLM_URL not set, skipping integration test")
	}

	cfg := ProviderConfig{
		BaseURL: baseURL,
		APIKey:  getTestAPIKey(),
		Model:   os.Getenv("OBS_TEST_LLM_MODEL"),
	}
	if cfg.Model == "" {
		cfg.Model = "default"
	}

	provider, err := NewProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewProviderFromConfig failed: %v", err)
	}

	sp, ok := provider.(StreamingModelProvider)
	if !ok {
		t.Fatal("Provider does not implement StreamingModelProvider")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ch, err := sp.GenerateStream(ctx, skills.GenerateRequest{
		Messages:    []skills.Message{{Role: "user", Content: "Count from 1 to 5."}},
		MaxTokens:   50,
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}

	var chunks []skills.StreamChunk
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("Stream error: %v", chunk.Error)
		}
		chunks = append(chunks, chunk)
	}
	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
	t.Logf("Received %d chunks", len(chunks))
}
