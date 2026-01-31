package model

import "time"

// EmbeddingStatus represents the status of bookmark embedding
type EmbeddingStatus string

const (
	EmbeddingStatusPending    EmbeddingStatus = "pending"
	EmbeddingStatusProcessing EmbeddingStatus = "processing"
	EmbeddingStatusCompleted  EmbeddingStatus = "completed"
	EmbeddingStatusFailed     EmbeddingStatus = "failed"
	EmbeddingStatusSkipped    EmbeddingStatus = "skipped"
)

// BookmarkChunk represents a chunk of bookmark content with its embedding
type BookmarkChunk struct {
	ID         int       `db:"id"          json:"id"`
	BookmarkID int       `db:"bookmark_id" json:"bookmarkId"`
	ChunkIndex int       `db:"chunk_index" json:"chunkIndex"`
	Content    string    `db:"content"     json:"content"`
	Embedding  []float32 `db:"embedding"   json:"-"` // Don't expose embedding in JSON
	CreatedAt  time.Time `db:"created_at"  json:"createdAt"`
}

// ChunkResult represents a search result with similarity score
type ChunkResult struct {
	BookmarkID int     `db:"bookmark_id" json:"bookmarkId"`
	ChunkIndex int     `db:"chunk_index" json:"chunkIndex"`
	Content    string  `db:"content"     json:"content"`
	Similarity float64 `db:"similarity"  json:"similarity"`
}

// EmbeddingStatusSummary contains aggregate embedding status information
type EmbeddingStatusSummary struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Skipped    int `json:"skipped"`
}

// BookmarkEmbeddingInfo contains embedding-specific fields for a bookmark
type BookmarkEmbeddingInfo struct {
	EmbeddingStatus EmbeddingStatus `db:"embedding_status" json:"embeddingStatus"`
	EmbeddingError  *string         `db:"embedding_error"  json:"embeddingError,omitempty"`
	EmbeddedAt      *time.Time      `db:"embedded_at"      json:"embeddedAt,omitempty"`
}
