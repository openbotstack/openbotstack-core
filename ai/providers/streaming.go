package providers

import (
	"context"
	"net/http"

	"github.com/openbotstack/openbotstack-core/control/skills"
)

// openAICompatibleStream will be implemented in Task 5.
// This is a temporary stub to allow compilation.
func openAICompatibleStream(
	ctx context.Context,
	client *http.Client,
	baseURL, apiKey, model string,
	headers map[string]string,
	req skills.GenerateRequest,
	maxRetries int,
) (<-chan skills.StreamChunk, error) {
	ch := make(chan skills.StreamChunk)
	close(ch)
	return ch, nil
}
