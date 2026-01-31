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

// ChatMessage represents a message in a conversation
type ChatMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// ChatResponse represents the response from a chat request
type ChatResponse struct {
	Message string           `json:"message"`
	Sources []BookmarkSource `json:"sources"`
}

// BookmarkSource represents a source bookmark for citation
type BookmarkSource struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Excerpt string `json:"excerpt"`
}

// StreamChatEvent represents an event from streaming chat
type StreamChatEvent struct {
	Type    string           `json:"type"`    // "chunk", "sources", "done", "error"
	Content string           `json:"content"` // Text content for chunk events
	Sources []BookmarkSource `json:"sources"` // Sources for sources event
	Error   error            `json:"-"`       // Error for error events
}

// ChatStatusResponse represents the chat feature status
type ChatStatusResponse struct {
	Enabled         bool                    `json:"enabled"`
	EmbeddingStatus *EmbeddingStatusSummary `json:"embeddingStatus,omitempty"`
	Error           string                  `json:"error,omitempty"`
}
