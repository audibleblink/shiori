package core

import (
	"testing"

	"github.com/go-shiori/shiori/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestPromptBuilder_BuildSystemPrompt(t *testing.T) {
	pb := NewPromptBuilder()

	t.Run("EmptyChunks", func(t *testing.T) {
		prompt := pb.BuildSystemPrompt(nil)
		assert.Contains(t, prompt, "No relevant bookmarks found")
		assert.Contains(t, prompt, "Use ONLY the provided context")
	})

	t.Run("SingleChunk", func(t *testing.T) {
		chunks := []ChunkWithBookmark{
			{
				Chunk: model.ChunkResult{
					BookmarkID: 1,
					ChunkIndex: 0,
					Content:    "This is the content about Go programming.",
					Similarity: 0.9,
				},
				Bookmark: &model.BookmarkDTO{
					ID:    1,
					Title: "Go Tutorial",
					URL:   "https://example.com/go",
				},
			},
		}

		prompt := pb.BuildSystemPrompt(chunks)
		assert.Contains(t, prompt, "### Go Tutorial")
		assert.Contains(t, prompt, "URL: https://example.com/go")
		assert.Contains(t, prompt, "This is the content about Go programming.")
	})

	t.Run("MultipleChunksFromSameBookmark", func(t *testing.T) {
		bookmark := &model.BookmarkDTO{
			ID:    1,
			Title: "Go Best Practices",
			URL:   "https://example.com/go-best",
		}

		chunks := []ChunkWithBookmark{
			{
				Chunk: model.ChunkResult{
					BookmarkID: 1,
					ChunkIndex: 0,
					Content:    "First chunk content.",
				},
				Bookmark: bookmark,
			},
			{
				Chunk: model.ChunkResult{
					BookmarkID: 1,
					ChunkIndex: 1,
					Content:    "Second chunk content.",
				},
				Bookmark: bookmark,
			},
		}

		prompt := pb.BuildSystemPrompt(chunks)

		// Should only have one header
		assert.Equal(t, 1, countOccurrences(prompt, "### Go Best Practices"))

		// Should have both chunks
		assert.Contains(t, prompt, "First chunk content.")
		assert.Contains(t, prompt, "Second chunk content.")

		// Should have separator between chunks
		assert.Contains(t, prompt, "...")
	})

	t.Run("ChunksFromDifferentBookmarks", func(t *testing.T) {
		chunks := []ChunkWithBookmark{
			{
				Chunk: model.ChunkResult{
					BookmarkID: 1,
					ChunkIndex: 0,
					Content:    "Go content here.",
				},
				Bookmark: &model.BookmarkDTO{
					ID:    1,
					Title: "Go Programming",
					URL:   "https://example.com/go",
				},
			},
			{
				Chunk: model.ChunkResult{
					BookmarkID: 2,
					ChunkIndex: 0,
					Content:    "Python content here.",
				},
				Bookmark: &model.BookmarkDTO{
					ID:    2,
					Title: "Python Guide",
					URL:   "https://example.com/python",
				},
			},
		}

		prompt := pb.BuildSystemPrompt(chunks)

		// Should have both bookmarks
		assert.Contains(t, prompt, "### Go Programming")
		assert.Contains(t, prompt, "### Python Guide")
		assert.Contains(t, prompt, "Go content here.")
		assert.Contains(t, prompt, "Python content here.")
	})

	t.Run("ChunkWithoutBookmarkMetadata", func(t *testing.T) {
		chunks := []ChunkWithBookmark{
			{
				Chunk: model.ChunkResult{
					BookmarkID: 42,
					ChunkIndex: 0,
					Content:    "Content without bookmark metadata.",
				},
				Bookmark: nil,
			},
		}

		prompt := pb.BuildSystemPrompt(chunks)
		assert.Contains(t, prompt, "### Bookmark #42")
		assert.Contains(t, prompt, "Content without bookmark metadata.")
	})
}

func TestPromptBuilder_BuildSystemPromptSimple(t *testing.T) {
	pb := NewPromptBuilder()

	t.Run("EmptyChunks", func(t *testing.T) {
		prompt := pb.BuildSystemPromptSimple(nil)
		assert.Contains(t, prompt, "No relevant bookmarks found")
	})

	t.Run("MultipleChunks", func(t *testing.T) {
		chunks := []model.ChunkResult{
			{BookmarkID: 1, ChunkIndex: 0, Content: "First content."},
			{BookmarkID: 1, ChunkIndex: 1, Content: "Second content."},
			{BookmarkID: 2, ChunkIndex: 0, Content: "Third content."},
		}

		prompt := pb.BuildSystemPromptSimple(chunks)
		assert.Contains(t, prompt, "### Bookmark #1")
		assert.Contains(t, prompt, "### Bookmark #2")
		assert.Contains(t, prompt, "First content.")
		assert.Contains(t, prompt, "Second content.")
		assert.Contains(t, prompt, "Third content.")
	})
}

