# PRD: Bookmark Q&A Chat Interface

## Overview

Add an AI-powered chat interface to Shiori that enables users to ask questions about their bookmarked content using natural language. The system uses semantic search over bookmark content to retrieve relevant context and generates answers using Anthropic's Claude API.

## Problem Statement

Users with large bookmark collections struggle to find specific information across their saved articles. Traditional keyword search requires knowing exact terms, and browsing is time-consuming. An AI chat interface allows users to ask natural questions like "What did I save about Go error handling?" and receive synthesized answers with source citations.

## Goals

1. Enable natural language Q&A over bookmark content
2. Provide accurate, contextual answers with source attribution
3. Maintain Shiori's self-hosted philosophy with local embedding generation
4. Integrate seamlessly with existing bookmark workflow
5. Support multi-turn conversations within a session

## Non-Goals

- Persistent chat history across sessions (browser-only storage)
- Real-time collaborative chat
- Chat about content not in user's bookmarks
- Support for non-PostgreSQL databases (pgvector requirement)
- Per-bookmark or tag-scoped chat (v1 searches all bookmarks)

---

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| LLM | Anthropic Claude (`claude-sonnet-4-20250514`) | Best balance of quality, speed, cost |
| Embeddings | Ollama (`embeddinggemma`) | Local, no API costs, self-hosted aligned |
| Vector Storage | PostgreSQL + pgvector | Existing PG support, no new dependencies |
| Chunking | 512 tokens, 50 overlap | Standard retrieval-optimized chunking |
| Retrieval | Dynamic similarity threshold | Adapts to query specificity |
| Streaming | Server-Sent Events (SSE) | Real-time response streaming |
| Chat History | Browser sessionStorage | No server persistence required |

---

## User Stories

### US-1: Ask Questions About Bookmarks
**As a** Shiori user  
**I want to** ask natural language questions about my saved bookmarks  
**So that** I can quickly find information without manual searching

**Acceptance Criteria:**
- User can open chat panel from any page in the webapp
- User can type a question and receive an AI-generated answer
- Answer includes clickable links to source bookmarks
- Response streams in real-time as it's generated

### US-2: Multi-Turn Conversations
**As a** user asking follow-up questions  
**I want to** continue a conversation with context from previous messages  
**So that** I can drill down into topics without repeating context

**Acceptance Criteria:**
- Chat maintains conversation history within browser session
- Follow-up questions understand context from previous Q&A
- User can start a new conversation to clear context
- History is lost on page refresh/close (by design)

### US-3: Automatic Embedding on Save
**As a** user saving new bookmarks  
**I want** content automatically indexed for chat  
**So that** new bookmarks are immediately searchable via Q&A

**Acceptance Criteria:**
- Embedding generated asynchronously after bookmark save
- Bookmark save succeeds even if embedding fails
- Failed embeddings logged for admin visibility
- User not blocked waiting for embedding completion

### US-4: Backfill Existing Bookmarks
**As an** admin upgrading Shiori  
**I want** existing bookmarks automatically embedded  
**So that** the chat feature works with my existing collection

**Acceptance Criteria:**
- Background job processes unembedded bookmarks on server start
- Processing is throttled to avoid overwhelming Ollama
- Progress visible in server logs
- Job is idempotent (safe to restart)

### US-5: Graceful Degradation
**As a** user without AI configured  
**I want to** understand what's missing  
**So that** I can enable the feature if desired

**Acceptance Criteria:**
- Chat UI visible but shows disabled state
- Clear message explaining required configuration
- Links to documentation for setup
- No errors thrown for unconfigured state

---

## Functional Requirements

### FR-1: Configuration

New server configuration options:

```yaml
# AI Chat Configuration
ai:
  enabled: true
  
  # Anthropic API settings
  anthropic:
    api_key: "sk-ant-..."  # Required for chat
    model: "claude-sonnet-4-20250514"  # Default model
    max_tokens: 4096  # Max response tokens
  
  # Ollama embedding settings  
  ollama:
    host: "http://localhost:11434"  # Ollama server URL
    model: "embeddinggemma"  # Embedding model
    
  # Retrieval settings
  retrieval:
    similarity_threshold: 0.7  # Minimum similarity score
    max_chunks: 20  # Maximum chunks to retrieve
    
  # Embedding settings
  embedding:
    chunk_size: 512  # Tokens per chunk
    chunk_overlap: 50  # Overlap between chunks
    backfill_batch_size: 10  # Bookmarks per backfill batch
    backfill_delay_ms: 1000  # Delay between batches
```

Environment variable overrides:
- `SHIORI_AI_ENABLED`
- `SHIORI_ANTHROPIC_API_KEY`
- `SHIORI_ANTHROPIC_MODEL`
- `SHIORI_OLLAMA_HOST`
- `SHIORI_OLLAMA_MODEL`

### FR-2: Database Schema

New tables for pgvector storage:

