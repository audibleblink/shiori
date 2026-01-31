package domains

import (
	"context"
	"fmt"

	"github.com/go-shiori/shiori/internal/core"
	"github.com/go-shiori/shiori/internal/model"
)

// ChatDomainImpl handles AI-powered chat over bookmark content
type ChatDomainImpl struct {
	deps             model.Dependencies
	anthropicClient  *core.AnthropicClient
	embeddingService model.EmbeddingDomain
	promptBuilder    *core.PromptBuilder
	config           ChatDomainConfig
}

// ChatDomainConfig holds configuration for the chat domain
type ChatDomainConfig struct {
	SimilarityThreshold float64
	MaxChunks           int
}

// Ensure ChatDomainImpl implements model.ChatDomain
var _ model.ChatDomain = (*ChatDomainImpl)(nil)

// NewChatDomain creates a new ChatDomain
func NewChatDomain(
	deps model.Dependencies,
	anthropicClient *core.AnthropicClient,
	embeddingService model.EmbeddingDomain,
	config ChatDomainConfig,
) *ChatDomainImpl {
	return &ChatDomainImpl{
		deps:             deps,
		anthropicClient:  anthropicClient,
		embeddingService: embeddingService,
		promptBuilder:    core.NewPromptBuilder(),
		config:           config,
	}
}

// Chat performs Q&A over user's bookmarks and returns a streaming response
func (d *ChatDomainImpl) Chat(ctx context.Context, message string, history []model.ChatMessage) (<-chan model.StreamChatEvent, error) {
	db := d.deps.Database()

	// Check if database supports AI features
	if !db.SupportsAI() {
		return nil, fmt.Errorf("AI features require PostgreSQL with pgvector extension")
	}

	// Search for relevant chunks
	chunks, err := d.embeddingService.SearchSimilar(ctx, message, d.config.SimilarityThreshold, d.config.MaxChunks)
	if err != nil {
		return nil, fmt.Errorf("failed to search relevant content: %w", err)
	}

	// Enrich chunks with bookmark metadata
	enrichedChunks, err := d.enrichChunksWithBookmarks(ctx, chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich chunks: %w", err)
	}

	// Build system prompt with retrieved context
	systemPrompt := d.promptBuilder.BuildSystemPrompt(enrichedChunks)

	// Extract sources for later
	sources := d.convertSources(d.promptBuilder.ExtractSources(enrichedChunks))

	// Build messages for Anthropic API
	anthropicMessages := d.buildAnthropicMessages(message, history)

	// Stream the response
	anthropicStream, err := d.anthropicClient.StreamChat(ctx, anthropicMessages, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to start chat stream: %w", err)
	}

	// Create output channel
	eventChan := make(chan model.StreamChatEvent, 100)

	go func() {
		defer close(eventChan)

		for event := range anthropicStream {
			switch event.Type {
			case "chunk":
				eventChan <- model.StreamChatEvent{
					Type:    "chunk",
					Content: event.Content,
				}
			case "done":
				// Send sources before done
				eventChan <- model.StreamChatEvent{
					Type:    "sources",
					Sources: sources,
				}
				eventChan <- model.StreamChatEvent{
					Type: "done",
				}
			case "error":
				eventChan <- model.StreamChatEvent{
					Type:  "error",
					Error: event.Error,
				}
			}
		}
	}()

	return eventChan, nil
}

