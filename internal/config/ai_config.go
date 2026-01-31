package config

import (
	"fmt"
)

// AIConfig contains AI-related configuration for the chat feature
type AIConfig struct {
	Enabled   bool             `env:"AI_ENABLED,default=false"`
	Anthropic *AnthropicConfig `env:",prefix=ANTHROPIC_"`
	Ollama    *OllamaConfig    `env:",prefix=OLLAMA_"`
	Retrieval *RetrievalConfig `env:",prefix=RETRIEVAL_"`
	Embedding *EmbeddingConfig `env:",prefix=EMBEDDING_"`
}

// AnthropicConfig contains Anthropic API settings
type AnthropicConfig struct {
	APIKey    string `env:"API_KEY"`
	Model     string `env:"MODEL,default=claude-sonnet-4-20250514"`
	MaxTokens int    `env:"MAX_TOKENS,default=4096"`
}

// OllamaConfig contains Ollama embedding settings
type OllamaConfig struct {
	Host  string `env:"HOST,default=http://localhost:11434"`
	Model string `env:"MODEL,default=embeddinggemma"`
}

// RetrievalConfig contains RAG retrieval settings
type RetrievalConfig struct {
	SimilarityThreshold float64 `env:"SIMILARITY_THRESHOLD,default=0.7"`
	MaxChunks           int     `env:"MAX_CHUNKS,default=20"`
}

// EmbeddingConfig contains embedding generation settings
type EmbeddingConfig struct {
	ChunkSize       int `env:"CHUNK_SIZE,default=512"`
	ChunkOverlap    int `env:"CHUNK_OVERLAP,default=50"`
	BackfillBatch   int `env:"BACKFILL_BATCH_SIZE,default=10"`
	BackfillDelayMs int `env:"BACKFILL_DELAY_MS,default=1000"`
	VectorDimension int `env:"VECTOR_DIMENSION,default=768"`
}

// IsValid validates the AI configuration
func (c *AIConfig) IsValid() error {
	if !c.Enabled {
		return nil // No validation needed if AI is disabled
	}

	if c.Anthropic == nil || c.Anthropic.APIKey == "" {
		return fmt.Errorf("anthropic API key is required when AI is enabled")
	}

	if c.Ollama == nil || c.Ollama.Host == "" {
		return fmt.Errorf("ollama host is required when AI is enabled")
	}

	if c.Retrieval != nil {
		if c.Retrieval.SimilarityThreshold < 0 || c.Retrieval.SimilarityThreshold > 1 {
			return fmt.Errorf("similarity threshold must be between 0 and 1")
		}
		if c.Retrieval.MaxChunks < 1 {
			return fmt.Errorf("max chunks must be at least 1")
		}
	}

	if c.Embedding != nil {
		if c.Embedding.ChunkSize < 100 {
			return fmt.Errorf("chunk size must be at least 100")
		}
		if c.Embedding.ChunkOverlap < 0 || c.Embedding.ChunkOverlap >= c.Embedding.ChunkSize {
			return fmt.Errorf("chunk overlap must be between 0 and chunk size")
		}
		if c.Embedding.VectorDimension < 1 {
			return fmt.Errorf("vector dimension must be at least 1")
		}
	}

	return nil
}

// IsConfigured returns true if AI is enabled and properly configured
func (c *AIConfig) IsConfigured() bool {
	return c.Enabled && c.Anthropic != nil && c.Anthropic.APIKey != "" &&
		c.Ollama != nil && c.Ollama.Host != ""
}