```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Bookmark content chunks with embeddings
CREATE TABLE bookmark_chunks (
    id SERIAL PRIMARY KEY,
    bookmark_id INTEGER NOT NULL REFERENCES bookmark(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,  -- Order within bookmark
    content TEXT NOT NULL,  -- Chunk text
    embedding vector(768),  -- Dimension matches embeddinggemma
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(bookmark_id, chunk_index)
);

-- Index for vector similarity search
CREATE INDEX ON bookmark_chunks 
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- Track embedding status
ALTER TABLE bookmark ADD COLUMN embedding_status VARCHAR(20) 
    DEFAULT 'pending' 
    CHECK (embedding_status IN ('pending', 'processing', 'completed', 'failed', 'skipped'));
    
ALTER TABLE bookmark ADD COLUMN embedding_error TEXT;
ALTER TABLE bookmark ADD COLUMN embedded_at TIMESTAMP;
```

### FR-3: Chat Domain Service

New domain service: `internal/domains/chat.go`

```go
type ChatDomain interface {
    // Chat performs Q&A over user's bookmarks
    Chat(ctx context.Context, userID int, message string, history []ChatMessage) (*ChatResponse, error)
    
    // GetEmbeddingStatus returns embedding progress for a user
    GetEmbeddingStatus(ctx context.Context, userID int) (*EmbeddingStatus, error)
}

type ChatMessage struct {
    Role    string `json:"role"`    // "user" or "assistant"
    Content string `json:"content"`
}

type ChatResponse struct {
    Message string           `json:"message"`
    Sources []BookmarkSource `json:"sources"`
}

type BookmarkSource struct {
    ID      int    `json:"id"`
    Title   string `json:"title"`
    URL     string `json:"url"`
    Excerpt string `json:"excerpt"`  // Relevant chunk
}
```

### FR-4: Embedding Service

New service: `internal/domains/embedding.go`

```go
type EmbeddingService interface {
    // EmbedBookmark generates and stores embeddings for a bookmark
    EmbedBookmark(ctx context.Context, bookmarkID int) error
    
    // SearchSimilar finds chunks similar to query
    SearchSimilar(ctx context.Context, userID int, query string, threshold float64, limit int) ([]ChunkResult, error)
    
    // BackfillEmbeddings processes all pending bookmarks
    BackfillEmbeddings(ctx context.Context) error
}

type ChunkResult struct {
    BookmarkID int
    Content    string
    Similarity float64
}
```

### FR-5: API Endpoint

**POST /api/v1/chat**

Request:
```json
{
    "message": "What articles did I save about database optimization?",
    "history": [
        {"role": "user", "content": "Previous question"},
        {"role": "assistant", "content": "Previous answer"}
    ]
}
```

Response (SSE stream):
```
event: chunk
data: {"content": "Based on "}

event: chunk
data: {"content": "your bookmarks, "}

event: chunk  
data: {"content": "I found several articles..."}

event: sources
data: {"sources": [{"id": 123, "title": "PostgreSQL Tips", "url": "..."}]}

event: done
data: {}
```

Error response:
```json
{
    "error": "AI features not configured",
    "code": "AI_NOT_CONFIGURED"
}
```

**GET /api/v1/chat/status**

Returns AI feature configuration status:
```json
{
    "enabled": true,
    "embedding_status": {
        "total": 150,
        "completed": 145,
        "pending": 3,
        "failed": 2
    }
}
```

### FR-6: Embedding Lifecycle

1. **On Bookmark Save:**
   - Bookmark saved to database normally
   - `embedding_status` set to `pending`
   - Async job queued for embedding generation
   - Job extracts readable content, chunks it, generates embeddings via Ollama
   - On success: chunks stored, status set to `completed`
   - On failure: status set to `failed`, error logged

2. **On Bookmark Update:**
   - If content changed, delete existing chunks
   - Re-queue for embedding generation

3. **On Bookmark Delete:**
   - Chunks cascade deleted via foreign key

4. **Backfill Process:**
   - Runs on server startup if enabled
   - Queries bookmarks where `embedding_status = 'pending'`
   - Processes in batches with configurable delay
   - Logs progress: "Embedding backfill: 45/150 complete"

### FR-7: Chat UI Component

Vue component: `webapp/src/components/ChatPanel.vue`

**Layout:**
- Slide-out panel from right side of screen
- Toggle button in sidebar navigation
- Panel width: 400px (configurable)
- Overlay on mobile, side-by-side on desktop

**Components:**
- Message list (scrollable)
- Input field with send button
- "New conversation" button
- Source citations as expandable cards
- Loading indicator during streaming
- Disabled state with configuration instructions

**State Management:**
- Pinia store: `useChatStore`
- History stored in sessionStorage
- Clears on explicit "New conversation" or session end

---

## Non-Functional Requirements

### NFR-1: Performance
- Chat response start (first token) < 2 seconds
- Embedding generation < 5 seconds per bookmark (average)
- Vector search < 100ms for 1,000 bookmarks
- Backfill processes at minimum 1 bookmark/second

