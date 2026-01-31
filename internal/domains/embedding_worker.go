package domains

import (
	"context"
	"sync"
	"time"

	"github.com/go-shiori/shiori/internal/model"
	"github.com/sirupsen/logrus"
)

// EmbeddingWorker handles asynchronous embedding generation for bookmarks
type EmbeddingWorker struct {
	deps            model.Dependencies
	embeddingDomain model.EmbeddingDomain
	logger          *logrus.Logger

	// Job queue for async embedding
	jobQueue chan int
	// Worker control
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	// Configuration
	concurrency    int
	maxRetries     int
	retryBaseDelay time.Duration
	retryMaxDelay  time.Duration
}

// EmbeddingWorkerConfig holds configuration for the embedding worker
type EmbeddingWorkerConfig struct {
	// Concurrency is the number of concurrent embedding workers
	Concurrency int
	// QueueSize is the size of the job queue buffer
	QueueSize int
	// MaxRetries is the maximum number of retries for failed jobs
	MaxRetries int
	// RetryBaseDelay is the base delay for exponential backoff
	RetryBaseDelay time.Duration
	// RetryMaxDelay is the maximum delay for exponential backoff
	RetryMaxDelay time.Duration
}

// DefaultEmbeddingWorkerConfig returns default configuration
func DefaultEmbeddingWorkerConfig() EmbeddingWorkerConfig {
	return EmbeddingWorkerConfig{
		Concurrency:    2,
		QueueSize:      1000,
		MaxRetries:     3,
		RetryBaseDelay: time.Second,
		RetryMaxDelay:  30 * time.Second,
	}
}

// NewEmbeddingWorker creates a new EmbeddingWorker
func NewEmbeddingWorker(deps model.Dependencies, embeddingDomain model.EmbeddingDomain, config EmbeddingWorkerConfig) *EmbeddingWorker {
	ctx, cancel := context.WithCancel(context.Background())

	worker := &EmbeddingWorker{
		deps:            deps,
		embeddingDomain: embeddingDomain,
		logger:          deps.Logger(),
		jobQueue:        make(chan int, config.QueueSize),
		ctx:             ctx,
		cancel:          cancel,
		concurrency:     config.Concurrency,
		maxRetries:      config.MaxRetries,
		retryBaseDelay:  config.RetryBaseDelay,
		retryMaxDelay:   config.RetryMaxDelay,
	}

	return worker
}

// Start starts the embedding worker goroutines
func (w *EmbeddingWorker) Start() {
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}

	w.logger.WithField("concurrency", w.concurrency).Info("Embedding worker started")
}

// Stop gracefully shuts down the embedding worker
func (w *EmbeddingWorker) Stop() {
	w.logger.Info("Stopping embedding worker...")

	// Cancel the context to signal workers to stop
	w.cancel()

	// Close the job queue to prevent new jobs
	close(w.jobQueue)

	// Wait for all workers to finish
	w.wg.Wait()

	w.logger.Info("Embedding worker stopped")
}

// EnqueueBookmark adds a bookmark ID to the embedding queue
// Returns true if the job was enqueued, false if the queue is full
func (w *EmbeddingWorker) EnqueueBookmark(bookmarkID int) bool {
	select {
	case w.jobQueue <- bookmarkID:
		w.logger.WithField("bookmark_id", bookmarkID).Debug("Bookmark enqueued for embedding")
		return true
	default:
		w.logger.WithField("bookmark_id", bookmarkID).Warn("Embedding queue full, bookmark will be processed during backfill")
		return false
	}
}

// worker is the main worker goroutine
func (w *EmbeddingWorker) worker(id int) {
	defer w.wg.Done()

	logger := w.logger.WithField("worker_id", id)
	logger.Debug("Embedding worker started")

	for {
		select {
		case <-w.ctx.Done():
			logger.Debug("Embedding worker shutting down")
			return
		case bookmarkID, ok := <-w.jobQueue:
			if !ok {
				logger.Debug("Job queue closed, worker exiting")
				return
			}

			w.processJob(logger, bookmarkID)
		}
	}
}

// processJob processes a single embedding job with retries
func (w *EmbeddingWorker) processJob(logger *logrus.Entry, bookmarkID int) {
	jobLogger := logger.WithField("bookmark_id", bookmarkID)

	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			delay := w.calculateBackoffDelay(attempt)
			jobLogger.WithFields(logrus.Fields{
				"attempt": attempt,
				"delay":   delay,
			}).Debug("Retrying embedding after delay")

			select {
			case <-w.ctx.Done():
				return
			case <-time.After(delay):
			}
		}

		err := w.embeddingDomain.EmbedBookmark(w.ctx, bookmarkID)
		if err == nil {
			jobLogger.Debug("Bookmark embedded successfully")
			return
		}

		lastErr = err
		jobLogger.WithError(err).WithField("attempt", attempt).Warn("Failed to embed bookmark")
	}

	// All retries failed
	jobLogger.WithError(lastErr).Error("Failed to embed bookmark after all retries")
}

// calculateBackoffDelay calculates the delay for exponential backoff
func (w *EmbeddingWorker) calculateBackoffDelay(attempt int) time.Duration {
	delay := w.retryBaseDelay * (1 << (attempt - 1)) // 2^(attempt-1)
	if delay > w.retryMaxDelay {
		delay = w.retryMaxDelay
	}
	return delay
}

// QueueLength returns the current number of jobs in the queue
func (w *EmbeddingWorker) QueueLength() int {
	return len(w.jobQueue)
}

// IsRunning returns true if the worker is running
func (w *EmbeddingWorker) IsRunning() bool {
	select {
	case <-w.ctx.Done():
		return false
	default:
		return true
	}
}

// RunBackfill runs the backfill process to embed all pending bookmarks
func (w *EmbeddingWorker) RunBackfill(ctx context.Context, batchSize int, delayMs int) error {
	w.logger.Info("Starting embedding backfill")
	return w.embeddingDomain.BackfillEmbeddings(ctx, batchSize, delayMs)
}
