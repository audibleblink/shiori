package domains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-shiori/shiori/internal/core"
	"github.com/go-shiori/shiori/internal/domains"
	"github.com/go-shiori/shiori/internal/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockOllama(t *testing.T, embeddingSize int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/embeddings":
			embedding := make([]float64, embeddingSize)
			for i := range embedding {
				embedding[i] = float64(i) / float64(embeddingSize)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"embedding": embedding,
			})
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]string{
					{"name": "embeddinggemma:latest"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestEmbeddingDomain_IsOllamaAvailable(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	t.Run("Available", func(t *testing.T) {
		server := setupMockOllama(t, 768)
		defer server.Close()

		ollamaClient := core.NewOllamaClient(server.URL, "embeddinggemma")
		chunker := core.NewTextChunker(512, 50)
		embeddingDomain := domains.NewEmbeddingDomain(deps, ollamaClient, chunker)

		err := embeddingDomain.IsOllamaAvailable(ctx)
		require.NoError(t, err)
	})

	t.Run("Unavailable", func(t *testing.T) {
		ollamaClient := core.NewOllamaClient("http://localhost:99999", "embeddinggemma")
		chunker := core.NewTextChunker(512, 50)
		embeddingDomain := domains.NewEmbeddingDomain(deps, ollamaClient, chunker)

		err := embeddingDomain.IsOllamaAvailable(ctx)
		require.Error(t, err)
	})
}

func TestEmbeddingDomain_EmbedBookmark_NotSupported(t *testing.T) {
	// This tests that SQLite (the default test database) returns ErrAINotSupported
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	server := setupMockOllama(t, 768)
	defer server.Close()

	ollamaClient := core.NewOllamaClient(server.URL, "embeddinggemma")
	chunker := core.NewTextChunker(512, 50)
	embeddingDomain := domains.NewEmbeddingDomain(deps, ollamaClient, chunker)

	// SQLite doesn't support AI features
	err := embeddingDomain.EmbedBookmark(ctx, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support AI features")
}

func TestEmbeddingDomain_SearchSimilar_NotSupported(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	server := setupMockOllama(t, 768)
	defer server.Close()

	ollamaClient := core.NewOllamaClient(server.URL, "embeddinggemma")
	chunker := core.NewTextChunker(512, 50)
	embeddingDomain := domains.NewEmbeddingDomain(deps, ollamaClient, chunker)

	// SQLite doesn't support AI features
	_, err := embeddingDomain.SearchSimilar(ctx, "test query", 0.7, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support AI features")
}

func TestEmbeddingDomain_GetEmbeddingStatus_NotSupported(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	server := setupMockOllama(t, 768)
	defer server.Close()

	ollamaClient := core.NewOllamaClient(server.URL, "embeddinggemma")
	chunker := core.NewTextChunker(512, 50)
	embeddingDomain := domains.NewEmbeddingDomain(deps, ollamaClient, chunker)

	// SQLite doesn't support AI features
	_, err := embeddingDomain.GetEmbeddingStatus(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support AI features")
}

func TestEmbeddingDomain_BackfillEmbeddings_NotSupported(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	server := setupMockOllama(t, 768)
	defer server.Close()

	ollamaClient := core.NewOllamaClient(server.URL, "embeddinggemma")
	chunker := core.NewTextChunker(512, 50)
	embeddingDomain := domains.NewEmbeddingDomain(deps, ollamaClient, chunker)

	// SQLite doesn't support AI features
	err := embeddingDomain.BackfillEmbeddings(ctx, 10, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support AI features")
}

func TestEmbeddingDomain_ReembedBookmark_NotSupported(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	server := setupMockOllama(t, 768)
	defer server.Close()

	ollamaClient := core.NewOllamaClient(server.URL, "embeddinggemma")
	chunker := core.NewTextChunker(512, 50)
	embeddingDomain := domains.NewEmbeddingDomain(deps, ollamaClient, chunker)

	// SQLite doesn't support AI features
	err := embeddingDomain.ReembedBookmark(ctx, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support AI features")
}
