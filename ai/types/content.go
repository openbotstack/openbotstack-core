package types

import (
	"errors"
	"fmt"
	"strings"
)

// ContentBlock represents a single content element in a message.
type ContentBlock struct {
	Type     string `json:"type"`                   // "text" | "image"
	Text     string `json:"text,omitempty"`         // for type="text"
	ImageURL string `json:"image_url,omitempty"`    // for type="image" (URL or data URI)
	Base64   string `json:"base64,omitempty"`       // for type="image" (raw base64 data)
	Media    string `json:"media_type,omitempty"`   // for type="image" (e.g. "image/png")
}

// NewTextBlock creates a text content block.
func NewTextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

// NewImageBlock creates an image content block from a URL.
func NewImageBlock(imageURL string) ContentBlock {
	return ContentBlock{Type: "image", ImageURL: imageURL}
}

// NewImageBlockBase64 creates an image content block from base64 data.
// mediaType should be "image/png", "image/jpeg", etc.
func NewImageBlockBase64(mediaType, base64Data string) ContentBlock {
	return ContentBlock{Type: "image", Media: mediaType, Base64: base64Data}
}

// ImageRef returns the best available image reference:
// - data URI if base64 data is present
// - raw URL otherwise
func (c ContentBlock) ImageRef() string {
	if c.Base64 != "" {
		media := c.Media
		if media == "" {
			media = "image/png"
		}
		return "data:" + media + ";base64," + c.Base64
	}
	return c.ImageURL
}

// NewTextMessage creates a message with a single text content block.
func NewTextMessage(role, text string) Message {
	return Message{
		Role:     role,
		Contents: []ContentBlock{NewTextBlock(text)},
	}
}

// FlattenToText converts Contents to plain text for planner consumption.
// text blocks → their text content
// image blocks → "[Image: url]" or "[Image: base64 data (N bytes)]"
func FlattenToText(contents []ContentBlock) string {
	if len(contents) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, c := range contents {
		if i > 0 {
			sb.WriteByte('\n')
		}
		switch c.Type {
		case "text":
			sb.WriteString(c.Text)
		case "image":
			if c.ImageURL != "" {
				fmt.Fprintf(&sb, "[Image: %s]", c.ImageURL)
			} else {
				fmt.Fprintf(&sb, "[Image: base64 %s (%d bytes)]", c.Media, len(c.Base64))
			}
		}
	}
	return sb.String()
}

// ValidateContents checks block validity.
func ValidateContents(contents []ContentBlock) error {
	if len(contents) == 0 {
		return errors.New("contents must not be empty")
	}
	for i, c := range contents {
		switch c.Type {
		case "text":
			if strings.TrimSpace(c.Text) == "" {
				return fmt.Errorf("content block %d: text block must not be empty", i)
			}
		case "image":
			if strings.TrimSpace(c.ImageURL) == "" && strings.TrimSpace(c.Base64) == "" {
				return fmt.Errorf("content block %d: image block must have a URL or base64 data", i)
			}
		default:
			return fmt.Errorf("content block %d: unknown type %q", i, c.Type)
		}
	}
	return nil
}
