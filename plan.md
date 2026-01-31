# Execution Plan: Bookmark Q&A Chat Interface

## Overview

This plan implements an AI-powered chat interface for Shiori that enables natural language Q&A over bookmarked content. The implementation is divided into 6 phases, each building on the previous to deliver incremental, testable functionality.

**Key Constraints:**
- PostgreSQL with pgvector required (no SQLite/MySQL support)
- Ollama for local embeddings (self-hosted, no API costs)
- Anthropic Claude for chat responses
- Must not break existing bookmark workflows

---

## Phase 0: Prerequisites & Setup

**Goal:** Establish development environment and verify external dependencies

**Dependencies:** None

### Tasks

- [ ] Verify PostgreSQL 14+ available for development
- [ ] Install and verify pgvector extension
- [ ] Install Ollama locally
- [ ] Pull `embeddinggemma` model via Ollama
- [ ] Obtain Anthropic API key for testing
- [ ] Create test configuration file with AI settings
- [ ] Document local setup in `docs/development/ai-setup.md`

### Completion Criteria

**Test Script:** `scripts/verify-ai-prerequisites.sh`

```bash
#!/bin/bash
# Check PostgreSQL version
psql --version | grep -E "1[4-9]\.|[2-9][0-9]\."

# Check pgvector extension
psql -c "CREATE EXTENSION IF NOT EXISTS vector;" $SHIORI_TEST_PG_URL

# Check Ollama
curl -s http://localhost:11434/api/tags | grep embeddinggemma

# Check Anthropic key
[ -n "$SHIORI_ANTHROPIC_API_KEY" ]

echo "✓ All prerequisites met"
```

**Exit Criteria:**
- Script exits with code 0
- Can connect to PostgreSQL with pgvector
- Ollama returns embedding for test input
- Anthropic API key is valid (via test call)

---

## Phase 1: Configuration & Database Schema

**Goal:** Add configuration support and database schema for embeddings

**Dependencies:** Phase 0

### Tasks

- [x] Add AI configuration struct to `internal/config/`
  - `AIConfig` with nested `AnthropicConfig`, `OllamaConfig`, `RetrievalConfig`, `EmbeddingConfig`
  - Environment variable mapping
  - Validation logic
- [x] Create database migration `internal/database/migrations/postgres/0003_ai_features.up.sql`
  - `CREATE EXTENSION vector`
  - `bookmark_chunks` table with vector(768) column
  - IVFFlat index on embedding column
  - Add columns to `bookmark`: `embedding_status`, `embedding_error`, `embedded_at`
- [x] Update `internal/model/ai.go` with new embedding-related types
- [x] Add database methods to `internal/model/database.go` interface:
  - `CreateBookmarkChunks(ctx, chunks []BookmarkChunk) error`
  - `GetBookmarkChunks(ctx, bookmarkID int) ([]BookmarkChunk, error)`
  - `DeleteBookmarkChunks(ctx, bookmarkID int) error`
  - `UpdateBookmarkEmbeddingStatus(ctx, id int, status EmbeddingStatus, error *string) error`
  - `SearchSimilarChunks(ctx, embedding []float32, threshold float64, limit int) ([]ChunkResult, error)`
  - `GetBookmarksByEmbeddingStatus(ctx, status EmbeddingStatus, limit int) ([]BookmarkDTO, error)`
  - `GetEmbeddingStatusSummary(ctx) (*EmbeddingStatusSummary, error)`
  - `SupportsAI() bool`
- [x] Implement database methods for PostgreSQL only
- [x] Add graceful handling for SQLite/MySQL (return `ErrAINotSupported`)

### Completion Criteria

**Unit Tests:**