// ChatSync performs Q&A over user's bookmarks and returns a complete response
func (d *ChatDomainImpl) ChatSync(ctx context.Context, message string, history []model.ChatMessage) (*model.ChatResponse, error) {
	db := d.deps.Database()

	// Check if database supports AI features
	if !db.SupportsAI() {
		return nil, fmt.Errorf("AI features require PostgreSQL with pgvector extension")
	}

	// Search for relevant chunks
	chunks, err := d.embeddingService.SearchSimilar(ctx, message, d.config.SimilarityThreshold, d.config.MaxChunks)
	if err != nil {
		return nil, fmt.Errorf("failed to search relevant content: %w", err)
	}

	// Enrich chunks with bookmark metadata
	enrichedChunks, err := d.enrichChunksWithBookmarks(ctx, chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich chunks: %w", err)
	}

	// Build system prompt with retrieved context
	systemPrompt := d.promptBuilder.BuildSystemPrompt(enrichedChunks)

	// Extract sources
	sources := d.convertSources(d.promptBuilder.ExtractSources(enrichedChunks))

	// Build messages for Anthropic API
	anthropicMessages := d.buildAnthropicMessages(message, history)

	// Get the response
	response, err := d.anthropicClient.Chat(ctx, anthropicMessages, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat response: %w", err)
	}

	return &model.ChatResponse{
		Message: response,
		Sources: sources,
	}, nil
}

// GetEmbeddingStatus returns the embedding status summary
func (d *ChatDomainImpl) GetEmbeddingStatus(ctx context.Context) (*model.EmbeddingStatusSummary, error) {
	return d.embeddingService.GetEmbeddingStatus(ctx)
}

// IsAvailable checks if the chat service is available
func (d *ChatDomainImpl) IsAvailable(ctx context.Context) error {
	// Check Anthropic availability
	if err := d.anthropicClient.IsAvailable(ctx); err != nil {
		return fmt.Errorf("Anthropic API not available: %w", err)
	}

	// Check Ollama availability (through embedding service)
	if err := d.embeddingService.IsOllamaAvailable(ctx); err != nil {
		return fmt.Errorf("Ollama not available: %w", err)
	}

	return nil
}

// enrichChunksWithBookmarks fetches bookmark metadata for chunks
func (d *ChatDomainImpl) enrichChunksWithBookmarks(ctx context.Context, chunks []model.ChunkResult) ([]core.ChunkWithBookmark, error) {
	db := d.deps.Database()

	if len(chunks) == 0 {
		return nil, nil
	}

	// Collect unique bookmark IDs
	bookmarkIDs := make(map[int]bool)
	for _, chunk := range chunks {
		bookmarkIDs[chunk.BookmarkID] = true
	}

	// Convert to slice
	ids := make([]int, 0, len(bookmarkIDs))
	for id := range bookmarkIDs {
		ids = append(ids, id)
	}

	// Fetch bookmarks
	bookmarks, err := db.GetBookmarks(ctx, model.DBGetBookmarksOptions{
		IDs: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmarks: %w", err)
	}

	// Create lookup map
	bookmarkMap := make(map[int]*model.BookmarkDTO)
	for i := range bookmarks {
		bookmarkMap[bookmarks[i].ID] = &bookmarks[i]
	}

	// Enrich chunks
	enriched := make([]core.ChunkWithBookmark, len(chunks))
	for i, chunk := range chunks {
		enriched[i] = core.ChunkWithBookmark{
			Chunk:    chunk,
			Bookmark: bookmarkMap[chunk.BookmarkID],
		}
	}

	return enriched, nil
}

// buildAnthropicMessages converts chat messages to Anthropic format
func (d *ChatDomainImpl) buildAnthropicMessages(message string, history []model.ChatMessage) []core.ChatMessage {
	messages := make([]core.ChatMessage, 0, len(history)+1)

	// Add history
	for _, msg := range history {
		messages = append(messages, core.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current message
	messages = append(messages, core.ChatMessage{
		Role:    "user",
		Content: message,
	})

	return messages
}

// convertSources converts core.BookmarkSource to model.BookmarkSource
func (d *ChatDomainImpl) convertSources(sources []core.BookmarkSource) []model.BookmarkSource {
	result := make([]model.BookmarkSource, len(sources))
	for i, s := range sources {
		result[i] = model.BookmarkSource{
			ID:      s.ID,
			Title:   s.Title,
			URL:     s.URL,
			Excerpt: s.Excerpt,
		}
	}
	return result
}
