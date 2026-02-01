# PRD: BookmarkCard Component with Offline Content Navigation

## Overview

Create a `BookmarkCard.vue` component for the Shiori webapp that displays bookmark information and provides intuitive navigation to offline/readable content with fallback to original URLs.

## Problem Statement

The current HomeView displays bookmarks inline without a dedicated component, limiting reusability and lacking proper offline content navigation. Users need a clear, intuitive way to:
1. Access saved offline/readable versions of bookmarks
2. Navigate to original articles when needed
3. Fetch offline content for bookmarks that don't have it cached

## Goals

- Provide a reusable, well-structured BookmarkCard component
- Enable seamless navigation to offline content with intelligent fallbacks
- Allow users to fetch offline content on-demand
- Deliver clear feedback during async operations via toasts

## Non-Goals

- Edit/delete bookmark functionality (separate feature)
- Bulk operations on bookmarks
- Bookmark creation flow
- Search/filter functionality

---

## Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Toast Library | **vue-sonner** | Modern, lightweight, beautiful defaults |
| Readable Content | **New Vue route/page** | Create `/bookmark/:id/content` with ReadableView.vue |
| Re-fetch Behavior | **Hide icon if content exists** | No icon shown when `hasContent: true` |
| Loading State | **Icon transforms to spinner** | Download icon becomes spinning loader |
| Component Location | **`components/bookmarks/`** | Namespaced folder for bookmark components |

---

## Feature Requirements

### BookmarkCard Display Elements

The card must display:

| Element | Source | Required |
|---------|--------|----------|
| Title | `bookmark.title` | Yes |
| URL | `bookmark.url` | Yes |
| Thumbnail | `bookmark.imageURL` | No (show placeholder or hide) |
| Excerpt | `bookmark.excerpt` | No (truncate if present) |
| Tags | `bookmark.tags[]` | No (show chips if present) |
| Created Date | `bookmark.createdAt` | Yes |
| Archive Indicator | `bookmark.hasArchive` | Yes (icon/badge) |
| Ebook Indicator | `bookmark.hasEbook` | Yes (icon/badge) |

### Interaction Behaviors

#### 1. Title Click
- **If `hasContent: true`**: Navigate to `/bookmark/:id/content` (ReadableView)
- **If `hasContent: false`**: Navigate to original `bookmark.url` (external)

#### 2. URL Text / External Link Icon Click
- Always opens `bookmark.url` in a new tab (`target="_blank"`)

#### 3. Cloud Download Icon (Action Area)
- **Visibility**: Only shown when `hasContent: false`
- **On Click**:
  1. Icon transforms to loading spinner
  2. Show toast: "Fetching offline content..."
  3. Call `PUT /api/v1/bookmarks/cache` with bookmark data
  4. **On Success**:
     - Show success toast: "Offline content saved"
     - Update card state (`hasContent: true`)
     - Hide download icon
  5. **On Failure**:
     - Show error toast: "Failed to fetch content"
     - Navigate to original `bookmark.url`
     - Restore download icon state

### Content Indicators (Read-only)

| Indicator | Condition | Display |
|-----------|-----------|---------|
| Archive | `hasArchive: true` | Archive icon/badge |
| Ebook | `hasEbook: true` | Book icon/badge |
| Offline Content | `hasContent: true` | No indicator needed (title behavior changes) |

---

## New Components & Files

### 1. `webapp/src/components/bookmarks/BookmarkCard.vue`

Primary component displaying a single bookmark.

**Props:**
```typescript
interface Props {
  bookmark: ModelBookmarkDTO
}
```

**Emits:**
```typescript
interface Emits {
  (e: 'content-fetched', bookmark: ModelBookmarkDTO): void
}
```

### 2. `webapp/src/views/ReadableView.vue`

New view for displaying offline/readable content.

**Route**: `/bookmark/:id/content`

**Behavior**:
- Fetch readable content via `GET /api/v1/bookmarks/:id/readable`
- Display HTML content with proper styling
- Provide back navigation to home
- Show original URL link

### 3. Router Updates

Add route to `webapp/src/router/index.ts`:
```typescript
{
  path: '/bookmark/:id/content',
  name: 'readable',
  component: () => import('../views/ReadableView.vue'),
  meta: { requiresAuth: true }
}
```

### 4. Toast Setup

Install and configure vue-sonner in `webapp/src/main.ts`.

---

