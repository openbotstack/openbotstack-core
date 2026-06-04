package types

import (
	"strings"
	"testing"
)

func TestNewTextBlock(t *testing.T) {
	b := NewTextBlock("hello")
	if b.Type != "text" {
		t.Errorf("Type = %q, want %q", b.Type, "text")
	}
	if b.Text != "hello" {
		t.Errorf("Text = %q, want %q", b.Text, "hello")
	}
	if b.ImageURL != "" {
		t.Errorf("ImageURL = %q, want empty", b.ImageURL)
	}
}

func TestNewImageBlock(t *testing.T) {
	b := NewImageBlock("https://example.com/img.png")
	if b.Type != "image" {
		t.Errorf("Type = %q, want %q", b.Type, "image")
	}
	if b.ImageURL != "https://example.com/img.png" {
		t.Errorf("ImageURL = %q, want %q", b.ImageURL, "https://example.com/img.png")
	}
	if b.Text != "" {
		t.Errorf("Text = %q, want empty", b.Text)
	}
}

func TestNewTextMessage(t *testing.T) {
	m := NewTextMessage("user", "hello world")
	if m.Role != "user" {
		t.Errorf("Role = %q, want %q", m.Role, "user")
	}
	if len(m.Contents) != 1 {
		t.Fatalf("len(Contents) = %d, want 1", len(m.Contents))
	}
	if m.Contents[0].Type != "text" || m.Contents[0].Text != "hello world" {
		t.Errorf("Contents[0] = %+v, want text block with 'hello world'", m.Contents[0])
	}
}

func TestFlattenToText_TextOnly(t *testing.T) {
	contents := []ContentBlock{
		NewTextBlock("hello"),
		NewTextBlock("world"),
	}
	got := FlattenToText(contents)
	want := "hello\nworld"
	if got != want {
		t.Errorf("FlattenToText = %q, want %q", got, want)
	}
}

func TestFlattenToText_ImageOnly(t *testing.T) {
	contents := []ContentBlock{
		NewImageBlock("https://example.com/img.png"),
	}
	got := FlattenToText(contents)
	if !strings.Contains(got, "[Image:") {
		t.Errorf("FlattenToText = %q, want to contain [Image:]", got)
	}
	if !strings.Contains(got, "https://example.com/img.png") {
		t.Errorf("FlattenToText = %q, want to contain URL", got)
	}
}

func TestFlattenToText_Mixed(t *testing.T) {
	contents := []ContentBlock{
		NewTextBlock("What is in this image?"),
		NewImageBlock("https://example.com/cat.jpg"),
		NewTextBlock("Describe it."),
	}
	got := FlattenToText(contents)
	if !strings.HasPrefix(got, "What is in this image?") {
		t.Errorf("should start with text, got %q", got)
	}
	if !strings.Contains(got, "[Image:") {
		t.Errorf("should contain [Image:], got %q", got)
	}
	if !strings.Contains(got, "Describe it.") {
		t.Errorf("should contain second text, got %q", got)
	}
}

func TestFlattenToText_Empty(t *testing.T) {
	got := FlattenToText(nil)
	if got != "" {
		t.Errorf("FlattenToText(nil) = %q, want empty", got)
	}
}

func TestValidateContents_ValidText(t *testing.T) {
	err := ValidateContents([]ContentBlock{NewTextBlock("hello")})
	if err != nil {
		t.Errorf("ValidateContents = %v, want nil", err)
	}
}

func TestValidateContents_ValidImage(t *testing.T) {
	err := ValidateContents([]ContentBlock{NewImageBlock("https://example.com/img.png")})
	if err != nil {
		t.Errorf("ValidateContents = %v, want nil", err)
	}
}

func TestValidateContents_InvalidType(t *testing.T) {
	err := ValidateContents([]ContentBlock{{Type: "audio"}})
	if err == nil {
		t.Error("ValidateContents should reject unknown type")
	}
}

func TestValidateContents_EmptyText(t *testing.T) {
	err := ValidateContents([]ContentBlock{{Type: "text", Text: ""}})
	if err == nil {
		t.Error("ValidateContents should reject empty text block")
	}
}

func TestValidateContents_EmptyImageURL(t *testing.T) {
	err := ValidateContents([]ContentBlock{{Type: "image", ImageURL: ""}})
	if err == nil {
		t.Error("ValidateContents should reject empty image URL")
	}
}

func TestValidateContents_EmptySlice(t *testing.T) {
	err := ValidateContents([]ContentBlock{})
	if err == nil {
		t.Error("ValidateContents should reject empty contents")
	}
}

func TestValidateContents_NilSlice(t *testing.T) {
	err := ValidateContents(nil)
	if err == nil {
		t.Error("ValidateContents should reject nil contents")
	}
}
