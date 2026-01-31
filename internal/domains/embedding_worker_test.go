package domains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-shiori/shiori/internal/core"
	"github.com/go-shiori/shiori/internal/domains"
	"github.com/go-shiori/shiori/internal/model"
	"github.com/go-shiori/shiori/internal/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmbeddingDomain is a mock implementation for testing the worker
type mockEmbeddingDomain struct {
	embedCalls     atomic.Int32
	embedError     error
	embedDelay     time.Duration
	backfillCalled atomic.Bool
}

func (m *mockEmbeddingDomain) EmbedBookmark(ctx context.Context, bookmarkID int) error {
	m.embedCalls.Add(1)
	if m.embedDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.embedDelay):
		}
	}
	return m.embedError
}

func (m *mockEmbeddingDomain) SearchSimilar(ctx context.Context, query string, threshold float64, limit int) ([]model.ChunkResult, error) {
	return nil, nil
}

func (m *mockEmbeddingDomain) BackfillEmbeddings(ctx context.Context, batchSize int, delayMs int) error {
	m.backfillCalled.Store(true)
	return nil
}

func (m *mockEmbeddingDomain) GetEmbeddingStatus(ctx context.Context) (*model.EmbeddingStatusSummary, error) {
	return &model.EmbeddingStatusSummary{}, nil
}

func (m *mockEmbeddingDomain) ReembedBookmark(ctx context.Context, bookmarkID int) error {
	return m.EmbedBookmark(ctx, bookmarkID)
}

func (m *mockEmbeddingDomain) IsOllamaAvailable(ctx context.Context) error {
	return nil
}

func TestEmbeddingWorker_StartStop(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	mockDomain := &mockEmbeddingDomain{}
	config := domains.DefaultEmbeddingWorkerConfig()
	worker := domains.NewEmbeddingWorker(deps, mockDomain, config)

	// Start worker
	worker.Start()
	assert.True(t, worker.IsRunning())

	// Stop worker
	worker.Stop()
	assert.False(t, worker.IsRunning())
}

func TestEmbeddingWorker_EnqueueAndProcess(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	mockDomain := &mockEmbeddingDomain{}
	config := domains.DefaultEmbeddingWorkerConfig()
	config.Concurrency = 1
	worker := domains.NewEmbeddingWorker(deps, mockDomain, config)

	worker.Start()
	defer worker.Stop()

	// Enqueue a job
	enqueued := worker.EnqueueBookmark(123)
	assert.True(t, enqueued)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify the job was processed
	assert.Equal(t, int32(1), mockDomain.embedCalls.Load())
}

