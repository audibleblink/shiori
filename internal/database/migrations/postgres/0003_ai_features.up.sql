-- Enable pgvector extension for vector similarity search
CREATE EXTENSION IF NOT EXISTS vector;

-- Add embedding status tracking columns to bookmark table
ALTER TABLE bookmark 
ADD COLUMN IF NOT EXISTS embedding_status VARCHAR(20) DEFAULT 'pending'
    CHECK (embedding_status IN ('pending', 'processing', 'completed', 'failed', 'skipped'));

ALTER TABLE bookmark 
ADD COLUMN IF NOT EXISTS embedding_error TEXT;

ALTER TABLE bookmark 
ADD COLUMN IF NOT EXISTS embedded_at TIMESTAMP;

-- Create bookmark_chunks table to store chunked content with embeddings
CREATE TABLE IF NOT EXISTS bookmark_chunks (
    id SERIAL PRIMARY KEY,
    bookmark_id INTEGER NOT NULL REFERENCES bookmark(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,  -- Order within bookmark
    content TEXT NOT NULL,         -- Chunk text content
    embedding vector(768),         -- Embedding vector (768 dimensions for embeddinggemma)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(bookmark_id, chunk_index)
);

-- Create index for foreign key lookups
CREATE INDEX IF NOT EXISTS idx_bookmark_chunks_bookmark_id ON bookmark_chunks(bookmark_id);

-- Create IVFFlat index for vector similarity search
-- Using cosine similarity which is common for text embeddings
CREATE INDEX IF NOT EXISTS idx_bookmark_chunks_embedding ON bookmark_chunks 
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- Index for finding bookmarks by embedding status (for backfill processing)
CREATE INDEX IF NOT EXISTS idx_bookmark_embedding_status ON bookmark(embedding_status);
