package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient provides an interface to generate embeddings using Ollama
type OllamaClient struct {
	host       string
	model      string
	httpClient *http.Client
}

// OllamaClientOption is a function that configures an OllamaClient
type OllamaClientOption func(*OllamaClient)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) OllamaClientOption {
	return func(c *OllamaClient) {
		c.httpClient = client
	}
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(host, model string, opts ...OllamaClientOption) *OllamaClient {
	client := &OllamaClient{
		host:  host,
		model: model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Embedding generation can take a while
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// ollamaEmbeddingRequest is the request body for Ollama embedding API
type ollamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbeddingResponse is the response from Ollama embedding API
type ollamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// GenerateEmbedding generates an embedding vector for the given text
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	reqBody := ollamaEmbeddingRequest{
		Model:  c.model,
		Prompt: text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/embeddings", c.host)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("Ollama returned empty embedding")
	}

	// Convert float64 to float32
	embedding := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// GenerateEmbeddings generates embeddings for multiple texts
func (c *OllamaClient) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		embedding, err := c.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// IsAvailable checks if Ollama is reachable and the model is available
func (c *OllamaClient) IsAvailable(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/tags", c.host)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama is not reachable at %s: %w", c.host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	// Check if the model is available
	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	for _, model := range tagsResp.Models {
		if model.Name == c.model || model.Name == c.model+":latest" {
			return nil
		}
	}

	return fmt.Errorf("model %s is not available in Ollama", c.model)
}
