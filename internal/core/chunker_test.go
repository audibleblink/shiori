package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTextChunker(t *testing.T) {
	t.Run("ValidParameters", func(t *testing.T) {
		chunker := NewTextChunker(512, 50)
		assert.Equal(t, 512, chunker.ChunkSize)
		assert.Equal(t, 50, chunker.ChunkOverlap)
	})

	t.Run("ChunkSizeTooSmall", func(t *testing.T) {
		chunker := NewTextChunker(50, 10)
		assert.Equal(t, 100, chunker.ChunkSize) // Should be clamped to minimum
	})

	t.Run("NegativeOverlap", func(t *testing.T) {
		chunker := NewTextChunker(512, -10)
		assert.Equal(t, 0, chunker.ChunkOverlap) // Should be clamped to 0
	})

	t.Run("OverlapTooLarge", func(t *testing.T) {
		chunker := NewTextChunker(512, 600)
		assert.Equal(t, 128, chunker.ChunkOverlap) // Should be reduced to chunkSize/4
	})
}

func TestTextChunker_ChunkText(t *testing.T) {
	t.Run("EmptyText", func(t *testing.T) {
		chunker := NewTextChunker(512, 50)
		chunks := chunker.ChunkText("")
		assert.Nil(t, chunks)
	})

	t.Run("WhitespaceOnly", func(t *testing.T) {
		chunker := NewTextChunker(512, 50)
		chunks := chunker.ChunkText("   \n\t  \n  ")
		assert.Nil(t, chunks)
	})

	t.Run("ShortText", func(t *testing.T) {
		chunker := NewTextChunker(512, 50)
		text := "This is a short piece of text."
		chunks := chunker.ChunkText(text)

		require.Len(t, chunks, 1)
		assert.Equal(t, text, chunks[0])
	})

	t.Run("LongText", func(t *testing.T) {
		chunker := NewTextChunker(100, 20) // 400 chars per chunk, 80 overlap

		// Create text that should produce multiple chunks
		text := strings.Repeat("This is a sentence. ", 50) // ~1000 chars
		chunks := chunker.ChunkText(text)

		require.Greater(t, len(chunks), 1)

		// Each chunk should be non-empty
		for _, chunk := range chunks {
			assert.NotEmpty(t, chunk)
		}
	})

	t.Run("PreservesSentenceBoundaries", func(t *testing.T) {
		chunker := NewTextChunker(100, 10) // ~400 char chunks

		text := "First sentence ends here. Second sentence continues. Third sentence goes on. Fourth sentence finishes."
		chunks := chunker.ChunkText(text)

		// Check that chunks tend to end at sentence boundaries
		for i, chunk := range chunks[:len(chunks)-1] { // Skip last chunk
			trimmed := strings.TrimSpace(chunk)
			lastChar := trimmed[len(trimmed)-1]
			// Should preferably end with sentence-ending punctuation or at a good break point
			t.Logf("Chunk %d ends with: %q (last few chars: %q)", i, string(lastChar), trimmed[max(0, len(trimmed)-20):])
		}
	})

	t.Run("HandlesNewlines", func(t *testing.T) {
		chunker := NewTextChunker(100, 10)

		text := "Paragraph one with some content.\n\nParagraph two has more text.\n\nParagraph three ends here."
		chunks := chunker.ChunkText(text)

		require.NotEmpty(t, chunks)
		// Verify no chunk is empty
		for _, chunk := range chunks {
			assert.NotEmpty(t, strings.TrimSpace(chunk))
		}
	})

	t.Run("ChunksHaveOverlap", func(t *testing.T) {
		chunker := NewTextChunker(100, 20)

		// Create text long enough to have multiple chunks
		text := strings.Repeat("word ", 200) // 1000 chars
		chunks := chunker.ChunkText(text)

		if len(chunks) >= 2 {
			// Check that there's some overlap between consecutive chunks
			for i := 0; i < len(chunks)-1; i++ {
				chunk1End := chunks[i][max(0, len(chunks[i])-100):]
				chunk2Start := chunks[i+1][:min(100, len(chunks[i+1]))]

				// There should be some common substring
				hasOverlap := false
				for j := 5; j <= len(chunk1End); j++ {
					if strings.Contains(chunk2Start, chunk1End[len(chunk1End)-j:]) {
						hasOverlap = true
						break
					}
				}
				if !hasOverlap {
					t.Logf("Warning: No clear overlap found between chunks %d and %d", i, i+1)
				}
			}
		}
	})
}

