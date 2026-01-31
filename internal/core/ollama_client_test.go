package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOllamaClient_GenerateEmbedding(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Create mock server
		expectedEmbedding := make([]float64, 768)
		for i := range expectedEmbedding {
			expectedEmbedding[i] = float64(i) / 768.0
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/embeddings", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			var req ollamaEmbeddingRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Equal(t, "embeddinggemma", req.Model)
			assert.Equal(t, "test text", req.Prompt)

			resp := ollamaEmbeddingResponse{Embedding: expectedEmbedding}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewOllamaClient(server.URL, "embeddinggemma")
		embedding, err := client.GenerateEmbedding(context.Background(), "test text")

		require.NoError(t, err)
		assert.Len(t, embedding, 768)
		assert.InDelta(t, float32(0), embedding[0], 0.001)
		assert.InDelta(t, float32(767)/768.0, embedding[767], 0.001)
	})

	t.Run("EmptyText", func(t *testing.T) {
		client := NewOllamaClient("http://localhost:11434", "embeddinggemma")
		_, err := client.GenerateEmbedding(context.Background(), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("ServerError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		client := NewOllamaClient(server.URL, "embeddinggemma")
		_, err := client.GenerateEmbedding(context.Background(), "test")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("EmptyEmbeddingResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ollamaEmbeddingResponse{Embedding: []float64{}}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewOllamaClient(server.URL, "embeddinggemma")
		_, err := client.GenerateEmbedding(context.Background(), "test")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty embedding")
	})

	t.Run("ServerUnavailable", func(t *testing.T) {
		client := NewOllamaClient("http://localhost:99999", "embeddinggemma")
		_, err := client.GenerateEmbedding(context.Background(), "test")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to call Ollama API")
	})
}

func TestOllamaClient_GenerateEmbeddings(t *testing.T) {
	t.Run("MultipleTexts", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			embedding := make([]float64, 768)
			for i := range embedding {
				embedding[i] = float64(callCount)
			}
			resp := ollamaEmbeddingResponse{Embedding: embedding}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewOllamaClient(server.URL, "embeddinggemma")
		embeddings, err := client.GenerateEmbeddings(context.Background(), []string{"text1", "text2", "text3"})

		require.NoError(t, err)
		assert.Len(t, embeddings, 3)
		assert.Equal(t, 3, callCount)

		// Each embedding should have different values based on call order
		assert.InDelta(t, float32(1), embeddings[0][0], 0.001)
		assert.InDelta(t, float32(2), embeddings[1][0], 0.001)
		assert.InDelta(t, float32(3), embeddings[2][0], 0.001)
	})
}

func TestOllamaClient_IsAvailable(t *testing.T) {
	t.Run("Available", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/tags", r.URL.Path)
			resp := struct {
				Models []struct {
					Name string `json:"name"`
				} `json:"models"`
			}{
				Models: []struct {
					Name string `json:"name"`
				}{
					{Name: "embeddinggemma:latest"},
					{Name: "llama2"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewOllamaClient(server.URL, "embeddinggemma")
		err := client.IsAvailable(context.Background())

		require.NoError(t, err)
	})

	t.Run("ModelNotFound", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := struct {
				Models []struct {
					Name string `json:"name"`
				} `json:"models"`
			}{
				Models: []struct {
					Name string `json:"name"`
				}{
					{Name: "llama2"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewOllamaClient(server.URL, "embeddinggemma")
		err := client.IsAvailable(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not available")
	})

	t.Run("ServerUnavailable", func(t *testing.T) {
		client := NewOllamaClient("http://localhost:99999", "embeddinggemma")
		err := client.IsAvailable(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not reachable")
	})
}
