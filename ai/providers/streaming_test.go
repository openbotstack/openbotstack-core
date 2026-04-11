package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/control/skills"
)

func TestStreamingMultipleChunks(t *testing.T) {
	sseData := "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"!\"},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	ch, err := openAICompatibleStream(
		context.Background(), client, server.URL, "key", "model", nil,
		skills.GenerateRequest{Messages: []skills.Message{{Role: "user", Content: "hi"}}},
		0,
	)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var chunks []skills.StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
		if chunk.Error != nil {
			t.Fatalf("Unexpected stream error: %v", chunk.Error)
		}
	}

	if len(chunks) != 3 {
		t.Fatalf("Expected 3 chunks, got %d", len(chunks))
	}
	if chunks[0].Content != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", chunks[0].Content)
	}
	if chunks[1].Content != " world" {
		t.Errorf("Expected ' world', got '%s'", chunks[1].Content)
	}
	if chunks[2].Content != "!" {
		t.Errorf("Expected '!', got '%s'", chunks[2].Content)
	}
	if chunks[2].FinishReason != "stop" {
		t.Errorf("Expected finish_reason 'stop', got '%s'", chunks[2].FinishReason)
	}
}

func TestStreamingSingleChunk(t *testing.T) {
	sseData := "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	ch, _ := openAICompatibleStream(
		context.Background(), client, server.URL, "key", "model", nil,
		skills.GenerateRequest{Messages: []skills.Message{{Role: "user", Content: "hi"}}},
		0,
	)

	var chunks []skills.StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}
	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Content != "Hi" {
		t.Errorf("Expected 'Hi', got '%s'", chunks[0].Content)
	}
}

func TestStreamingMalformedJSON(t *testing.T) {
	sseData := "data: {bad json}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	ch, _ := openAICompatibleStream(
		context.Background(), client, server.URL, "key", "model", nil,
		skills.GenerateRequest{Messages: []skills.Message{{Role: "user", Content: "hi"}}},
		0,
	)

	var chunks []skills.StreamChunk
	for chunk := range ch {
		if chunk.Error == nil {
			chunks = append(chunks, chunk)
		}
	}
	if len(chunks) != 1 {
		t.Fatalf("Expected 1 valid chunk, got %d", len(chunks))
	}
	if chunks[0].Content != "ok" {
		t.Errorf("Expected 'ok', got '%s'", chunks[0].Content)
	}
}

func TestStreamingContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Continuously write chunks until the context is cancelled.
		// This ensures the goroutine will always hit the context check
		// in the select statement while trying to send on the channel.
		for {
			select {
			case <-r.Context().Done():
				return
			default:
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"x\"}}]}\n\n")
				w.(http.Flusher).Flush()
			}
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	ch, _ := openAICompatibleStream(
		ctx, client, server.URL, "key", "model", nil,
		skills.GenerateRequest{Messages: []skills.Message{{Role: "user", Content: "hi"}}},
		0,
	)

	// Read at least one chunk to confirm streaming is working
	<-ch
	// Cancel the context — the goroutine should detect this and send an error chunk
	cancel()

	var gotError bool
	for chunk := range ch {
		if chunk.Error != nil {
			gotError = true
		}
	}
	if !gotError {
		t.Error("Expected error chunk on context cancellation")
	}
}

func TestStreamingEmptyLines(t *testing.T) {
	sseData := "\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"hi\"},\"finish_reason\":\"stop\"}]}\n\n\n\ndata: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseData)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	ch, _ := openAICompatibleStream(
		context.Background(), client, server.URL, "key", "model", nil,
		skills.GenerateRequest{Messages: []skills.Message{{Role: "user", Content: "hi"}}},
		0,
	)

	var chunks []skills.StreamChunk
	for chunk := range ch {
		if chunk.Error == nil {
			chunks = append(chunks, chunk)
		}
	}
	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}
}
