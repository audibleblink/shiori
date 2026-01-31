package domains

import (
	"context"
	"fmt"
	"time"

	"github.com/go-shiori/shiori/internal/core"
	"github.com/go-shiori/shiori/internal/model"
)

// EmbeddingDomain handles embedding generation and similarity search for bookmarks
type EmbeddingDomain struct {
	deps         model.Dependencies
	ollamaClient *core.OllamaClient
	chunker      *core.TextChunker
}

// NewEmbeddingDomain creates a new EmbeddingDomain
func NewEmbeddingDomain(deps model.Dependencies, ollamaClient *core.OllamaClient, chunker *core.TextChunker) *EmbeddingDomain {
	return &EmbeddingDomain{
		deps:         deps,
		ollamaClient: ollamaClient,
		chunker:      chunker,
	}
}

// EmbedBookmark generates and stores embeddings for a bookmark
func (d *EmbeddingDomain) EmbedBookmark(ctx context.Context, bookmarkID int) error {
	db := d.deps.Database()

	// Check if database supports AI features
	if !db.SupportsAI() {
		return fmt.Errorf("database does not support AI features")
	}

	// Get the bookmark
	bookmark, exists, err := db.GetBookmark(ctx, bookmarkID, "")
	if err != nil {
		return fmt.Errorf("failed to get bookmark: %w", err)
	}
	if !exists {
		return model.ErrBookmarkNotFound
	}

	// Mark as processing
	if err := db.UpdateBookmarkEmbeddingStatus(ctx, bookmarkID, model.EmbeddingStatusProcessing, nil); err != nil {
		return fmt.Errorf("failed to update embedding status: %w", err)
	}

	// Get content to embed
	content := d.getEmbeddableContent(&bookmark)
	if content == "" {
		// No content to embed, mark as skipped
		if err := db.UpdateBookmarkEmbeddingStatus(ctx, bookmarkID, model.EmbeddingStatusSkipped, nil); err != nil {
			return fmt.Errorf("failed to update embedding status: %w", err)
		}
		return nil
	}

	// Chunk the content
	chunks := d.chunker.ChunkText(content)
	if len(chunks) == 0 {
		// No chunks generated, mark as skipped
		if err := db.UpdateBookmarkEmbeddingStatus(ctx, bookmarkID, model.EmbeddingStatusSkipped, nil); err != nil {
			return fmt.Errorf("failed to update embedding status: %w", err)
		}
		return nil
	}

	// Delete any existing chunks for this bookmark
	if err := db.DeleteBookmarkChunks(ctx, bookmarkID); err != nil {
		errMsg := err.Error()
		_ = db.UpdateBookmarkEmbeddingStatus(ctx, bookmarkID, model.EmbeddingStatusFailed, &errMsg)
		return fmt.Errorf("failed to delete existing chunks: %w", err)
	}

	// Generate embeddings for each chunk
	bookmarkChunks := make([]model.BookmarkChunk, len(chunks))
	for i, chunkText := range chunks {
		// Prepare text for embedding
		preparedText := core.PrepareTextForEmbedding(chunkText)

		embedding, err := d.ollamaClient.GenerateEmbedding(ctx, preparedText)
		if err != nil {
			errMsg := fmt.Sprintf("failed to generate embedding for chunk %d: %s", i, err.Error())
			_ = db.UpdateBookmarkEmbeddingStatus(ctx, bookmarkID, model.EmbeddingStatusFailed, &errMsg)
			return fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
		}

		bookmarkChunks[i] = model.BookmarkChunk{
			BookmarkID: bookmarkID,
			ChunkIndex: i,
			Content:    chunkText,
			Embedding:  embedding,
			CreatedAt:  time.Now(),
		}
	}

	// Store chunks in database
	if err := db.CreateBookmarkChunks(ctx, bookmarkChunks); err != nil {
		errMsg := err.Error()
		_ = db.UpdateBookmarkEmbeddingStatus(ctx, bookmarkID, model.EmbeddingStatusFailed, &errMsg)
		return fmt.Errorf("failed to store chunks: %w", err)
	}

	// Mark as completed
	if err := db.UpdateBookmarkEmbeddingStatus(ctx, bookmarkID, model.EmbeddingStatusCompleted, nil); err != nil {
		return fmt.Errorf("failed to update embedding status: %w", err)
	}

	return nil
}

