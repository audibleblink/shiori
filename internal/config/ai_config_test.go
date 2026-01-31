package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIConfigValidation(t *testing.T) {
	t.Run("DisabledConfigIsValid", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled: false,
		}
		err := cfg.IsValid()
		require.NoError(t, err)
	})

	t.Run("EnabledWithoutAPIKeyFails", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "anthropic API key")
	})

	t.Run("EnabledWithoutOllamaHostFails", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ollama host")
	})

	t.Run("ValidConfigPasses", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test", Model: "claude-sonnet-4-20250514"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434", Model: "embeddinggemma"},
			Retrieval: &RetrievalConfig{SimilarityThreshold: 0.7, MaxChunks: 20},
			Embedding: &EmbeddingConfig{ChunkSize: 512, ChunkOverlap: 50, VectorDimension: 768},
		}
		err := cfg.IsValid()
		require.NoError(t, err)
	})

	t.Run("InvalidSimilarityThresholdFails", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
			Retrieval: &RetrievalConfig{SimilarityThreshold: 1.5, MaxChunks: 20},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "similarity threshold")
	})

	t.Run("NegativeSimilarityThresholdFails", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
			Retrieval: &RetrievalConfig{SimilarityThreshold: -0.5, MaxChunks: 20},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "similarity threshold")
	})

	t.Run("ZeroMaxChunksFails", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
			Retrieval: &RetrievalConfig{SimilarityThreshold: 0.7, MaxChunks: 0},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max chunks")
	})

	t.Run("ChunkSizeTooSmallFails", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
			Embedding: &EmbeddingConfig{ChunkSize: 50, ChunkOverlap: 10, VectorDimension: 768},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chunk size")
	})

	t.Run("ChunkOverlapTooLargeFails", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
			Embedding: &EmbeddingConfig{ChunkSize: 512, ChunkOverlap: 600, VectorDimension: 768},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chunk overlap")
	})

	t.Run("NegativeChunkOverlapFails", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
			Embedding: &EmbeddingConfig{ChunkSize: 512, ChunkOverlap: -10, VectorDimension: 768},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chunk overlap")
	})

	t.Run("ZeroVectorDimensionFails", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
			Embedding: &EmbeddingConfig{ChunkSize: 512, ChunkOverlap: 50, VectorDimension: 0},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "vector dimension")
	})
}

func TestAIConfigIsConfigured(t *testing.T) {
	t.Run("DisabledReturnsFalse", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   false,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
		}
		assert.False(t, cfg.IsConfigured())
	})

	t.Run("MissingAPIKeyReturnsFalse", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
		}
		assert.False(t, cfg.IsConfigured())
	})

	t.Run("MissingOllamaHostReturnsFalse", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{},
		}
		assert.False(t, cfg.IsConfigured())
	})

	t.Run("FullConfigReturnsTrue", func(t *testing.T) {
		cfg := &AIConfig{
			Enabled:   true,
			Anthropic: &AnthropicConfig{APIKey: "sk-test"},
			Ollama:    &OllamaConfig{Host: "http://localhost:11434"},
		}
		assert.True(t, cfg.IsConfigured())
	})
}