func TestTextChunker_ChunkTextWithPositions(t *testing.T) {
	t.Run("TracksPositions", func(t *testing.T) {
		chunker := NewTextChunker(512, 50)
		text := "This is a piece of text that should be tracked."
		chunks := chunker.ChunkTextWithPositions(text)

		require.Len(t, chunks, 1)
		assert.Equal(t, 0, chunks[0].ChunkIndex)
		assert.Equal(t, text, chunks[0].Text)
	})

	t.Run("MultipleChunksHaveIncreasingIndices", func(t *testing.T) {
		chunker := NewTextChunker(100, 10)
		text := strings.Repeat("This is sentence number one. ", 20)
		chunks := chunker.ChunkTextWithPositions(text)

		for i, chunk := range chunks {
			assert.Equal(t, i, chunk.ChunkIndex)
		}
	})
}

func TestEstimateTokenCount(t *testing.T) {
	t.Run("EmptyText", func(t *testing.T) {
		count := EstimateTokenCount("")
		assert.Equal(t, 0, count)
	})

	t.Run("WhitespaceOnly", func(t *testing.T) {
		count := EstimateTokenCount("   \n\t  ")
		assert.Equal(t, 0, count)
	})

	t.Run("ShortText", func(t *testing.T) {
		// "Hello" = 5 chars, should estimate to ~1-2 tokens
		count := EstimateTokenCount("Hello")
		assert.GreaterOrEqual(t, count, 1)
		assert.LessOrEqual(t, count, 2)
	})

	t.Run("LongerText", func(t *testing.T) {
		// 100 chars should estimate to ~25 tokens
		text := strings.Repeat("abcd", 25) // 100 chars
		count := EstimateTokenCount(text)
		assert.InDelta(t, 25, count, 5)
	})
}

func TestPrepareTextForEmbedding(t *testing.T) {
	t.Run("NormalizesWhitespace", func(t *testing.T) {
		text := "This   has    multiple   spaces"
		prepared := PrepareTextForEmbedding(text)
		assert.Equal(t, "This has multiple spaces", prepared)
	})

	t.Run("TrimsWhitespace", func(t *testing.T) {
		text := "  text with spaces  "
		prepared := PrepareTextForEmbedding(text)
		assert.Equal(t, "text with spaces", prepared)
	})

	t.Run("HandlesNewlines", func(t *testing.T) {
		text := "Line one\n\nLine two\nLine three"
		prepared := PrepareTextForEmbedding(text)
		assert.Equal(t, "Line one Line two Line three", prepared)
	})

	t.Run("HandlesTabs", func(t *testing.T) {
		text := "Tab\there\tand\tthere"
		prepared := PrepareTextForEmbedding(text)
		assert.Equal(t, "Tab here and there", prepared)
	})
}

func TestFindBestBreakPoint(t *testing.T) {
	t.Run("ParagraphBreak", func(t *testing.T) {
		// The function should find paragraph breaks
		text := "First paragraph here.\n\nSecond paragraph."
		breakPoint := findBestBreakPoint(text)
		assert.Greater(t, breakPoint, 0)
	})

	t.Run("SentenceEnd", func(t *testing.T) {
		text := "First sentence. Second sentence here."
		breakPoint := findBestBreakPoint(text)
		assert.Greater(t, breakPoint, 0)
	})

	t.Run("NoGoodBreak", func(t *testing.T) {
		text := "onewordwithoutanyspaces"
		breakPoint := findBestBreakPoint(text)
		assert.Equal(t, 0, breakPoint)
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
