package skills

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrKeyNotFound is returned when a KV key doesn't exist.
	ErrKeyNotFound = errors.New("skill: key not found")

	// ErrInvalidURL is returned when URL is empty or invalid.
	ErrInvalidURL = errors.New("skill: invalid URL")
)

// HTTPRequest represents an HTTP request.
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

// HTTPResponse contains the HTTP response.
type HTTPResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// HostAPI provides the host APIs available to Wasm skills.
type HostAPI struct {
	mu sync.RWMutex
	kv map[string][]byte
}

// NewHostAPI creates a new host API instance.
func NewHostAPI() *HostAPI {
	return &HostAPI{
		kv: make(map[string][]byte),
	}
}

func (h *HostAPI) KVGet(ctx context.Context, key string) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	val, exists := h.kv[key]
	if !exists {
		return nil, ErrKeyNotFound
	}
	return val, nil
}

// KVSet stores a value by key.
func (h *HostAPI) KVSet(ctx context.Context, key string, value []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.kv[key] = value
	return nil
}

// KVDelete removes a key.
func (h *HostAPI) KVDelete(ctx context.Context, key string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.kv, key)
	return nil
}

// HTTPFetch performs an HTTP request.
func (h *HostAPI) HTTPFetch(ctx context.Context, req HTTPRequest) (*HTTPResponse, error) {
	if req.URL == "" {
		return nil, ErrInvalidURL
	}

	// TODO: Implement actual HTTP client with sandboxing
	// This is a stub
	return &HTTPResponse{
		StatusCode: 200,
		Headers:    map[string]string{},
		Body:       []byte("stub response"),
	}, nil
}