func TestPromptBuilder_ExtractSources(t *testing.T) {
	pb := NewPromptBuilder()

	t.Run("EmptyChunks", func(t *testing.T) {
		sources := pb.ExtractSources(nil)
		assert.Empty(t, sources)
	})

	t.Run("UniqueBookmarks", func(t *testing.T) {
		chunks := []ChunkWithBookmark{
			{
				Chunk: model.ChunkResult{
					BookmarkID: 1,
					Content:    "Content from bookmark 1.",
				},
				Bookmark: &model.BookmarkDTO{
					ID:    1,
					Title: "First Bookmark",
					URL:   "https://example.com/1",
				},
			},
			{
				Chunk: model.ChunkResult{
					BookmarkID: 2,
					Content:    "Content from bookmark 2.",
				},
				Bookmark: &model.BookmarkDTO{
					ID:    2,
					Title: "Second Bookmark",
					URL:   "https://example.com/2",
				},
			},
		}

		sources := pb.ExtractSources(chunks)
		assert.Len(t, sources, 2)
		assert.Equal(t, 1, sources[0].ID)
		assert.Equal(t, "First Bookmark", sources[0].Title)
		assert.Equal(t, "https://example.com/1", sources[0].URL)
		assert.Equal(t, 2, sources[1].ID)
		assert.Equal(t, "Second Bookmark", sources[1].Title)
	})

	t.Run("DuplicateBookmarks", func(t *testing.T) {
		bookmark := &model.BookmarkDTO{
			ID:    1,
			Title: "Single Bookmark",
			URL:   "https://example.com/single",
		}

		chunks := []ChunkWithBookmark{
			{
				Chunk:    model.ChunkResult{BookmarkID: 1, Content: "First chunk."},
				Bookmark: bookmark,
			},
			{
				Chunk:    model.ChunkResult{BookmarkID: 1, Content: "Second chunk."},
				Bookmark: bookmark,
			},
			{
				Chunk:    model.ChunkResult{BookmarkID: 1, Content: "Third chunk."},
				Bookmark: bookmark,
			},
		}

		sources := pb.ExtractSources(chunks)
		assert.Len(t, sources, 1)
		assert.Equal(t, 1, sources[0].ID)
	})

	t.Run("ExcerptTruncation", func(t *testing.T) {
		longContent := "This is a very long content that should be truncated because it exceeds the maximum length allowed for excerpts. We want to make sure the truncation happens at a word boundary if possible, and that ellipsis is added at the end."

		chunks := []ChunkWithBookmark{
			{
				Chunk: model.ChunkResult{
					BookmarkID: 1,
					Content:    longContent,
				},
				Bookmark: &model.BookmarkDTO{
					ID:    1,
					Title: "Test",
					URL:   "https://example.com",
				},
			},
		}

		sources := pb.ExtractSources(chunks)
		assert.Len(t, sources, 1)
		assert.True(t, len(sources[0].Excerpt) <= 203) // 200 + "..."
		assert.True(t, len(sources[0].Excerpt) > 0)
	})
}

func TestTruncateText(t *testing.T) {
	t.Run("ShortText", func(t *testing.T) {
		text := "Short text"
		result := truncateText(text, 50)
		assert.Equal(t, text, result)
	})

	t.Run("ExactLength", func(t *testing.T) {
		text := "Exactly 10"
		result := truncateText(text, 10)
		assert.Equal(t, text, result)
	})

	t.Run("LongText", func(t *testing.T) {
		text := "This is a longer text that needs to be truncated"
		result := truncateText(text, 20)
		assert.True(t, len(result) <= 23) // 20 + "..."
		assert.True(t, result[len(result)-3:] == "...")
	})

	t.Run("BreakAtWordBoundary", func(t *testing.T) {
		text := "Word one two three four five"
		result := truncateText(text, 15)
		// Should break at word boundary before exceeding limit
		assert.True(t, len(result) <= 18)
		assert.Contains(t, result, "...")
	})
}

// Helper function to count occurrences of a substring
func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}