```go
// internal/config/ai_config_test.go
func TestAIConfigValidation(t *testing.T) {
    // Valid config passes
    // Missing API key fails
    // Invalid threshold values fail
}

// internal/database/ai_postgres_test.go
func TestBookmarkChunksOperations(t *testing.T) {
    // Create chunk with 768-dim embedding
    // Retrieve chunks by bookmark ID
    // Delete cascades on bookmark delete
    // Vector search returns similar chunks
}

func TestEmbeddingStatusTracking(t *testing.T) {
    // Update status through lifecycle: pending -> processing -> completed
    // Update status with error
    // Query bookmarks by status
}
```

**Migration Test:**

```bash
# Run migration on clean PostgreSQL database
go test -tags test_postgres_only ./internal/database -run TestMigration0009

# Verify schema
psql $SHIORI_TEST_PG_URL -c "\d bookmark_chunks"
psql $SHIORI_TEST_PG_URL -c "\d bookmark" | grep embedding_status
```

**Exit Criteria:**
- ✅ All configuration unit tests pass
- ✅ AI config implementation complete with validation
- ✅ Database migration created
- ✅ Database methods implemented for PostgreSQL
- ✅ SQLite/MySQL gracefully return `ErrNotSupported` for AI methods
- ⏸️ PostgreSQL tests pending (requires pgvector installation)
  - Note: pgvector not available on remote server - see PGVECTOR_SETUP_REQUIRED.md
  - Tests written and ready to run once pgvector is installed

---

## Phase 2: Embedding Service (Ollama Integration)

**Goal:** Implement embedding generation and vector search

**Dependencies:** Phase 1

### Tasks

- [ ] Create Ollama client in `internal/core/ollama_client.go`
  - `GenerateEmbedding(ctx context.Context, text string) ([]float32, error)`
  - HTTP client with configurable endpoint
  - Error handling and retries
- [ ] Create text chunking utility in `internal/core/chunker.go`
  - `ChunkText(text string, chunkSize int, overlap int) []string`
  - Token-aware chunking (use character approximation: ~4 chars/token)
  - Preserve sentence boundaries where possible
- [ ] Implement `EmbeddingService` in `internal/domains/embedding.go`
  - `EmbedBookmark(ctx, bookmarkID)` - extract content, chunk, embed, store
  - `SearchSimilar(ctx, userID, queryText, threshold, limit)` - embed query, vector search
  - Status tracking (pending -> processing -> completed/failed)
- [ ] Add embedding service to Dependencies struct

### Completion Criteria

**Unit Tests:**

```go
// internal/core/ollama_client_test.go
func TestOllamaClient(t *testing.T) {
    // Generate embedding for sample text
    // Verify 768-dimensional output
    // Handle Ollama unavailable gracefully
    // Retry on transient failures
}

// internal/core/chunker_test.go
func TestChunkText(t *testing.T) {
    t.Run("RespectChunkSize", ...)
    t.Run("OverlapCorrectly", ...)
    t.Run("PreserveSentences", ...)
    t.Run("HandleShortText", ...)
}

// internal/domains/embedding_test.go
func TestEmbedBookmark(t *testing.T) {
    t.Run("Success", ...) // Chunks stored, status=completed
    t.Run("NoContent", ...) // Status=skipped
    t.Run("OllamaFailure", ...) // Status=failed, error recorded
}

func TestSearchSimilar(t *testing.T) {
    // Setup: Embed 3 bookmarks about different topics
    // Query about topic A
    // Verify top results are from topic A
    // Verify similarity scores above threshold
}
```

**Integration Test Script:** `scripts/test-embedding-e2e.sh`

```bash
#!/bin/bash
# Create test bookmark with known content
BOOKMARK_ID=$(psql -t $SHIORI_TEST_PG_URL -c "INSERT INTO bookmark (...) VALUES (...) RETURNING id")

# Run embedding service
go run ./cmd/test-embed $BOOKMARK_ID

# Verify chunks created
CHUNK_COUNT=$(psql -t $SHIORI_TEST_PG_URL -c "SELECT COUNT(*) FROM bookmark_chunks WHERE bookmark_id=$BOOKMARK_ID")
[ $CHUNK_COUNT -gt 0 ]

# Test search
go run ./cmd/test-search "database optimization"

echo "✓ Embedding pipeline works end-to-end"
```

