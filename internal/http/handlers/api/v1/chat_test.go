package api_v1

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-shiori/shiori/internal/config"
	"github.com/go-shiori/shiori/internal/model"
	"github.com/go-shiori/shiori/internal/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestChatRequestValidation(t *testing.T) {
	t.Run("empty message", func(t *testing.T) {
		req := &chatRequest{Message: "", History: nil}
		err := req.IsValid()
		require.Error(t, err)
		require.Contains(t, err.Error(), "message is required")
	})

	t.Run("valid message", func(t *testing.T) {
		req := &chatRequest{Message: "What is Go?", History: nil}
		err := req.IsValid()
		require.NoError(t, err)
	})

	t.Run("valid message with history", func(t *testing.T) {
		req := &chatRequest{
			Message: "Tell me more",
			History: []model.ChatMessage{
				{Role: "user", Content: "What is Go?"},
				{Role: "assistant", Content: "Go is a programming language."},
			},
		}
		err := req.IsValid()
		require.NoError(t, err)
	})

	t.Run("invalid role in history", func(t *testing.T) {
		req := &chatRequest{
			Message: "Tell me more",
			History: []model.ChatMessage{
				{Role: "invalid", Content: "test"},
			},
		}
		err := req.IsValid()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid role")
	})

	t.Run("empty content in history", func(t *testing.T) {
		req := &chatRequest{
			Message: "Tell me more",
			History: []model.ChatMessage{
				{Role: "user", Content: ""},
			},
		}
		err := req.IsValid()
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty content")
	})
}

func TestHandleChatPost(t *testing.T) {
	logger := logrus.New()

	t.Run("requires authentication", func(t *testing.T) {
		ctx := context.Background()
		_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

		body := `{"message": "What is Go?"}`
		w := testutil.PerformRequest(deps, HandleChatPost, "POST", "/api/v1/chat", testutil.WithBody(body))

		// Should return 401 for unauthenticated requests
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("AI not enabled returns error", func(t *testing.T) {
		ctx := context.Background()
		cfg, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

		// Ensure AI is disabled
		cfg.AI = nil

		body := `{"message": "What is Go?"}`
		w := testutil.PerformRequest(deps, HandleChatPost, "POST", "/api/v1/chat",
			testutil.WithBody(body),
			testutil.WithFakeUser(),
		)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		require.Contains(t, w.Body.String(), "not enabled")
	})

	t.Run("AI enabled but database not supported", func(t *testing.T) {
		ctx := context.Background()
		cfg, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

		// Enable AI but SQLite database doesn't support it
		cfg.AI = &config.AIConfig{
			Enabled: true,
		}

		body := `{"message": "What is Go?"}`
		w := testutil.PerformRequest(deps, HandleChatPost, "POST", "/api/v1/chat",
			testutil.WithBody(body),
			testutil.WithFakeUser(),
		)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		require.Contains(t, w.Body.String(), "PostgreSQL")
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		ctx := context.Background()
		cfg, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

		// Enable AI (we'll hit DB check first since SQLite)
		cfg.AI = &config.AIConfig{
			Enabled: true,
		}

		body := `{invalid json`
		w := testutil.PerformRequest(deps, HandleChatPost, "POST", "/api/v1/chat",
			testutil.WithBody(body),
			testutil.WithFakeUser(),
		)

		// Will hit DB_NOT_SUPPORTED first since SQLite doesn't support AI
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("empty message validation error", func(t *testing.T) {
		ctx := context.Background()
		cfg, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

		// Enable AI
		cfg.AI = &config.AIConfig{
			Enabled: true,
		}

		body := `{"message": ""}`
		w := testutil.PerformRequest(deps, HandleChatPost, "POST", "/api/v1/chat",
			testutil.WithBody(body),
			testutil.WithFakeUser(),
		)

		// Will hit DB_NOT_SUPPORTED first since SQLite doesn't support AI
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func TestHandleChatStatus(t *testing.T) {
	logger := logrus.New()

	t.Run("requires authentication", func(t *testing.T) {
		ctx := context.Background()
		_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

		c, w := testutil.NewTestWebContext()
		HandleChatStatus(deps, c)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("AI not enabled", func(t *testing.T) {
		ctx := context.Background()
		cfg, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

		// Ensure AI is disabled
		cfg.AI = nil

		c, w := testutil.NewTestWebContext()
		testutil.SetFakeUser(c)
		HandleChatStatus(deps, c)

		require.Equal(t, http.StatusOK, w.Code)

		var resp model.ChatStatusResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		require.False(t, resp.Enabled)
		require.Contains(t, resp.Error, "not enabled")
	})

	t.Run("AI enabled but database not supported", func(t *testing.T) {
		ctx := context.Background()
		cfg, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

		// Enable AI but SQLite doesn't support it
		cfg.AI = &config.AIConfig{
			Enabled: true,
		}

		c, w := testutil.NewTestWebContext()
		testutil.SetFakeUser(c)
		HandleChatStatus(deps, c)

		require.Equal(t, http.StatusOK, w.Code)

		var resp model.ChatStatusResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		require.False(t, resp.Enabled)
		require.Contains(t, resp.Error, "PostgreSQL")
	})

	t.Run("AI enabled and chat domain not set", func(t *testing.T) {
		ctx := context.Background()
		cfg, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)

		// Enable AI
		cfg.AI = &config.AIConfig{
			Enabled: true,
		}

		// Note: Chat domain is not set in test dependencies by default

		c, w := testutil.NewTestWebContext()
		testutil.SetFakeUser(c)
		HandleChatStatus(deps, c)

		require.Equal(t, http.StatusOK, w.Code)

		var resp model.ChatStatusResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		// Should show not enabled since DB doesn't support AI
		require.False(t, resp.Enabled)
	})
}

func TestWriteSSEEvent(t *testing.T) {
	t.Run("writes chunk event", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeSSEEvent(w, "chunk", sseChunkData{Content: "Hello"})

		body := w.Body.String()
		require.Contains(t, body, "event: chunk")
		require.Contains(t, body, `"content":"Hello"`)
	})

	t.Run("writes sources event", func(t *testing.T) {
		w := httptest.NewRecorder()
		sources := []model.BookmarkSource{
			{ID: 1, Title: "Test", URL: "https://test.com", Excerpt: "Test excerpt"},
		}
		writeSSEEvent(w, "sources", sseSourcesData{Sources: sources})

		body := w.Body.String()
		require.Contains(t, body, "event: sources")
		require.Contains(t, body, `"id":1`)
		require.Contains(t, body, `"title":"Test"`)
	})

	t.Run("writes error event", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeSSEEvent(w, "error", sseErrorData{Error: "Something failed", Code: "TEST_ERROR"})

		body := w.Body.String()
		require.Contains(t, body, "event: error")
		require.Contains(t, body, `"error":"Something failed"`)
		require.Contains(t, body, `"code":"TEST_ERROR"`)
	})

	t.Run("writes done event", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeSSEEvent(w, "done", struct{}{})

		body := w.Body.String()
		require.Contains(t, body, "event: done")
	})
}

func TestSSEEventParsing(t *testing.T) {
	// Test that the SSE events we write can be properly parsed
	t.Run("parse chunk event", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeSSEEvent(w, "chunk", sseChunkData{Content: "Hello world"})

		scanner := bufio.NewScanner(w.Body)
		var eventType, data string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			}
		}

		require.Equal(t, "chunk", eventType)
		require.Contains(t, data, "Hello world")

		var chunk sseChunkData
		err := json.Unmarshal([]byte(data), &chunk)
		require.NoError(t, err)
		require.Equal(t, "Hello world", chunk.Content)
	})

	t.Run("parse sources event", func(t *testing.T) {
		w := httptest.NewRecorder()
		sources := []model.BookmarkSource{
			{ID: 1, Title: "Test Article", URL: "https://example.com", Excerpt: "Sample text"},
			{ID: 2, Title: "Another Article", URL: "https://example.org", Excerpt: "More text"},
		}
		writeSSEEvent(w, "sources", sseSourcesData{Sources: sources})

		scanner := bufio.NewScanner(w.Body)
		var eventType, data string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			}
		}

		require.Equal(t, "sources", eventType)

		var sourcesData sseSourcesData
		err := json.Unmarshal([]byte(data), &sourcesData)
		require.NoError(t, err)
		require.Len(t, sourcesData.Sources, 2)
		require.Equal(t, "Test Article", sourcesData.Sources[0].Title)
	})
}

