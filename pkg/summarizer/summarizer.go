// Package summarizer provides functionality to generate summaries and release notes
// using the Ollama AI model API.
package summarizer

import (
	"context"
	"fmt"
	"log"
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
	log.Printf("Creating new summarizer with model: %s", config.Model)
	if config.Model == "" {
		config.Model = "mistral" // Default to mistral model
		log.Printf("No model specified, using default model: %s", config.Model)
	}

	log.Printf("Initializing Ollama client")
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Printf("Failed to create Ollama client: %v", err)
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}
	log.Printf("Ollama client initialized successfully")

	return &Summarizer{
		client: client,
		config: config,
	}, nil
}

// SummarizeChanges generates a summary of the provided changes
func (s *Summarizer) SummarizeChanges(ctx context.Context, changes string) (string, error) {
	log.Printf("Starting to summarize changes with model: %s", s.config.Model)
	prompt := fmt.Sprintf(`Please analyze this GitHub issue description and create a clear, structured summary suitable for a Jira issue:

%s

Please format the response as follows:
1. Issue Overview (1-2 sentences)
2. Key Details (bullet points)
3. Technical Requirements (if any)
4. Dependencies and Impact (if mentioned)
`, changes)

	log.Printf("Created prompt for summarization")
	request := &api.GenerateRequest{
		Model:  s.config.Model,
		Prompt: prompt,
	}

	var fullResponse strings.Builder
	stream := make(chan api.GenerateResponse)
	errChan := make(chan error, 1)

	log.Printf("Starting generation goroutine")
	go func() {
		defer close(stream)
		if err := s.client.Generate(ctx, request, func(response api.GenerateResponse) error {
			select {
			case stream <- response:
				log.Printf("Received response chunk from model")
				return nil
			case <-ctx.Done():
				log.Printf("Context cancelled during generation")
				return ctx.Err()
			}
		}); err != nil {
			log.Printf("Error during generation: %v", err)
			errChan <- err
		}
		close(errChan)
		log.Printf("Generation goroutine completed")
	}()

	log.Printf("Collecting responses from stream")
	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("Error received from error channel: %v", err)
				return "", fmt.Errorf("failed to generate summary: %w", err)
			}
		case response, ok := <-stream:
			if !ok {
				log.Printf("Stream closed, summarization complete")
				return fullResponse.String(), nil
			}
			log.Printf("Appending response chunk to full response")
			fullResponse.WriteString(response.Response)
		case <-ctx.Done():
			log.Printf("Context deadline exceeded or cancelled")
			return "", ctx.Err()
		}
	}
}

// SummarizeWithCustomPrompt generates a summary using a custom prompt template
func (s *Summarizer) SummarizeWithCustomPrompt(ctx context.Context, content, promptTemplate string) (string, error) {
	log.Printf("Starting custom prompt summarization with model: %s", s.config.Model)
	// If no custom prompt is provided, use a default one for GitHub to Jira conversion
	if promptTemplate == "" {
		log.Printf("No custom prompt provided, using default prompt")
		promptTemplate = `Please analyze this GitHub issue description and create a clear, structured summary for Jira:

%s

Please format the response as follows:
1. Issue Overview (1-2 sentences)
2. Key Details (bullet points)
3. Technical Requirements (if any)
4. Dependencies and Impact (if mentioned)
`
	}

	log.Printf("Creating generation request")
	request := &api.GenerateRequest{
		Model:  s.config.Model,
		Prompt: fmt.Sprintf(promptTemplate, content),
	}

	var fullResponse strings.Builder
	stream := make(chan api.GenerateResponse)
	errChan := make(chan error, 1)

	log.Printf("Starting generation goroutine")
	go func() {
		defer close(stream)
		if err := s.client.Generate(ctx, request, func(response api.GenerateResponse) error {
			select {
			case stream <- response:
				log.Printf("Received response chunk from model")
				return nil
			case <-ctx.Done():
				log.Printf("Context cancelled during generation")
				return ctx.Err()
			}
		}); err != nil {
			log.Printf("Error during generation: %v", err)
			errChan <- err
		}
		close(errChan)
		log.Printf("Generation goroutine completed")
	}()

	log.Printf("Collecting responses from stream")
	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("Error received from error channel: %v", err)
				return "", fmt.Errorf("failed to generate summary: %w", err)
			}
		case response, ok := <-stream:
			if !ok {
				log.Printf("Stream closed, summarization complete")
				return fullResponse.String(), nil
			}
			log.Printf("Appending response chunk to full response")
			fullResponse.WriteString(response.Response)
		case <-ctx.Done():
			log.Printf("Context deadline exceeded or cancelled")
			return "", ctx.Err()
		}
	}
}
