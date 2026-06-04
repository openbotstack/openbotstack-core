package types

import (
	"errors"
	"fmt"
	"strings"
)

// ContentBlock represents a single content element in a message.
type ContentBlock struct {
	Type     string `json:"type"`                // "text" | "image"
	Text     string `json:"text,omitempty"`      // for type="text"
	ImageURL string `json:"image_url,omitempty"` // for type="image"
}

// NewTextBlock creates a text content block.
func NewTextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

// NewImageBlock creates an image content block.
func NewImageBlock(imageURL string) ContentBlock {
	return ContentBlock{Type: "image", ImageURL: imageURL}
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
// image blocks → "[Image: url]"
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
			fmt.Fprintf(&sb, "[Image: %s]", c.ImageURL)
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
			if strings.TrimSpace(c.ImageURL) == "" {
				return fmt.Errorf("content block %d: image block must have a URL", i)
			}
		default:
			return fmt.Errorf("content block %d: unknown type %q", i, c.Type)
		}
	}
	return nil
}
