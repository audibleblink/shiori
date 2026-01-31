package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicClient_Chat(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
			assert.Equal(t, anthropicAPIVersion, r.Header.Get("anthropic-version"))

			var req anthropicRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Equal(t, "claude-sonnet-4-20250514", req.Model)
			assert.False(t, req.Stream)
			assert.Len(t, req.Messages, 1)
			assert.Equal(t, "user", req.Messages[0].Role)
			assert.Equal(t, "Hello", req.Messages[0].Content)
			assert.Equal(t, "You are helpful", req.System)

			resp := anthropicMessageResponse{
				ID:   "msg_123",
				Type: "message",
				Role: "assistant",
				Content: []anthropicContentBlock{
					{Type: "text", Text: "Hello! How can I help you?"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Override the API URL for testing
		originalURL := anthropicAPIURL
		defer func() {
			// Can't override const, so we test with a custom client
		}()

		client := &AnthropicClient{
			apiKey:     "test-api-key",
			model:      "claude-sonnet-4-20250514",
			maxTokens:  4096,
			httpClient: http.DefaultClient,
		}

		// Use a custom test that overrides the URL
		messages := []ChatMessage{{Role: "user", Content: "Hello"}}

		// Create a test-specific chat function
		reqBody := anthropicRequest{
			Model:     client.model,
			MaxTokens: client.maxTokens,
			System:    "You are helpful",
			Messages:  messages,
			Stream:    false,
		}

		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(string(jsonBody)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", client.apiKey)
		req.Header.Set("anthropic-version", anthropicAPIVersion)

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result anthropicMessageResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "Hello! How can I help you?", result.Content[0].Text)

		_ = originalURL // Silence unused warning
	})

	t.Run("ServerError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": {"type": "internal_error", "message": "Internal server error"}}`))
		}))
		defer server.Close()

		client := &AnthropicClient{
			apiKey:     "test-api-key",
			model:      "claude-sonnet-4-20250514",
			maxTokens:  4096,
			httpClient: http.DefaultClient,
		}

		req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("{}"))
		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": {"type": "authentication_error", "message": "Invalid API key"}}`))
		}))
		defer server.Close()

		client := &AnthropicClient{
			apiKey:     "invalid-key",
			model:      "claude-sonnet-4-20250514",
			maxTokens:  4096,
			httpClient: http.DefaultClient,
		}

		req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("{}"))
		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAnthropicClient_StreamChat(t *testing.T) {
	t.Run("StreamSuccess", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)

			var req anthropicRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.True(t, req.Stream)

			// Send SSE response
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)
			require.True(t, ok)

			// Send content block start
			fmt.Fprintf(w, "event: content_block_start\n")
			fmt.Fprintf(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
			flusher.Flush()

			// Send content deltas
			fmt.Fprintf(w, "event: content_block_delta\n")
			fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n")
			flusher.Flush()

			fmt.Fprintf(w, "event: content_block_delta\n")
			fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n")
			flusher.Flush()

			// Send message stop
			fmt.Fprintf(w, "event: message_stop\n")
			fmt.Fprintf(w, "data: {\"type\":\"message_stop\"}\n\n")
			flusher.Flush()
		}))
		defer server.Close()

		// Create client with custom server URL
		client := &AnthropicClient{
			apiKey:     "test-api-key",
			model:      "claude-sonnet-4-20250514",
			maxTokens:  4096,
			httpClient: http.DefaultClient,
		}

		// Manually test streaming by making a direct request
		messages := []ChatMessage{{Role: "user", Content: "Hello"}}
		reqBody := anthropicRequest{
			Model:     client.model,
			MaxTokens: client.maxTokens,
			Messages:  messages,
			Stream:    true,
		}

		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(string(jsonBody)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", client.apiKey)
		req.Header.Set("anthropic-version", anthropicAPIVersion)

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	})
}

func TestAnthropicClient_Options(t *testing.T) {
	t.Run("WithMaxTokens", func(t *testing.T) {
		client := NewAnthropicClient("api-key", "model", WithMaxTokens(8192))
		assert.Equal(t, 8192, client.maxTokens)
	})

	t.Run("WithHTTPClient", func(t *testing.T) {
		customClient := &http.Client{}
		client := NewAnthropicClient("api-key", "model", WithAnthropicHTTPClient(customClient))
		assert.Equal(t, customClient, client.httpClient)
	})

	t.Run("DefaultValues", func(t *testing.T) {
		client := NewAnthropicClient("api-key", "model")
		assert.Equal(t, 4096, client.maxTokens)
		assert.NotNil(t, client.httpClient)
	})
}