### NFR-2: Reliability
- Embedding failures don't block bookmark operations
- Chat gracefully handles Ollama/Anthropic unavailability
- Automatic retry with exponential backoff for transient failures

### NFR-3: Security
- Anthropic API key stored securely (not logged, not exposed to frontend)
- Users can only search their own bookmarks (user_id filter enforced)
- Input sanitization on chat messages
- SSE endpoint requires authentication

### NFR-4: Observability
- Structured logging for embedding jobs
- Metrics: embedding success/failure rate, chat latency, token usage
- Embedding status queryable via API

---

## Database Compatibility

This feature **requires PostgreSQL** with pgvector extension.

| Database | Support |
|----------|---------|
| PostgreSQL 14+ | ✅ Full support with pgvector |
| SQLite | ❌ Not supported |
| MySQL/MariaDB | ❌ Not supported |

Users on SQLite/MySQL will see the chat feature disabled with a message explaining PostgreSQL is required.

---

## Migration Path

### For New Installations
1. PostgreSQL setup includes pgvector extension
2. AI configuration optional but documented
3. Chat feature available immediately if configured

### For Existing PostgreSQL Users
1. Migration adds pgvector extension and new tables
2. Existing bookmarks have `embedding_status = 'pending'`
3. Backfill runs automatically on first server start
4. Feature works once Ollama + Anthropic configured

### For SQLite/MySQL Users
1. Feature gracefully disabled
2. Documentation explains PostgreSQL migration path
3. No breaking changes to existing functionality

---

## Implementation Phases

### Phase 1: Foundation (Backend)
- [ ] Add AI configuration schema
- [ ] Create pgvector migration
- [ ] Implement Ollama client for embeddings
- [ ] Implement chunking logic
- [ ] Create embedding service with embed/search methods
- [ ] Add embedding status to bookmark model

### Phase 2: Chat Backend
- [ ] Implement Anthropic client with streaming
- [ ] Create chat domain service
- [ ] Implement RAG retrieval logic
- [ ] Create POST /api/v1/chat endpoint with SSE
- [ ] Create GET /api/v1/chat/status endpoint
- [ ] Add chat handler tests

### Phase 3: Embedding Automation
- [ ] Hook embedding into bookmark save flow
- [ ] Implement backfill background job
- [ ] Add embedding failure handling and logging
- [ ] Test embedding lifecycle

### Phase 4: Chat UI
- [ ] Create ChatPanel Vue component
- [ ] Implement chat Pinia store
- [ ] Add SSE client for streaming responses
- [ ] Build message list with source citations
- [ ] Add toggle button to sidebar
- [ ] Implement disabled state UI
- [ ] Add sessionStorage persistence

### Phase 5: Polish & Documentation
- [ ] End-to-end testing
- [ ] Performance optimization
- [ ] Documentation for AI setup
- [ ] Error message improvements

---

## Open Questions

1. **Vector dimension flexibility:** Should we support configurable dimensions for different Ollama models, or lock to 768 for embeddinggemma?

2. **Content extraction:** Should we re-extract content on embed, or use stored `content` field? (Re-extraction ensures consistency but adds processing time)

3. **Token counting:** How to accurately count tokens for chunking? Use tiktoken-go or approximate by characters?

4. **Conversation limits:** Should there be a max conversation length before suggesting "New conversation"?

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Chat response relevance | >80% user satisfaction (future survey) |
| Embedding success rate | >95% of bookmarks with readable content |
| p95 chat latency (first token) | <3 seconds |
| Feature adoption | >30% of active users try chat within 30 days |

---

## Appendix A: System Prompt Template

```
You are a helpful assistant that answers questions about the user's bookmarked articles and web pages. 

Use ONLY the provided context from their bookmarks to answer questions. If the context doesn't contain relevant information, say so clearly rather than making up an answer.

When referencing information, naturally mention which bookmark it came from so the user can explore further.

Keep answers concise but informative. Use markdown formatting when helpful.

Context from bookmarks:
---
{retrieved_chunks}
---
```

## Appendix B: Example Interaction

**User:** What did I save about Go error handling?

**System retrieves chunks from:**
- "Error Handling in Go" (bookmark #42)
- "Go Best Practices 2024" (bookmark #87)

**Assistant:** Based on your bookmarks, you've saved a couple of articles about Go error handling:

From **"Error Handling in Go"**, the key points are:
- Always check errors immediately after function calls
- Use `fmt.Errorf` with `%w` verb to wrap errors with context
- Create sentinel errors for expected error conditions

**"Go Best Practices 2024"** adds that you should:
- Prefer `errors.Is()` and `errors.As()` over type assertions
- Consider using structured errors for complex applications

Would you like more details on any of these topics?

**Sources:**
- [Error Handling in Go](link) - "Always handle errors explicitly..."
- [Go Best Practices 2024](link) - "Modern Go code should use..."
