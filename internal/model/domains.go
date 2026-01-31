package model

import (
	"context"
	"io/fs"
	"os"
	"time"

	"github.com/go-shiori/warc"
	"github.com/spf13/afero"
)

type BookmarksDomain interface {
	HasEbook(b *BookmarkDTO) bool
	HasArchive(b *BookmarkDTO) bool
	HasThumbnail(b *BookmarkDTO) bool
	GetBookmark(ctx context.Context, id DBID) (*BookmarkDTO, error)
	GetBookmarks(ctx context.Context, ids []int) ([]BookmarkDTO, error)
	UpdateBookmarkCache(ctx context.Context, bookmark BookmarkDTO, keepMetadata bool, skipExist bool) (*BookmarkDTO, error)
	BulkUpdateBookmarkTags(ctx context.Context, bookmarkIDs []int, tagIDs []int) error
	AddTagToBookmark(ctx context.Context, bookmarkID int, tagID int) error
	RemoveTagFromBookmark(ctx context.Context, bookmarkID int, tagID int) error
	BookmarkExists(ctx context.Context, id int) (bool, error)
}

type AuthDomain interface {
	CheckToken(ctx context.Context, userJWT string) (*AccountDTO, error)
	GetAccountFromCredentials(ctx context.Context, username, password string) (*AccountDTO, error)
	CreateTokenForAccount(account *AccountDTO, expiration time.Time) (string, error)
}

type AccountsDomain interface {
	ListAccounts(ctx context.Context) ([]AccountDTO, error)
	GetAccountByUsername(ctx context.Context, username string) (*AccountDTO, error)
	CreateAccount(ctx context.Context, account AccountDTO) (*AccountDTO, error)
	UpdateAccount(ctx context.Context, account AccountDTO) (*AccountDTO, error)
	DeleteAccount(ctx context.Context, id int) error
}

type ArchiverDomain interface {
	DownloadBookmarkArchive(book BookmarkDTO) (*BookmarkDTO, error)
	GetBookmarkArchive(book *BookmarkDTO) (*warc.Archive, error)
}

type StorageDomain interface {
	Stat(name string) (fs.FileInfo, error)
	FS() afero.Fs
	FileExists(path string) bool
	DirExists(path string) bool
	WriteData(dst string, data []byte) error
	WriteFile(dst string, src *os.File) error
}

type TagsDomain interface {
	ListTags(ctx context.Context, opts ListTagsOptions) ([]TagDTO, error)
	CreateTag(ctx context.Context, tag TagDTO) (TagDTO, error)
	GetTag(ctx context.Context, id int) (TagDTO, error)
	UpdateTag(ctx context.Context, tag TagDTO) (TagDTO, error)
	DeleteTag(ctx context.Context, id int) error
	TagExists(ctx context.Context, id int) (bool, error)
}

// EmbeddingDomain handles embedding generation and similarity search
type EmbeddingDomain interface {
	// EmbedBookmark generates and stores embeddings for a bookmark
	EmbedBookmark(ctx context.Context, bookmarkID int) error
	// SearchSimilar finds chunks similar to the query text
	SearchSimilar(ctx context.Context, query string, threshold float64, limit int) ([]ChunkResult, error)
	// BackfillEmbeddings processes all bookmarks with pending embedding status
	BackfillEmbeddings(ctx context.Context, batchSize int, delayMs int) error
	// GetEmbeddingStatus returns the embedding status summary
	GetEmbeddingStatus(ctx context.Context) (*EmbeddingStatusSummary, error)
	// ReembedBookmark re-generates embeddings for a bookmark
	ReembedBookmark(ctx context.Context, bookmarkID int) error
	// IsOllamaAvailable checks if Ollama is reachable
	IsOllamaAvailable(ctx context.Context) error
}

// ChatDomain handles AI-powered chat over bookmark content
type ChatDomain interface {
	// Chat performs Q&A over user's bookmarks and returns a streaming response
	Chat(ctx context.Context, message string, history []ChatMessage) (<-chan StreamChatEvent, error)
	// ChatSync performs Q&A over user's bookmarks and returns a complete response
	ChatSync(ctx context.Context, message string, history []ChatMessage) (*ChatResponse, error)
	// GetEmbeddingStatus returns the embedding status summary
	GetEmbeddingStatus(ctx context.Context) (*EmbeddingStatusSummary, error)
	// IsAvailable checks if the chat service is available
	IsAvailable(ctx context.Context) error
}
