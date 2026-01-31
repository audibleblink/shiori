package api_v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-shiori/shiori/internal/http/middleware"
	"github.com/go-shiori/shiori/internal/http/response"
	"github.com/go-shiori/shiori/internal/model"
)

// chatRequest represents the request body for the chat endpoint
type chatRequest struct {
	Message string              `json:"message"`
	History []model.ChatMessage `json:"history"`
}

func (r *chatRequest) IsValid() error {
	if r.Message == "" {
		return fmt.Errorf("message is required")
	}
	// Validate history entries if present
	for i, msg := range r.History {
		if msg.Role != "user" && msg.Role != "assistant" {
			return fmt.Errorf("invalid role in history[%d]: must be 'user' or 'assistant'", i)
		}
		if msg.Content == "" {
			return fmt.Errorf("empty content in history[%d]", i)
		}
	}
	return nil
}

// sseChunkData represents the data for a chunk event
type sseChunkData struct {
	Content string `json:"content"`
}

// sseSourcesData represents the data for a sources event
type sseSourcesData struct {
	Sources []model.BookmarkSource `json:"sources"`
}

// sseErrorData represents the data for an error event
type sseErrorData struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// HandleChatPost handles chat requests with SSE streaming response
//
//	@Summary					Chat with AI about bookmarks
//	@Description				Send a message and receive AI-generated answers about your bookmarked content.
//	@Description				Response is streamed using Server-Sent Events (SSE).
//	@Tags						Chat
//	@securityDefinitions.apikey	ApiKeyAuth
//	@Accept						json
//	@Produce					text/event-stream
//	@Param						payload	body		chatRequest	true	"Chat message and history"
//	@Success					200		{object}	nil			"SSE stream with chunk, sources, and done events"
//	@Failure					400		{object}	nil			"Invalid request payload"
//	@Failure					401		{object}	nil			"Authentication required"
//	@Failure					503		{object}	nil			"AI features not available"
//	@Router						/api/v1/chat [post]
func HandleChatPost(deps model.Dependencies, c model.WebContext) {
	// Require authentication
	if err := middleware.RequireLoggedInUser(deps, c); err != nil {
		return // Error already sent by middleware
	}

	// Check if AI features are enabled
	cfg := deps.Config()
	if cfg.AI == nil || !cfg.AI.Enabled {
		response.SendError(c, http.StatusServiceUnavailable, "AI features are not enabled. Please configure AI settings in your server configuration.")
		return
	}

	// Check if database supports AI
	if !deps.Database().SupportsAI() {
		response.SendError(c, http.StatusServiceUnavailable, "AI features require PostgreSQL with pgvector extension.")
		return
	}

	// Check if chat domain is available
	chatDomain := deps.Domains().Chat()
	if chatDomain == nil {
		response.SendError(c, http.StatusServiceUnavailable, "Chat service is not configured.")
		return
	}

	// Parse request
	var req chatRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.SendError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := req.IsValid(); err != nil {
		response.SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Set up SSE response headers
	w := c.ResponseWriter()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Flush headers
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Start chat streaming
	ctx := c.Request().Context()
	eventChan, err := chatDomain.Chat(ctx, req.Message, req.History)
	if err != nil {
		writeSSEEvent(w, "error", sseErrorData{Error: err.Error(), Code: "CHAT_ERROR"})
		return
	}

	// Stream events
	for event := range eventChan {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		default:
			switch event.Type {
			case "chunk":
				writeSSEEvent(w, "chunk", sseChunkData{Content: event.Content})
			case "sources":
				writeSSEEvent(w, "sources", sseSourcesData{Sources: event.Sources})
			case "done":
				writeSSEEvent(w, "done", struct{}{})
			case "error":
				errMsg := "An error occurred during chat"
				if event.Error != nil {
					errMsg = event.Error.Error()
				}
				writeSSEEvent(w, "error", sseErrorData{Error: errMsg, Code: "STREAM_ERROR"})
			}

			// Flush after each event
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// HandleChatStatus returns the status of AI chat features
//
//	@Summary					Get chat feature status
//	@Description				Returns whether AI chat is enabled and embedding statistics
//	@Tags						Chat
//	@securityDefinitions.apikey	ApiKeyAuth
//	@Produce					json
//	@Success					200	{object}	model.ChatStatusResponse
//	@Failure					401	{object}	nil	"Authentication required"
//	@Router						/api/v1/chat/status [get]
func HandleChatStatus(deps model.Dependencies, c model.WebContext) {
	// Require authentication
	if err := middleware.RequireLoggedInUser(deps, c); err != nil {
		return
	}

	ctx := c.Request().Context()
	cfg := deps.Config()

	// Check if AI is enabled
	aiEnabled := cfg.AI != nil && cfg.AI.Enabled

	// Check if database supports AI
	dbSupportsAI := deps.Database().SupportsAI()

	// Build response
	status := model.ChatStatusResponse{
		Enabled: aiEnabled && dbSupportsAI,
	}

	// If AI is not enabled, provide helpful error message
	if !aiEnabled {
		status.Error = "AI features are not enabled. Configure ai.enabled=true and provide Anthropic/Ollama settings."
		response.SendJSON(c, http.StatusOK, status)
		return
	}

	if !dbSupportsAI {
		status.Error = "AI features require PostgreSQL with pgvector extension."
		response.SendJSON(c, http.StatusOK, status)
		return
	}

	// Get chat domain for embedding status
	chatDomain := deps.Domains().Chat()
	if chatDomain == nil {
		status.Error = "Chat service is not configured."
		response.SendJSON(c, http.StatusOK, status)
		return
	}

	// Check service availability
	if err := chatDomain.IsAvailable(ctx); err != nil {
		status.Error = fmt.Sprintf("Chat service unavailable: %s", err.Error())
		response.SendJSON(c, http.StatusOK, status)
		return
	}

	// Get embedding status
	embeddingStatus, err := chatDomain.GetEmbeddingStatus(ctx)
	if err != nil {
		deps.Logger().WithError(err).Error("failed to get embedding status")
		// Still return enabled status, just without embedding stats
		response.SendJSON(c, http.StatusOK, status)
		return
	}

	status.EmbeddingStatus = embeddingStatus
	response.SendJSON(c, http.StatusOK, status)
}

// writeSSEEvent writes a Server-Sent Event to the response writer
func writeSSEEvent(w http.ResponseWriter, eventType string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		// Fallback to error event if marshal fails
		fmt.Fprintf(w, "event: error\ndata: {\"error\":\"Failed to serialize response\"}\n\n")
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, jsonData)
}
