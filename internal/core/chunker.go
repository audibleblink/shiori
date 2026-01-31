package core

import (
	"strings"
	"unicode"
)

// TextChunker splits text into overlapping chunks suitable for embedding
type TextChunker struct {
	// ChunkSize is the target size in tokens (approximated as ~4 chars per token)
	ChunkSize int
	// ChunkOverlap is the number of tokens to overlap between chunks
	ChunkOverlap int
}

// NewTextChunker creates a new TextChunker with the given parameters
func NewTextChunker(chunkSize, chunkOverlap int) *TextChunker {
	if chunkSize < 100 {
		chunkSize = 100
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 4
	}

	return &TextChunker{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
	}
}

// ChunkText splits text into overlapping chunks
// Returns empty slice if text is empty or whitespace only
func (c *TextChunker) ChunkText(text string) []string {
	// Clean and normalize text
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Approximate tokens as ~4 characters
	charsPerToken := 4
	targetChars := c.ChunkSize * charsPerToken
	overlapChars := c.ChunkOverlap * charsPerToken

	// If text is shorter than target, return as single chunk
	if len(text) <= targetChars {
		return []string{text}
	}

	chunks := []string{}
	start := 0

	for start < len(text) {
		end := start + targetChars

		// Don't go past the end
		if end >= len(text) {
			chunks = append(chunks, strings.TrimSpace(text[start:]))
			break
		}

		// Try to find a good break point (sentence or paragraph boundary)
		chunk := text[start:end]
		breakPoint := findBestBreakPoint(chunk)

		if breakPoint > 0 {
			chunk = chunk[:breakPoint]
		}

		chunks = append(chunks, strings.TrimSpace(chunk))

		// Move start forward, accounting for overlap
		start += len(chunk) - overlapChars
		if start < 0 {
			start = 0
		}

		// Prevent infinite loop
		if len(chunk) == 0 {
			start++
		}
	}

	// Filter out empty chunks
	result := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk) != "" {
			result = append(result, chunk)
		}
	}

	return result
}

// findBestBreakPoint finds the best position to break text to preserve sentence boundaries
// Returns the position (from end of text) or 0 if no good break point found
func findBestBreakPoint(text string) int {
	// Priority order: paragraph break, sentence end, clause break, word break

	// Look for paragraph break (double newline)
	if idx := strings.LastIndex(text, "\n\n"); idx > len(text)/2 {
		return idx + 2 // Include the newlines
	}

	// Look for sentence ending punctuation followed by space or newline
	sentenceEnders := []string{". ", "! ", "? ", ".\n", "!\n", "?\n"}
	bestIdx := -1
	for _, ender := range sentenceEnders {
		if idx := strings.LastIndex(text, ender); idx > len(text)/2 && idx > bestIdx {
			bestIdx = idx + len(ender)
		}
	}
	if bestIdx > 0 {
		return bestIdx
	}

	// Look for clause breaks
	clauseBreaks := []string{"; ", ", ", ";\n", ",\n"}
	for _, b := range clauseBreaks {
		if idx := strings.LastIndex(text, b); idx > len(text)*2/3 && idx > bestIdx {
			bestIdx = idx + len(b)
		}
	}
	if bestIdx > 0 {
		return bestIdx
	}

	// Look for word boundary (space)
	if idx := strings.LastIndex(text, " "); idx > len(text)/2 {
		return idx + 1
	}

	// No good break point found
	return 0
}

// ChunkTextWithMetadata returns chunks with their position info
type ChunkWithPosition struct {
	Text       string
	StartIndex int
	EndIndex   int
	ChunkIndex int
}

// ChunkTextWithPositions splits text and returns position information
func (c *TextChunker) ChunkTextWithPositions(text string) []ChunkWithPosition {
	chunks := c.ChunkText(text)
	result := make([]ChunkWithPosition, len(chunks))

	pos := 0
	for i, chunk := range chunks {
		// Find where this chunk appears in the original text
		idx := strings.Index(text[pos:], chunk)
		if idx == -1 {
			// Chunk was trimmed, search from beginning
			idx = strings.Index(text, chunk)
			if idx == -1 {
				idx = pos // fallback
			}
		} else {
			idx += pos
		}

		result[i] = ChunkWithPosition{
			Text:       chunk,
			StartIndex: idx,
			EndIndex:   idx + len(chunk),
			ChunkIndex: i,
		}
		pos = idx
	}

	return result
}

// EstimateTokenCount provides a rough estimate of token count
// Using approximation of ~4 characters per token
func EstimateTokenCount(text string) int {
	// Count non-whitespace character runs
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	charCount := 0
	for _, r := range text {
		if !unicode.IsSpace(r) {
			charCount++
		}
	}

	// Rough approximation: ~4 chars per token
	return (charCount + 3) / 4
}

// PrepareTextForEmbedding cleans and normalizes text for embedding
func PrepareTextForEmbedding(text string) string {
	// Replace multiple whitespace with single space
	text = strings.Join(strings.Fields(text), " ")

	// Trim to reasonable length
	text = strings.TrimSpace(text)

	return text
}