**Exit Criteria:**
- Ollama client successfully generates 768-dim embeddings
- Chunking produces expected number of chunks with correct overlap
- `EmbedBookmark` stores chunks in database with embeddings
- `SearchSimilar` returns relevant results ranked by similarity
- Failed embeddings update status with error message
- All unit tests pass
- Integration test succeeds

---

## Phase 3: Chat Domain Service (Anthropic Integration)

**Goal:** Implement RAG-based chat with streaming responses

**Dependencies:** Phase 2

### Tasks

- [ ] Create Anthropic client in `internal/core/anthropic_client.go`
  - `StreamChat(ctx, messages, systemPrompt) (<-chan StreamEvent, error)`
  - Server-Sent Events support
  - Handle streaming chunks, errors, completion
- [ ] Implement `ChatDomain` in `internal/domains/chat.go`
  - `Chat(ctx, userID, message, history)` - orchestrates RAG pipeline:
    1. Embed user's query via EmbeddingService
    2. Search similar chunks (threshold=0.7, limit=20)
    3. Build system prompt with retrieved context
    4. Stream chat response via Anthropic
    5. Extract source citations from chunks
  - `GetEmbeddingStatus(ctx, userID)` - aggregate stats by user
- [ ] Create RAG prompt builder in `internal/core/prompt_builder.go`
  - `BuildSystemPrompt(chunks []ChunkResult) string`
  - Format chunks with bookmark metadata
  - Implement template from PRD Appendix A
- [ ] Add chat domain to Dependencies struct

### Completion Criteria

**Unit Tests:**

```go
// internal/core/anthropic_client_test.go
func TestAnthropicStreaming(t *testing.T) {
    t.Run("StreamSuccess", ...) // Receive chunks, final event
    t.Run("StreamError", ...) // Handle API errors
    t.Run("ContextTooLong", ...) // Handle token limit errors
}

// internal/core/prompt_builder_test.go
func TestBuildSystemPrompt(t *testing.T) {
    // Include all chunk content
    // Include bookmark titles/URLs
    // Format as markdown
    // Handle empty chunks
}

// internal/domains/chat_test.go
func TestChat(t *testing.T) {
    t.Run("RelevantAnswer", ...) // Answer includes retrieved context
    t.Run("NoRelevantChunks", ...) // "I don't have information about..."
    t.Run("MultiTurnConversation", ...) // History included in API call
    t.Run("SourceAttribution", ...) // Returns correct bookmark sources
}

func TestGetEmbeddingStatus(t *testing.T) {
    // Count bookmarks by status for user
    // Filter by user_id correctly
}
```

**Mock-Free Integration Test:** `scripts/test-chat-e2e.sh`

```bash
#!/bin/bash
# Setup: Create 2 bookmarks with known content
# Bookmark 1: About "Go error handling"
# Bookmark 2: About "Python asyncio"

# Embed both bookmarks
go run ./cmd/test-embed-multiple 1 2

# Test chat query
RESPONSE=$(go run ./cmd/test-chat "How do I handle errors in Go?")

# Verify response mentions bookmark 1 content
echo $RESPONSE | grep -i "error"

# Verify sources include bookmark 1
echo $RESPONSE | jq '.sources[] | select(.id==1)'

echo "✓ Chat RAG pipeline works end-to-end"
```

**Exit Criteria:**
- Anthropic client streams responses correctly
- System prompt includes retrieved chunk context
- Chat responses are grounded in user's bookmarks
- Source citations include correct bookmark IDs and excerpts
- Multi-turn conversations maintain context
- Empty retrieval results in "no information" response
- All unit tests pass
- Integration test produces relevant answer

---

## Phase 4: API Endpoints

**Goal:** Expose chat functionality via HTTP API

