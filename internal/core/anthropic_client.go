package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	anthropicAPIURL     = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
)

// AnthropicClient provides an interface to chat using Anthropic's Claude API
type AnthropicClient struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// AnthropicClientOption is a function that configures an AnthropicClient
type AnthropicClientOption func(*AnthropicClient)

// WithAnthropicHTTPClient sets a custom HTTP client
func WithAnthropicHTTPClient(client *http.Client) AnthropicClientOption {
	return func(c *AnthropicClient) {
		c.httpClient = client
	}
}

// WithMaxTokens sets the maximum tokens for responses
func WithMaxTokens(maxTokens int) AnthropicClientOption {
	return func(c *AnthropicClient) {
		c.maxTokens = maxTokens
	}
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(apiKey, model string, opts ...AnthropicClientOption) *AnthropicClient {
	client := &AnthropicClient{
		apiKey:    apiKey,
		model:     model,
		maxTokens: 4096,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Long timeout for streaming responses
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// ChatMessage represents a message in a conversation
type ChatMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// StreamEvent represents an event from the SSE stream
type StreamEvent struct {
	Type    string // "chunk", "sources", "done", "error"
	Content string // Text content for chunk events
	Error   error  // Error for error events
}

// anthropicRequest is the request body for Anthropic Messages API
type anthropicRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system,omitempty"`
	Messages  []ChatMessage `json:"messages"`
	Stream    bool          `json:"stream"`
}

// anthropicStreamEvent represents a streaming event from Anthropic
type anthropicStreamEvent struct {
	Type         string                    `json:"type"`
	Index        int                       `json:"index,omitempty"`
	Delta        *anthropicDelta           `json:"delta,omitempty"`
	ContentBlock *anthropicContentBlock    `json:"content_block,omitempty"`
	Message      *anthropicMessageResponse `json:"message,omitempty"`
	Usage        *anthropicUsage           `json:"usage,omitempty"`
	Error        *anthropicError           `json:"error,omitempty"`
}

type anthropicDelta struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicMessageResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []anthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason,omitempty"`
	StopSequence string                  `json:"stop_sequence,omitempty"`
	Usage        *anthropicUsage         `json:"usage,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// StreamChat sends a chat request and returns a channel of streaming events
func (c *AnthropicClient) StreamChat(ctx context.Context, messages []ChatMessage, systemPrompt string) (<-chan StreamEvent, error) {
	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    systemPrompt,
		Messages:  messages,
		Stream:    true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Anthropic API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body))
	}

	eventChan := make(chan StreamEvent, 100)

	go func() {
		defer resp.Body.Close()
		defer close(eventChan)

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				eventChan <- StreamEvent{Type: "error", Error: ctx.Err()}
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("read error: %w", err)}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Skip event type lines (we parse the data directly)
			if strings.HasPrefix(line, "event:") {
				continue
			}

			if !strings.HasPrefix(line, "data:") {
				continue
			}

			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)

			if data == "[DONE]" {
				eventChan <- StreamEvent{Type: "done"}
				return
			}

			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				// Skip malformed events
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta != nil && event.Delta.Text != "" {
					eventChan <- StreamEvent{Type: "chunk", Content: event.Delta.Text}
				}
			case "message_stop":
				eventChan <- StreamEvent{Type: "done"}
				return
			case "error":
				if event.Error != nil {
					eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("%s: %s", event.Error.Type, event.Error.Message)}
				} else {
					eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("unknown error from Anthropic")}
				}
				return
			}
		}
	}()

	return eventChan, nil
}

// Chat sends a non-streaming chat request and returns the full response
func (c *AnthropicClient) Chat(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    systemPrompt,
		Messages:  messages,
		Stream:    false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result anthropicMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Combine all content blocks
	var response strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			response.WriteString(block.Text)
		}
	}

	return response.String(), nil
}

// IsAvailable checks if the Anthropic API is reachable with valid credentials
func (c *AnthropicClient) IsAvailable(ctx context.Context) error {
	// Send a minimal request to check API key validity
	messages := []ChatMessage{{Role: "user", Content: "Hi"}}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: 1,
		Messages:  messages,
		Stream:    false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Anthropic API is not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid Anthropic API key")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
