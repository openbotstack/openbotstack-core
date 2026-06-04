package providers

import (
	"encoding/json"
	"testing"

	"github.com/openbotstack/openbotstack-core/ai/types"
)

func TestContentsToOpenAI_SingleText(t *testing.T) {
	contents := []types.ContentBlock{types.NewTextBlock("hello")}
	result := contentsToOpenAI(contents)

	// Single text block should serialize as plain string
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"hello"` {
		t.Errorf("single text = %s, want %q", b, "hello")
	}
}

func TestContentsToOpenAI_Multimodal(t *testing.T) {
	contents := []types.ContentBlock{
		types.NewTextBlock("What is in this image?"),
		types.NewImageBlock("https://example.com/cat.jpg"),
	}
	result := contentsToOpenAI(contents)

	b, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	var blocks []map[string]any
	if err := json.Unmarshal(b, &blocks); err != nil {
		t.Fatalf("should be array, got %s: %v", b, err)
	}
	if len(blocks) != 2 {
		t.Fatalf("len(blocks) = %d, want 2", len(blocks))
	}
	if blocks[0]["type"] != "text" {
		t.Errorf("blocks[0].type = %v, want text", blocks[0]["type"])
	}
	if blocks[1]["type"] != "image_url" {
		t.Errorf("blocks[1].type = %v, want image_url", blocks[1]["type"])
	}
	imgURL, ok := blocks[1]["image_url"].(map[string]any)
	if !ok {
		t.Fatalf("blocks[1].image_url = %T, want map", blocks[1]["image_url"])
	}
	if imgURL["url"] != "https://example.com/cat.jpg" {
		t.Errorf("image_url.url = %v, want https://example.com/cat.jpg", imgURL["url"])
	}
}

func TestContentsToOpenAI_ImageOnly(t *testing.T) {
	contents := []types.ContentBlock{
		types.NewImageBlock("https://example.com/xray.png"),
	}
	result := contentsToOpenAI(contents)

	b, _ := json.Marshal(result)
	var blocks []map[string]any
	json.Unmarshal(b, &blocks)
	if len(blocks) != 1 {
		t.Fatalf("len = %d, want 1", len(blocks))
	}
	if blocks[0]["type"] != "image_url" {
		t.Errorf("type = %v, want image_url", blocks[0]["type"])
	}
}

func TestContentsToOpenAI_MultipleText(t *testing.T) {
	contents := []types.ContentBlock{
		types.NewTextBlock("first"),
		types.NewTextBlock("second"),
	}
	result := contentsToOpenAI(contents)

	b, _ := json.Marshal(result)
	var blocks []map[string]any
	json.Unmarshal(b, &blocks)
	if len(blocks) != 2 {
		t.Fatalf("len = %d, want 2", len(blocks))
	}
}

func TestContentsToOpenAI_Empty(t *testing.T) {
	result := contentsToOpenAI(nil)
	if result != nil {
		t.Errorf("empty input = %v, want nil", result)
	}
}

func TestChatMessage_MultimodalSerialization(t *testing.T) {
	msg := chatMessage{
		Role: "user",
		Content: []openAIContentBlock{
			{Type: "text", Text: "hello"},
			{Type: "image_url", ImageURL: &openAIImageURL{URL: "https://example.com/img.png"}},
		},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	json.Unmarshal(b, &parsed)
	content, ok := parsed["content"].([]any)
	if !ok {
		t.Fatalf("content should be array, got %T", parsed["content"])
	}
	if len(content) != 2 {
		t.Fatalf("len(content) = %d, want 2", len(content))
	}
}

func TestBuildChatMessages_Integration(t *testing.T) {
	req := types.GenerateRequest{
		Messages: []types.Message{
			types.NewTextMessage("system", "You are helpful"),
			{
				Role: "user",
				Contents: []types.ContentBlock{
					types.NewTextBlock("What is this?"),
					types.NewImageBlock("https://example.com/test.png"),
				},
			},
		},
	}

	messages := buildOpenAIMessages(req)
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}

	// First message: system, pure text
	b1, _ := json.Marshal(messages[0].Content)
	if string(b1) != `"You are helpful"` {
		t.Errorf("system content = %s, want plain text", b1)
	}

	// Second message: user, multimodal
	b2, _ := json.Marshal(messages[1].Content)
	// Should be array of content blocks
	var blocks []map[string]any
	if err := json.Unmarshal(b2, &blocks); err != nil {
		t.Errorf("user content should be array: %v", err)
	}
}
