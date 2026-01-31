package core

import (
	"fmt"
	"strings"

	"github.com/go-shiori/shiori/internal/model"
)

const systemPromptTemplate = `You are a helpful assistant that answers questions about the user's bookmarked articles and web pages.

Use ONLY the provided context from their bookmarks to answer questions. If the context doesn't contain relevant information, say so clearly rather than making up an answer.

When referencing information, naturally mention which bookmark it came from so the user can explore further.

Keep answers concise but informative. Use markdown formatting when helpful.

Context from bookmarks:
---
%s
---`

// PromptBuilder constructs prompts for the RAG pipeline
type PromptBuilder struct{}

// NewPromptBuilder creates a new PromptBuilder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// ChunkWithBookmark represents a chunk result with associated bookmark metadata
type ChunkWithBookmark struct {
	Chunk    model.ChunkResult
	Bookmark *model.BookmarkDTO
}

// BuildSystemPrompt constructs the system prompt with retrieved context
func (p *PromptBuilder) BuildSystemPrompt(chunks []ChunkWithBookmark) string {
	if len(chunks) == 0 {
		return fmt.Sprintf(systemPromptTemplate, "No relevant bookmarks found.")
	}

	var contextBuilder strings.Builder

	// Group chunks by bookmark
	bookmarkChunks := make(map[int][]ChunkWithBookmark)
	bookmarkOrder := []int{}

	for _, chunk := range chunks {
		if _, exists := bookmarkChunks[chunk.Chunk.BookmarkID]; !exists {
			bookmarkOrder = append(bookmarkOrder, chunk.Chunk.BookmarkID)
		}
		bookmarkChunks[chunk.Chunk.BookmarkID] = append(bookmarkChunks[chunk.Chunk.BookmarkID], chunk)
	}

	// Build context grouped by bookmark
	for i, bookmarkID := range bookmarkOrder {
		chunks := bookmarkChunks[bookmarkID]
		if len(chunks) == 0 {
			continue
		}

		firstChunk := chunks[0]
		bookmark := firstChunk.Bookmark

		if i > 0 {
			contextBuilder.WriteString("\n\n")
		}

		// Write bookmark header
		if bookmark != nil {
			contextBuilder.WriteString(fmt.Sprintf("### %s\n", bookmark.Title))
			if bookmark.URL != "" {
				contextBuilder.WriteString(fmt.Sprintf("URL: %s\n", bookmark.URL))
			}
			contextBuilder.WriteString("\n")
		} else {
			contextBuilder.WriteString(fmt.Sprintf("### Bookmark #%d\n\n", bookmarkID))
		}

		// Write all chunks from this bookmark
		for j, chunk := range chunks {
			if j > 0 {
				contextBuilder.WriteString("\n...\n")
			}
			contextBuilder.WriteString(chunk.Chunk.Content)
		}
	}

	return fmt.Sprintf(systemPromptTemplate, contextBuilder.String())
}

// BuildSystemPromptSimple constructs a system prompt from chunk results without bookmark metadata
func (p *PromptBuilder) BuildSystemPromptSimple(chunks []model.ChunkResult) string {
	if len(chunks) == 0 {
		return fmt.Sprintf(systemPromptTemplate, "No relevant bookmarks found.")
	}

	var contextBuilder strings.Builder

	// Group chunks by bookmark ID
	bookmarkChunks := make(map[int][]model.ChunkResult)
	bookmarkOrder := []int{}

	for _, chunk := range chunks {
		if _, exists := bookmarkChunks[chunk.BookmarkID]; !exists {
			bookmarkOrder = append(bookmarkOrder, chunk.BookmarkID)
		}
		bookmarkChunks[chunk.BookmarkID] = append(bookmarkChunks[chunk.BookmarkID], chunk)
	}

	// Build context grouped by bookmark
	for i, bookmarkID := range bookmarkOrder {
		chunks := bookmarkChunks[bookmarkID]
		if len(chunks) == 0 {
			continue
		}

		if i > 0 {
			contextBuilder.WriteString("\n\n")
		}

		contextBuilder.WriteString(fmt.Sprintf("### Bookmark #%d\n\n", bookmarkID))

		// Write all chunks from this bookmark
		for j, chunk := range chunks {
			if j > 0 {
				contextBuilder.WriteString("\n...\n")
			}
			contextBuilder.WriteString(chunk.Content)
		}
	}

	return fmt.Sprintf(systemPromptTemplate, contextBuilder.String())
}

// ExtractSources extracts unique bookmark sources from chunks
func (p *PromptBuilder) ExtractSources(chunks []ChunkWithBookmark) []BookmarkSource {
	seen := make(map[int]bool)
	var sources []BookmarkSource

	for _, chunk := range chunks {
		if seen[chunk.Chunk.BookmarkID] {
			continue
		}
		seen[chunk.Chunk.BookmarkID] = true

		source := BookmarkSource{
			ID:      chunk.Chunk.BookmarkID,
			Excerpt: truncateText(chunk.Chunk.Content, 200),
		}

		if chunk.Bookmark != nil {
			source.Title = chunk.Bookmark.Title
			source.URL = chunk.Bookmark.URL
		}

		sources = append(sources, source)
	}

	return sources
}

// BookmarkSource represents a source bookmark for citation
type BookmarkSource struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Excerpt string `json:"excerpt"`
}

// truncateText truncates text to maxLen characters, adding ellipsis if needed
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Try to break at a word boundary
	truncated := text[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