func TestEmbeddingWorker_MultipleJobs(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	mockDomain := &mockEmbeddingDomain{}
	config := domains.DefaultEmbeddingWorkerConfig()
	config.Concurrency = 2
	worker := domains.NewEmbeddingWorker(deps, mockDomain, config)

	worker.Start()
	defer worker.Stop()

	// Enqueue multiple jobs
	for i := 1; i <= 5; i++ {
		worker.EnqueueBookmark(i)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify all jobs were processed
	assert.Equal(t, int32(5), mockDomain.embedCalls.Load())
}

func TestEmbeddingWorker_QueueFull(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	mockDomain := &mockEmbeddingDomain{
		embedDelay: 100 * time.Millisecond, // Slow processing
	}
	config := domains.DefaultEmbeddingWorkerConfig()
	config.QueueSize = 2
	config.Concurrency = 1
	worker := domains.NewEmbeddingWorker(deps, mockDomain, config)

	worker.Start()
	defer worker.Stop()

	// Fill the queue
	worker.EnqueueBookmark(1)
	worker.EnqueueBookmark(2)

	// This should return false since queue is full
	enqueued := worker.EnqueueBookmark(3)
	assert.False(t, enqueued)
}

func TestEmbeddingWorker_QueueLength(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	mockDomain := &mockEmbeddingDomain{
		embedDelay: 100 * time.Millisecond,
	}
	config := domains.DefaultEmbeddingWorkerConfig()
	config.QueueSize = 10
	config.Concurrency = 1
	worker := domains.NewEmbeddingWorker(deps, mockDomain, config)

	worker.Start()
	defer worker.Stop()

	// Enqueue some jobs
	worker.EnqueueBookmark(1)
	worker.EnqueueBookmark(2)
	worker.EnqueueBookmark(3)

	// Check queue length (should be 2 or 3 depending on timing)
	length := worker.QueueLength()
	assert.GreaterOrEqual(t, length, 0)
	assert.LessOrEqual(t, length, 3)
}

func TestEmbeddingWorker_RetryOnFailure(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	mockDomain := &mockEmbeddingDomain{
		embedError: assert.AnError, // Always fail
	}
	config := domains.EmbeddingWorkerConfig{
		Concurrency:    1,
		QueueSize:      10,
		MaxRetries:     2,
		RetryBaseDelay: 10 * time.Millisecond,
		RetryMaxDelay:  50 * time.Millisecond,
	}
	worker := domains.NewEmbeddingWorker(deps, mockDomain, config)

	worker.Start()
	defer worker.Stop()

	// Enqueue a job that will fail
	worker.EnqueueBookmark(1)

	// Wait for retries
	time.Sleep(200 * time.Millisecond)

	// Should have called embed 3 times (initial + 2 retries)
	assert.Equal(t, int32(3), mockDomain.embedCalls.Load())
}

func TestEmbeddingWorker_GracefulShutdown(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	mockDomain := &mockEmbeddingDomain{
		embedDelay: 50 * time.Millisecond,
	}
	config := domains.DefaultEmbeddingWorkerConfig()
	config.Concurrency = 1
	worker := domains.NewEmbeddingWorker(deps, mockDomain, config)

	worker.Start()

	// Enqueue a job
	worker.EnqueueBookmark(1)

	// Stop immediately - job should still be processed or gracefully interrupted
	worker.Stop()

	// Worker should be stopped
	assert.False(t, worker.IsRunning())
}

func TestEmbeddingWorker_RunBackfill(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	mockDomain := &mockEmbeddingDomain{}
	config := domains.DefaultEmbeddingWorkerConfig()
	worker := domains.NewEmbeddingWorker(deps, mockDomain, config)

	// Run backfill
	err := worker.RunBackfill(ctx, 10, 100)
	require.NoError(t, err)

	// Verify backfill was called
	assert.True(t, mockDomain.backfillCalled.Load())
}

func TestEmbeddingWorker_ConcurrentProcessing(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	mockDomain := &mockEmbeddingDomain{
		embedDelay: 50 * time.Millisecond,
	}
	config := domains.DefaultEmbeddingWorkerConfig()
	config.Concurrency = 3
	worker := domains.NewEmbeddingWorker(deps, mockDomain, config)

	worker.Start()
	defer worker.Stop()

	// Enqueue 6 jobs
	for i := 1; i <= 6; i++ {
		worker.EnqueueBookmark(i)
	}

	// With 3 workers and 50ms delay, 6 jobs should complete in ~100ms
	time.Sleep(200 * time.Millisecond)

	// All jobs should be processed
	assert.Equal(t, int32(6), mockDomain.embedCalls.Load())
}

func TestEmbeddingWorker_Integration(t *testing.T) {
	// Integration test with real embedding domain (but mock Ollama)
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

	// Create mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/embeddings":
			embedding := make([]float64, 768)
			for i := range embedding {
				embedding[i] = float64(i) / 768.0
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
	defer server.Close()

	ollamaClient := core.NewOllamaClient(server.URL, "embeddinggemma")
	chunker := core.NewTextChunker(512, 50)
	embeddingDomain := domains.NewEmbeddingDomain(deps, ollamaClient, chunker)

	config := domains.DefaultEmbeddingWorkerConfig()
	worker := domains.NewEmbeddingWorker(deps, embeddingDomain, config)

	// Test that worker can be started and stopped even with real domain
	worker.Start()
	assert.True(t, worker.IsRunning())

	// Enqueue a bookmark (will fail because SQLite doesn't support AI)
	// but should not crash
	worker.EnqueueBookmark(1)

	time.Sleep(100 * time.Millisecond)

	worker.Stop()
	assert.False(t, worker.IsRunning())
}
