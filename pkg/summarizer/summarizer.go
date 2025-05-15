// Package summarizer provides functionality to generate summaries and release notes
// using the Ollama AI model API.
package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmorganca/ollama/api"
)

// Config holds the configuration for the summarizer
type Config struct {
	Model     string
	OllamaURL string
}

// Summarizer provides methods to generate summaries using Ollama
type Summarizer struct {
	client *api.Client
	config Config
}

// New creates a new instance of Summarizer with the given configuration
func New(config Config) (*Summarizer, error) {
	if config.Model == "" {
		config.Model = "mistral" // Default to mistral model
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	return &Summarizer{
		client: client,
		config: config,
	}, nil
}

// SummarizeChanges generates a summary of the provided changes
func (s *Summarizer) SummarizeChanges(ctx context.Context, changes string) (string, error) {
	prompt := fmt.Sprintf(`Please analyze these changes and create a concise, well-structured summary suitable for release notes:

%s

Please format the response as follows:
1. A brief overview (2-3 sentences)
2. Key changes (bullet points)
3. Impact on users/developers
`, changes)

	request := &api.GenerateRequest{
		Model:  s.config.Model,
		Prompt: prompt,
	}

	var fullResponse strings.Builder
	stream := make(chan api.GenerateResponse)

	go func() {
		defer close(stream)
		err := s.client.Generate(ctx, request, func(response api.GenerateResponse) error {
			stream <- response
			return nil
		})
		if err != nil {
			// Handle error by sending empty response
			close(stream)
		}
	}()

	for response := range stream {
		fullResponse.WriteString(response.Response)
	}

	return fullResponse.String(), nil
}

// SummarizeWithCustomPrompt generates a summary using a custom prompt template
func (s *Summarizer) SummarizeWithCustomPrompt(ctx context.Context, content, promptTemplate string) (string, error) {
	request := &api.GenerateRequest{
		Model:  s.config.Model,
		Prompt: fmt.Sprintf(promptTemplate, content),
	}

	var fullResponse strings.Builder
	stream := make(chan api.GenerateResponse)

	go func() {
		defer close(stream)
		err := s.client.Generate(ctx, request, func(response api.GenerateResponse) error {
			stream <- response
			return nil
		})
		if err != nil {
			// Handle error by sending empty response
			close(stream)
		}
	}()

	for response := range stream {
		fullResponse.WriteString(response.Response)
	}

	return fullResponse.String(), nil
}