**Dependencies:** Phase 3

### Tasks

- [ ] Create POST `/api/v1/chat` handler in `internal/http/handlers/api/v1/chat.go`
  - Parse request: `{message, history[]}`
  - Validate authentication (require logged-in user)
  - Call ChatDomain.Chat()
  - Stream SSE response:
    - `event: chunk` for text chunks
    - `event: sources` for bookmark citations
    - `event: done` on completion
    - `event: error` on failures
  - Handle graceful degradation (AI not configured)
- [ ] Create GET `/api/v1/chat/status` handler
  - Check if AI is enabled in config
  - Return embedding statistics via ChatDomain.GetEmbeddingStatus()
  - Include configuration hints if disabled
- [ ] Register routes in `internal/http/http.go`
- [ ] Add Swagger annotations
- [ ] Update API documentation

### Completion Criteria

**Unit Tests:**

```go
// internal/http/handlers/api/v1/chat_test.go
func TestHandleChatPost(t *testing.T) {
    t.Run("Success", ...) // SSE stream with chunks + sources
    t.Run("Unauthorized", ...) // 401 without auth
    t.Run("AIDisabled", ...) // Error with helpful message
    t.Run("InvalidRequest", ...) // 400 on malformed JSON
    t.Run("StreamingError", ...) // Handle mid-stream failures
}

func TestHandleChatStatus(t *testing.T) {
    t.Run("Enabled", ...) // Returns stats
    t.Run("Disabled", ...) // Returns disabled state
}
```

**Integration Test (cURL):** `scripts/test-api-chat.sh`

```bash
#!/bin/bash
# Get session token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -d '{"username":"test","password":"test"}' | jq -r .token)

# Test chat endpoint
curl -N -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"message":"What is Go?","history":[]}' \
    http://localhost:8080/api/v1/chat

# Verify SSE events received:
# - event: chunk
# - event: sources
# - event: done

# Test status endpoint
curl -H "Authorization: Bearer $TOKEN" \
    http://localhost:8080/api/v1/chat/status | jq '.enabled'

echo "✓ Chat API endpoints work"
```

**Exit Criteria:**
- POST /api/v1/chat streams SSE events correctly
- SSE includes text chunks, sources, and done event
- Authentication enforced (401 without token)
- AI disabled state returns helpful error
- GET /api/v1/chat/status returns accurate statistics
- Swagger docs updated and valid
- All unit tests pass
- cURL integration test succeeds

---

## Phase 5: Embedding Lifecycle Automation

**Goal:** Automatically embed bookmarks on save and backfill existing

**Dependencies:** Phase 2 (does not depend on Phase 3/4)

### Tasks

- [ ] Create background worker in `internal/domains/embedding_worker.go`
  - Job queue for async embedding
  - Configurable concurrency
  - Exponential backoff on failures
  - Graceful shutdown