// MockChatDomain provides a mock implementation for testing
type MockChatDomain struct {
	ChatFunc             func(ctx context.Context, message string, history []model.ChatMessage) (<-chan model.StreamChatEvent, error)
	ChatSyncFunc         func(ctx context.Context, message string, history []model.ChatMessage) (*model.ChatResponse, error)
	GetEmbeddingStatusFn func(ctx context.Context) (*model.EmbeddingStatusSummary, error)
	IsAvailableFn        func(ctx context.Context) error
}

func (m *MockChatDomain) Chat(ctx context.Context, message string, history []model.ChatMessage) (<-chan model.StreamChatEvent, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, message, history)
	}
	ch := make(chan model.StreamChatEvent)
	close(ch)
	return ch, nil
}

func (m *MockChatDomain) ChatSync(ctx context.Context, message string, history []model.ChatMessage) (*model.ChatResponse, error) {
	if m.ChatSyncFunc != nil {
		return m.ChatSyncFunc(ctx, message, history)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockChatDomain) GetEmbeddingStatus(ctx context.Context) (*model.EmbeddingStatusSummary, error) {
	if m.GetEmbeddingStatusFn != nil {
		return m.GetEmbeddingStatusFn(ctx)
	}
	return &model.EmbeddingStatusSummary{}, nil
}

func (m *MockChatDomain) IsAvailable(ctx context.Context) error {
	if m.IsAvailableFn != nil {
		return m.IsAvailableFn(ctx)
	}
	return nil
}

// Ensure MockChatDomain implements ChatDomain
var _ model.ChatDomain = (*MockChatDomain)(nil)