// SearchSimilar finds chunks similar to the query text
func (d *EmbeddingDomain) SearchSimilar(ctx context.Context, query string, threshold float64, limit int) ([]model.ChunkResult, error) {
	db := d.deps.Database()

	// Check if database supports AI features
	if !db.SupportsAI() {
		return nil, fmt.Errorf("database does not support AI features")
	}

	// Prepare and embed the query
	preparedQuery := core.PrepareTextForEmbedding(query)
	queryEmbedding, err := d.ollamaClient.GenerateEmbedding(ctx, preparedQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search for similar chunks
	results, err := db.SearchSimilarChunks(ctx, queryEmbedding, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar chunks: %w", err)
	}

	return results, nil
}

// BackfillEmbeddings processes all bookmarks with pending embedding status
func (d *EmbeddingDomain) BackfillEmbeddings(ctx context.Context, batchSize int, delayMs int) error {
	db := d.deps.Database()

	// Check if database supports AI features
	if !db.SupportsAI() {
		return fmt.Errorf("database does not support AI features")
	}

	logger := d.deps.Logger()
	totalProcessed := 0

	for {
		// Get a batch of pending bookmarks
		bookmarks, err := db.GetBookmarksByEmbeddingStatus(ctx, model.EmbeddingStatusPending, batchSize)
		if err != nil {
			return fmt.Errorf("failed to get pending bookmarks: %w", err)
		}

		if len(bookmarks) == 0 {
			break // No more pending bookmarks
		}

		for _, bookmark := range bookmarks {
			// Check context for cancellation
			select {
			case <-ctx.Done():
				logger.Info("Backfill cancelled")
				return ctx.Err()
			default:
			}

			if err := d.EmbedBookmark(ctx, bookmark.ID); err != nil {
				logger.WithField("bookmark_id", bookmark.ID).
					WithError(err).
					Warn("Failed to embed bookmark during backfill")
				// Continue with next bookmark
			} else {
				totalProcessed++
			}

			// Delay between embeddings to avoid overwhelming Ollama
			if delayMs > 0 {
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
		}

		if logger != nil {
			logger.WithField("processed", totalProcessed).Info("Backfill progress")
		}
	}

	if logger != nil {
		logger.WithField("total", totalProcessed).Info("Backfill complete")
	}

	return nil
}

// GetEmbeddingStatus returns the embedding status summary
func (d *EmbeddingDomain) GetEmbeddingStatus(ctx context.Context) (*model.EmbeddingStatusSummary, error) {
	db := d.deps.Database()

	if !db.SupportsAI() {
		return nil, fmt.Errorf("database does not support AI features")
	}

	return db.GetEmbeddingStatusSummary(ctx)
}

// ReembedBookmark re-generates embeddings for a bookmark (e.g., after content update)
func (d *EmbeddingDomain) ReembedBookmark(ctx context.Context, bookmarkID int) error {
	db := d.deps.Database()

	if !db.SupportsAI() {
		return fmt.Errorf("database does not support AI features")
	}

	// Reset status to pending then embed
	if err := db.UpdateBookmarkEmbeddingStatus(ctx, bookmarkID, model.EmbeddingStatusPending, nil); err != nil {
		return fmt.Errorf("failed to reset embedding status: %w", err)
	}

	return d.EmbedBookmark(ctx, bookmarkID)
}

// getEmbeddableContent extracts content suitable for embedding from a bookmark
func (d *EmbeddingDomain) getEmbeddableContent(bookmark *model.BookmarkDTO) string {
	// Combine title, excerpt, and content
	var parts []string

	if bookmark.Title != "" {
		parts = append(parts, "Title: "+bookmark.Title)
	}

	if bookmark.Excerpt != "" {
		parts = append(parts, "Excerpt: "+bookmark.Excerpt)
	}

	if bookmark.Content != "" {
		parts = append(parts, "Content: "+bookmark.Content)
	}

	if len(parts) == 0 {
		return ""
	}

	content := ""
	for i, part := range parts {
		if i > 0 {
			content += "\n\n"
		}
		content += part
	}

	return content
}

// IsOllamaAvailable checks if Ollama is reachable
func (d *EmbeddingDomain) IsOllamaAvailable(ctx context.Context) error {
	return d.ollamaClient.IsAvailable(ctx)
}