## API Dependencies

### Existing Endpoints Used

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/bookmarks` | GET | List bookmarks (existing) |
| `/api/v1/bookmarks/:id/readable` | GET | Fetch readable content |
| `/api/v1/bookmarks/cache` | PUT | Fetch/update offline content |

### API Client Updates

The generated API client may need fixes:
- `apiV1BookmarksIdReadableGet()` currently doesn't accept ID parameter (bug in generator)
- May need manual API method or client regeneration

---

## State Management

### Bookmarks Store Updates

Add to `webapp/src/stores/bookmarks.ts`:

```typescript
// Update a single bookmark in the store
const updateBookmark = (updatedBookmark: ModelBookmarkDTO) => {
  const index = bookmarks.value.findIndex(b => b.id === updatedBookmark.id)
  if (index !== -1) {
    bookmarks.value[index] = updatedBookmark
  }
}

// Fetch content for a bookmark
const fetchBookmarkContent = async (bookmark: ModelBookmarkDTO): Promise<ModelBookmarkDTO> => {
  // Call cache endpoint
  // Return updated bookmark
}
```

---

## UI/UX Specifications

### Card Layout

```
+--------------------------------------------------+
|  [Thumbnail]  |  Title (clickable)               |
|               |  url.com (clickable) [external]  |
|               |  Excerpt text truncated...       |
|               |                                  |
|               |  [tag1] [tag2] [tag3]           |
|               |                                  |
|               |  Jan 31, 2026  [archive][ebook] |
+--------------------------------------------------+
|  Actions: [download-cloud] (if no content)       |
+--------------------------------------------------+
```

### Icon Choices (using existing icon library or add)

| Action | Icon | State |
|--------|------|-------|
| External link | `external-link` or `arrow-up-right` | Static |
| Download content | `cloud-download` | Default |
| Loading | `loader` / `spinner` | Animated rotation |
| Has archive | `archive` | Indicator |
| Has ebook | `book` | Indicator |

### Toast Messages

| Event | Type | Message |
|-------|------|---------|
| Fetching started | Info | "Fetching offline content..." |
| Fetch success | Success | "Offline content saved" |
| Fetch failed | Error | "Failed to fetch content" |

---

## Acceptance Criteria

### BookmarkCard Component

- [ ] Displays all required bookmark fields (title, URL, thumbnail, excerpt, tags, date)
- [ ] Shows archive indicator when `hasArchive: true`
- [ ] Shows ebook indicator when `hasEbook: true`
- [ ] Title click navigates to `/bookmark/:id/content` when `hasContent: true`
- [ ] Title click navigates to original URL when `hasContent: false`
- [ ] URL click always opens original URL in new tab
- [ ] External link icon opens original URL in new tab
- [ ] Download icon only visible when `hasContent: false`
- [ ] Download icon shows spinner during fetch
- [ ] Successful fetch shows success toast and updates card
- [ ] Failed fetch shows error toast and navigates to original URL

### ReadableView

- [ ] Route `/bookmark/:id/content` is accessible
- [ ] Displays readable HTML content
- [ ] Shows bookmark title and original URL
- [ ] Provides navigation back to home
- [ ] Handles loading and error states

### Toast System

- [ ] vue-sonner is installed and configured
- [ ] Toasts appear for fetch start, success, and failure
- [ ] Toasts auto-dismiss appropriately

### Integration

- [ ] BookmarkCard replaces inline bookmark display in HomeView
- [ ] Bookmarks store updated with new methods
- [ ] All existing functionality preserved

---

## Technical Notes

### Styling

- Use existing Tailwind CSS classes
- Match current webapp design language
- Ensure responsive behavior

### Error Handling

- Network errors should show user-friendly messages
- Auth errors (401) should redirect to login
- API errors should be logged to console

### Accessibility

- Clickable elements should be focusable
- Icons should have aria-labels
- Loading states should be announced

---

## Dependencies to Install

```bash
cd webapp
bun add vue-sonner
```

---

## Out of Scope (Future Considerations)

- Edit bookmark inline
- Delete bookmark with confirmation
- Drag-and-drop reordering
- Bookmark sharing
- Archive/Ebook download actions (indicators only for now)

---

## Open Questions

None - all clarifications resolved.

---

## Revision History

| Date | Version | Changes |
|------|---------|---------|
| 2026-01-31 | 1.0 | Initial PRD |