- [ ] Hook embedding into bookmark save flow
  - Update `internal/domains/bookmarks.go`:
    - After bookmark save, enqueue embedding job
    - Set `embedding_status = 'pending'`
    - Job runs asynchronously (don't block response)
- [ ] Implement backfill job (already in embedding.go BackfillEmbeddings)
  - Query bookmarks with `embedding_status = 'pending'`
  - Process in batches (configurable size)
  - Delay between batches to avoid overwhelming Ollama
  - Idempotent (safe to restart)
  - Log progress: "Backfill: 45/150 complete"
- [ ] Start backfill on server startup (if enabled in config)
- [ ] Add embedding re-generation on bookmark content update
  - Detect content changes
  - Delete old chunks
  - Re-enqueue for embedding

### Completion Criteria

**Unit Tests:**

```go
// internal/core/embedding_worker_test.go
func TestEmbeddingWorker(t *testing.T) {
    t.Run("ProcessJob", ...) // Job completes, status updated
    t.Run("RetryOnFailure", ...) // Exponential backoff
    t.Run("GracefulShutdown", ...) // Drains queue on shutdown
}

// internal/core/embedding_backfill_test.go
func TestBackfillJob(t *testing.T) {
    t.Run("ProcessPendingBookmarks", ...)
    t.Run("RespectBatchSize", ...)
    t.Run("SkipCompleted", ...) // Idempotent
}
```

**End-to-End Test:** `scripts/test-embedding-lifecycle.sh`

```bash
#!/bin/bash
# Start server with backfill enabled
go run main.go serve &
SERVER_PID=$!

# Create bookmark via API
BOOKMARK_ID=$(curl -X POST -H "Authorization: Bearer $TOKEN" \
    http://localhost:8080/api/v1/bookmarks \
    -d '{"url":"https://example.com/article"}' | jq -r .id)

# Wait for async embedding (max 10s)
for i in {1..10}; do
    STATUS=$(psql -t $SHIORI_TEST_PG_URL -c \
        "SELECT embedding_status FROM bookmark WHERE id=$BOOKMARK_ID")
    [ "$STATUS" = "completed" ] && break
    sleep 1
done

# Verify chunks created
CHUNKS=$(psql -t $SHIORI_TEST_PG_URL -c \
    "SELECT COUNT(*) FROM bookmark_chunks WHERE bookmark_id=$BOOKMARK_ID")
[ $CHUNKS -gt 0 ]

# Test backfill: Create 5 bookmarks manually with status=pending
psql $SHIORI_TEST_PG_URL -c "INSERT INTO bookmark (...) VALUES (...)"
# Trigger backfill
curl -X POST http://localhost:8080/api/v1/admin/backfill-embeddings

# Wait and verify all completed
sleep 5
PENDING=$(psql -t $SHIORI_TEST_PG_URL -c \
    "SELECT COUNT(*) FROM bookmark WHERE embedding_status='pending'")
[ $PENDING -eq 0 ]

kill $SERVER_PID
echo "✓ Embedding lifecycle automation works"
```

**Exit Criteria:**
- New bookmarks automatically queued for embedding
- Embedding completes asynchronously without blocking save
- Backfill processes all pending bookmarks on startup
- Backfill logs progress to stdout
- Failed embeddings marked with status=failed and error message
- Content updates trigger re-embedding
- All unit tests pass
- E2E lifecycle test succeeds

---

## Phase 6: Chat UI (Vue Frontend)

**Goal:** Build interactive chat panel in webapp

**Dependencies:** Phase 4

### Tasks

- [ ] Create Pinia store `webapp/src/stores/chat.ts`
  - State: messages[], isOpen, isStreaming, embeddingStatus
  - Actions: sendMessage(), openPanel(), closePanel(), clearHistory()
  - sessionStorage persistence for history
- [ ] Create Vue component `webapp/src/components/ChatPanel.vue`
  - Slide-out panel (400px width)
  - Message list with auto-scroll
  - User/assistant message bubbles
  - Source citation cards (expandable)
  - Input field with send button
  - "New conversation" button
  - Loading indicator during streaming
- [ ] Implement SSE client in `webapp/src/services/chatApi.ts`
  - `streamChat(message, history): EventSource`
  - Parse SSE events: chunk, sources, done, error
  - Reconnect on connection drop
- [ ] Add chat toggle button to sidebar navigation
- [ ] Create disabled state component
  - Show when `GET /api/v1/chat/status` returns `enabled: false`
  - Display configuration instructions
  - Link to documentation
- [ ] Style with Tailwind CSS (match existing Shiori design)
- [ ] Add mobile responsive layout (overlay on small screens)

### Completion Criteria

**Unit Tests (Vitest):**

```typescript
// webapp/src/stores/chat.test.ts
describe('useChatStore', () => {
  it('adds messages to history', ...)
  it('persists to sessionStorage', ...)
  it('clears history on clearHistory()', ...)
  it('toggles panel open/closed', ...)
})

// webapp/src/services/chatApi.test.ts
describe('streamChat', () => {
  it('parses SSE chunk events', ...)
  it('parses SSE sources event', ...)
  it('handles stream errors', ...)
})
```

**Component Tests:**

```typescript
// webapp/src/components/ChatPanel.test.ts
describe('ChatPanel', () => {
  it('renders messages', ...)
  it('sends message on button click', ...)
  it('shows loading during streaming', ...)
  it('displays source citations', ...)
  it('shows disabled state when AI not configured', ...)
})
```

**Manual Test Checklist:**

```markdown
- [ ] Panel slides in/out smoothly
- [ ] Messages appear in real-time during streaming
- [ ] Source citations clickable and navigate to bookmarks
- [ ] New conversation clears history
- [ ] History persists on page refresh (within session)
- [ ] History clears on browser tab close
- [ ] Mobile layout works (panel overlays content)
- [ ] Disabled state shows helpful instructions
- [ ] Keyboard shortcuts work (Enter to send)
- [ ] Auto-scroll keeps latest message visible
- [ ] Error messages display clearly
```

**E2E Test (Playwright):** `webapp/e2e/chat.spec.ts`

```typescript
test('chat flow', async ({ page }) => {
  await page.goto('/');
  await page.getByRole('button', { name: 'Chat' }).click();
  
  // Panel opens
  await expect(page.getByTestId('chat-panel')).toBeVisible();
  
  // Send message
  await page.getByPlaceholder('Ask about your bookmarks...').fill('What is Go?');
  await page.getByRole('button', { name: 'Send' }).click();
  
  // Wait for streaming response
  await expect(page.getByText(/Go is/)).toBeVisible({ timeout: 10000 });
  
  // Verify sources appear
  await expect(page.getByTestId('source-citation')).toBeVisible();
  
  // Click source navigates to bookmark
  await page.getByTestId('source-citation').first().click();
  await expect(page.url()).toContain('/bookmarks/');
});
```

**Exit Criteria:**
- All Vitest unit tests pass
- All component tests pass
- Manual test checklist 100% complete
- E2E test passes
- Chat UI matches design mockups
- Disabled state displays when AI not configured
- SSE streaming works without reconnection issues
- Mobile responsive layout works
- `bun run lint` and `bun run type-check` pass

---

## Phase 7: Documentation & Polish

**Goal:** Finalize documentation, error handling, and deployment

**Dependencies:** All previous phases

### Tasks

- [ ] Write user documentation in `docs/ai-chat.md`
  - Feature overview
  - Configuration guide (Ollama + Anthropic setup)
  - Usage examples
  - Troubleshooting
- [ ] Write admin documentation in `docs/ai-chat-admin.md`
  - PostgreSQL requirements
  - pgvector installation
  - Embedding backfill process
  - Monitoring and metrics
  - Performance tuning
- [ ] Update main README.md with AI chat feature
- [ ] Add migration guide for SQLite users
- [ ] Improve error messages:
  - AI not configured: Link to docs
  - Ollama unavailable: "Check Ollama is running at {host}"
  - Anthropic quota exceeded: Clear error with link to billing
  - pgvector not installed: Migration instructions
- [ ] Add structured logging with context:
  - Embedding success/failure with bookmark ID
  - Chat latency with user ID
  - Backfill progress with batch number
- [ ] Performance testing:
  - Load test chat API (100 concurrent users)
  - Verify vector search <100ms with 1000 bookmarks
  - Verify embedding generation <5s per bookmark
- [ ] Security audit:
  - API key not logged or exposed
  - User isolation enforced (can't search other users' bookmarks)
  - Input sanitization on chat messages
  - Rate limiting on chat endpoint

### Completion Criteria

**Documentation Review Checklist:**

```markdown
- [ ] All steps in setup guide tested on clean system
- [ ] Screenshots included for UI components
- [ ] Troubleshooting covers common errors
- [ ] Migration guide tested with sample SQLite database
- [ ] API documentation in Swagger is complete
```

**Load Test:** `scripts/load-test-chat.sh`

```bash
#!/bin/bash
# Use k6 or vegeta to simulate load
k6 run --vus 100 --duration 30s scripts/k6-chat.js

# Verify:
# - p95 latency < 3s
# - Error rate < 1%
# - No memory leaks
```

**Security Audit Checklist:**

```markdown
- [ ] API key not in logs (grep SHIORI_ANTHROPIC_API_KEY logs/)
- [ ] User A cannot retrieve user B's bookmarks via chat
- [ ] SQL injection prevented (parameterized queries)
- [ ] XSS prevented (sanitized chat input)
- [ ] Rate limiting prevents abuse (max 10 req/min per user)
- [ ] HTTPS enforced in production config
```

**Exit Criteria:**
- All documentation complete and reviewed
- Setup guide verified on clean Ubuntu 22.04 + macOS
- Load test passes performance targets
- Security audit checklist 100% complete
- All error messages user-friendly with actionable guidance
- Logging structured and queryable
- Feature ready for production deployment

---

## Cross-Phase Testing Strategy

### Continuous Integration

Each phase includes a CI job that must pass:

```yaml
# .github/workflows/ai-chat.yml
jobs:
  test-phase-1:
    - Run migration
    - Run unit tests for config + database
  
  test-phase-2:
    - Requires: Ollama service
    - Run embedding service tests
    - Run integration test script
  
  test-phase-3:
    - Requires: Ollama + Anthropic API key (from secrets)
    - Run chat domain tests
    - Run E2E test script
  
  test-phase-4:
    - Run API handler tests
    - Run cURL integration test
  
  test-phase-5:
    - Run worker tests
    - Run lifecycle E2E test
  
  test-phase-6:
    - Run Vitest unit tests
    - Run component tests
    - Run Playwright E2E tests
  
  test-phase-7:
    - Run full test suite
    - Run load tests
    - Run security audit
```

### Test Data Setup

Create reusable test fixtures:

```sql
-- testdata/ai_test_fixtures.sql
-- 5 bookmarks with known content on different topics:
-- 1. Go programming
-- 2. Python programming
-- 3. Database optimization
-- 4. Frontend frameworks
-- 5. DevOps practices
```

### Regression Testing

Before merging each phase:

```bash
# Run full existing test suite to prevent regressions
make unittest
make e2e

# Run AI-specific tests
go test -tags test_postgres_only,osusergo,netgo,fts5 ./internal/domains/...
cd webapp && bun run test:unit
```

---

## Dependency Graph

```
Phase 0 (Prerequisites)
    ↓
Phase 1 (Config + DB Schema)
    ↓
Phase 2 (Embedding Service) ──────┐
    ↓                              ↓
Phase 3 (Chat Domain)          Phase 5 (Lifecycle)
    ↓                              ↓
Phase 4 (API Endpoints)            ↓
    ↓                              ↓
Phase 6 (Chat UI) ←────────────────┘
    ↓
Phase 7 (Documentation)
```

**Notes:**
- Phase 5 only depends on Phase 2 (can run parallel to Phase 3-4)
- Phase 6 requires Phase 4 for API, but benefits from Phase 5 for full testing
- Phase 7 requires all phases complete

---

## Rollback Strategy

Each phase is designed to be independently revertible:

### Phase 1 Rollback
- Drop migration 0009
- Remove AI config code (feature flag keeps existing code working)

### Phase 2-3 Rollback
- Keep database schema (harmless if unused)
- Remove embedding/chat services
- No impact on existing features

### Phase 4 Rollback
- Remove API routes
- Frontend won't call missing endpoints

### Phase 5 Rollback
- Disable background worker
- Embeddings stop generating (existing data preserved)

### Phase 6 Rollback
- Hide chat UI button
- All backend functionality remains

### Full Rollback
- Run down migration to remove pgvector tables
- Remove AI config section
- System returns to pre-AI state

---

## Success Metrics (Post-Deployment)

Track these metrics to validate feature success:

| Metric | Target | Measurement |
|--------|--------|-------------|
| Embedding success rate | >95% | `COUNT(embedding_status='completed') / COUNT(*)` |
| Chat p95 latency | <3s | Application logs |
| Feature adoption | >30% of active users try within 30 days | Analytics event |
| User satisfaction | >80% positive | In-app survey (future) |
| Embedding backlog | <10% pending after 24h | `GET /api/v1/chat/status` |

---

## Risk Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Ollama rate limiting | High | Backfill stalls | Configurable batch delay, retry logic |
| Anthropic API costs | Medium | Unexpected bill | Token usage logging, budget alerts |
| pgvector performance | Low | Slow search | Index tuning, query optimization, load testing |
| Breaking change in dependencies | Low | Feature breaks | Pin versions, integration tests |
| Large embeddings overwhelm DB | Medium | Storage issues | Monitor table size, document cleanup strategy |

---

## Open Questions & Decisions

### Q1: Vector Dimension Flexibility
**Decision:** Lock to 768 dimensions for embeddinggemma in v1. Future versions can support configurable dimensions with schema migration.

**Rationale:** Simplifies implementation, embeddinggemma is sufficient for MVP.

### Q2: Content Extraction Timing
**Decision:** Re-extract content during embedding (don't use stored `content` field).

**Rationale:** Ensures consistency, allows for improved extraction logic over time. Slight performance cost is acceptable.

### Q3: Token Counting for Chunking
**Decision:** Use character approximation (4 chars ≈ 1 token) in v1.

**Rationale:** Avoids dependency on tiktoken. Good enough for chunking (exact count not critical).

### Q4: Conversation Length Limits
**Decision:** No hard limit in v1, but log warning at 10+ turns.

**Rationale:** Trust users to start new conversations when needed. Monitor usage to inform future limits.

### Q5: Embedding Re-generation on Update
**Decision:** Only re-embed if `content` or `html` fields change.

**Rationale:** Metadata updates (title, tags) don't affect retrieval quality.

---

## Definition of Done (Per Phase)

Each phase is considered complete when:

1. ✅ All tasks in checklist marked complete
2. ✅ All unit tests pass
3. ✅ Integration/E2E tests pass
4. ✅ Code review approved (if team-based)
5. ✅ Documentation updated
6. ✅ CI pipeline green
7. ✅ Manual testing checklist complete
8. ✅ No regressions in existing features
9. ✅ Performance benchmarks met
10. ✅ Security considerations addressed

---

## Estimated Timeline

| Phase | Estimated Duration | Parallel Work Possible |
|-------|-------------------|------------------------|
| Phase 0 | 1 day | No |
| Phase 1 | 2-3 days | No |
| Phase 2 | 3-4 days | No |
| Phase 3 | 3-4 days | Yes (with Phase 5) |
| Phase 4 | 2 days | No |
| Phase 5 | 2-3 days | Yes (with Phase 3) |
| Phase 6 | 4-5 days | No |
| Phase 7 | 2-3 days | No |
| **Total** | **19-27 days** | **~15-20 days with parallelization** |

**Note:** Timeline assumes single developer working full-time. Adjust for team size and availability.

---

## Post-Launch Iteration Plan

Features to consider after v1 launch:

1. **Tag-scoped chat** - "Ask about my 'golang' bookmarks"
2. **Persistent chat history** - Store conversations in database
3. **Multi-user chat** - Share conversations with other users
4. **Alternative embedding models** - Support for other Ollama models
5. **Hybrid search** - Combine vector search with keyword search
6. **Export conversations** - Download as markdown
7. **Custom prompts** - User-defined system prompts
8. **Chat analytics** - Usage dashboards for admins

Priority based on user feedback and adoption metrics.
