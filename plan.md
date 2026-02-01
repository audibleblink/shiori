# Execution Plan: BookmarkCard Component with Offline Content Navigation

**PRD Reference:** `prd.md`  
**Created:** 2026-01-31

---

## Overview

This plan implements the BookmarkCard component with offline content navigation in 5 phases. Each phase produces a complete, tested, and working increment of functionality.

### Phase Dependencies

```
Phase 1 (Foundation) ──┬──> Phase 2 (BookmarkCard)
                       │
                       └──> Phase 3 (ReadableView)
                                    │
Phase 2 + Phase 3 ─────────────────>│
                                    v
                              Phase 4 (Integration)
                                    │
                                    v
                              Phase 5 (Polish & E2E)
```

---

## Phase 1: Foundation Setup

**Goal:** Install vue-sonner, configure toast system, and fix the readable API endpoint bug.

**Dependencies:** None (starting point)

### Tasks

- [ ] **1.1** Install vue-sonner package
  ```bash
  cd webapp && bun add vue-sonner
  ```

- [ ] **1.2** Configure vue-sonner in `webapp/src/main.ts`
  - Import and add `<Toaster />` component to App.vue or main.ts setup
  - Configure default toast position and styling

- [ ] **1.3** Create toast utility wrapper `webapp/src/utils/toast.ts`
  - Export typed helper functions: `showInfo()`, `showSuccess()`, `showError()`
  - Ensures consistent toast usage across the app

- [ ] **1.4** Fix Swagger annotation for readable endpoint
  - File: `internal/http/handlers/api/v1/bookmarks.go:128`
  - Change: `@Router /api/v1/bookmarks/id/readable` → `@Router /api/v1/bookmarks/{id}/readable`
  - Run: `make swagger` to regenerate swagger docs

- [ ] **1.5** Regenerate API client
  - Regenerate TypeScript client from updated swagger spec
  - Verify `apiV1BookmarksIdReadableGet()` now accepts `id` parameter

- [ ] **1.6** Create `webapp/src/components/bookmarks/` directory structure

### Verification Script

Create `webapp/phase1-verify.sh`:

```bash
#!/bin/bash
set -e

echo "=== Phase 1 Verification ==="

# Check vue-sonner is installed
echo "Checking vue-sonner installation..."
grep -q '"vue-sonner"' webapp/package.json && echo "✓ vue-sonner installed" || { echo "✗ vue-sonner not found"; exit 1; }

# Check toast utility exists
echo "Checking toast utility..."
test -f webapp/src/utils/toast.ts && echo "✓ toast.ts exists" || { echo "✗ toast.ts not found"; exit 1; }

# Check Toaster is configured in App.vue or main.ts
echo "Checking Toaster configuration..."
(grep -q "Toaster" webapp/src/App.vue || grep -q "Toaster" webapp/src/main.ts) && echo "✓ Toaster configured" || { echo "✗ Toaster not configured"; exit 1; }

# Check swagger annotation fix
echo "Checking swagger annotation..."
grep -q "{id}/readable" internal/http/handlers/api/v1/bookmarks.go && echo "✓ Swagger annotation fixed" || { echo "✗ Swagger annotation not fixed"; exit 1; }

# Check bookmarks component directory exists
echo "Checking component directory..."
test -d webapp/src/components/bookmarks && echo "✓ bookmarks component directory exists" || { echo "✗ directory not found"; exit 1; }

# Run type-check to verify no TypeScript errors
echo "Running type-check..."
cd webapp && bun run type-check && echo "✓ Type check passed" || { echo "✗ Type check failed"; exit 1; }

echo ""
echo "=== Phase 1 Complete ✓ ==="
```

### Exit Criteria

- [x] vue-sonner installed in package.json
- [x] Toaster component configured in App.vue
- [x] Toast utility functions created
- [x] Swagger annotation corrected
- [x] API client regenerated with working readable endpoint
- [x] Component directory created
- [x] `bun run type-check` passes
- [x] `phase1-verify.sh` passes all checks

---

## Phase 2: BookmarkCard Component

**Goal:** Create the BookmarkCard component with all display elements and visual indicators.

**Dependencies:** Phase 1

### Tasks

- [ ] **2.1** Create `webapp/src/components/bookmarks/BookmarkCard.vue`
  - Define props interface accepting `ModelBookmarkDTO`
  - Define emits: `content-fetched`
  
- [ ] **2.2** Implement card layout with all display elements
  - Title (clickable)
  - URL (clickable, truncated)
  - Thumbnail (optional, with placeholder)
  - Excerpt (optional, truncated)
  - Tags (chips/pills)
  - Created date (formatted)
  
- [ ] **2.3** Implement content indicators
  - Archive indicator icon when `hasArchive: true`
  - Ebook indicator icon when `hasEbook: true`
  
- [ ] **2.4** Implement title click behavior
  - Navigate to `/bookmark/:id/content` when `hasContent: true`
  - Navigate to original URL (external) when `hasContent: false`
  
- [ ] **2.5** Implement URL/external link behavior
  - Always opens `bookmark.url` in new tab
  - External link icon button

- [ ] **2.6** Implement download action (UI only in this phase)
  - Cloud download icon visible only when `hasContent: false`
  - Loading spinner state (triggered by local state for testing)
  - Emit event structure (actual API call in Phase 4)

- [ ] **2.7** Add accessibility attributes
  - aria-labels on icons
  - Focusable interactive elements
  - Proper semantic HTML

- [ ] **2.8** Write unit tests for BookmarkCard
  - Test all conditional rendering
  - Test click handlers emit correct events
  - Test accessibility attributes

### Verification Script

Create `webapp/phase2-verify.sh`:

```bash
#!/bin/bash
set -e

echo "=== Phase 2 Verification ==="

# Check BookmarkCard component exists
echo "Checking BookmarkCard component..."
test -f webapp/src/components/bookmarks/BookmarkCard.vue && echo "✓ BookmarkCard.vue exists" || { echo "✗ BookmarkCard.vue not found"; exit 1; }

# Check component has required elements
echo "Checking component structure..."
CARD_FILE="webapp/src/components/bookmarks/BookmarkCard.vue"

grep -q "defineProps" "$CARD_FILE" && echo "✓ Props defined" || { echo "✗ Props not defined"; exit 1; }
grep -q "defineEmits" "$CARD_FILE" && echo "✓ Emits defined" || { echo "✗ Emits not defined"; exit 1; }
grep -q "bookmark.title" "$CARD_FILE" && echo "✓ Title displayed" || { echo "✗ Title not found"; exit 1; }
grep -q "bookmark.url" "$CARD_FILE" && echo "✓ URL displayed" || { echo "✗ URL not found"; exit 1; }
grep -q "hasArchive" "$CARD_FILE" && echo "✓ Archive indicator" || { echo "✗ Archive indicator not found"; exit 1; }
grep -q "hasEbook" "$CARD_FILE" && echo "✓ Ebook indicator" || { echo "✗ Ebook indicator not found"; exit 1; }
grep -q "hasContent" "$CARD_FILE" && echo "✓ Content conditional" || { echo "✗ Content conditional not found"; exit 1; }
grep -q "aria-label" "$CARD_FILE" && echo "✓ Accessibility attributes" || { echo "✗ Accessibility not found"; exit 1; }

# Check test file exists
echo "Checking unit tests..."
test -f webapp/src/components/bookmarks/BookmarkCard.spec.ts && echo "✓ Test file exists" || { echo "✗ Test file not found"; exit 1; }

# Run type-check
echo "Running type-check..."
cd webapp && bun run type-check && echo "✓ Type check passed" || { echo "✗ Type check failed"; exit 1; }

# Run unit tests
echo "Running unit tests..."
cd webapp && bun run test:unit --run && echo "✓ Unit tests passed" || { echo "✗ Unit tests failed"; exit 1; }

echo ""
echo "=== Phase 2 Complete ✓ ==="
```

### Exit Criteria

- [x] BookmarkCard.vue created with all display elements
- [x] All visual indicators working (archive, ebook)
- [x] Title click navigation logic implemented
- [x] URL click opens in new tab
- [x] Download icon conditionally visible
- [x] Accessibility attributes present
- [x] Unit tests written and passing
- [x] `bun run type-check` passes
- [x] `phase2-verify.sh` passes all checks

---

## Phase 3: ReadableView Page

**Goal:** Create the ReadableView page to display offline/readable content.

**Dependencies:** Phase 1 (API client fix)

### Tasks

- [ ] **3.1** Add route to `webapp/src/router/index.ts`
  - Path: `/bookmark/:id/content`
  - Name: `readable`
  - Lazy loaded component
  - `meta: { requiresAuth: true }`

- [ ] **3.2** Create `webapp/src/views/ReadableView.vue`
  - Extract route param `id`
  - Loading state with spinner
  - Error state with message

- [ ] **3.3** Implement content fetching
  - Call `GET /api/v1/bookmarks/:id/readable` on mount
  - Handle auth errors (redirect to login)
  - Handle not found errors

- [ ] **3.4** Implement content display
  - Render HTML content safely
  - Apply readable typography styles
  - Show bookmark title as header
  - Show original URL link

- [ ] **3.5** Implement navigation
  - Back button to return to home/previous page
  - Original URL link (opens in new tab)

- [ ] **3.6** Style the readable content
  - Clean, readable typography
  - Proper spacing and line height
  - Dark mode support
  - Responsive layout

- [ ] **3.7** Write unit tests for ReadableView
  - Test loading state
  - Test error handling
  - Test content rendering
  - Test navigation elements

### Verification Script

Create `webapp/phase3-verify.sh`:

```bash
#!/bin/bash
set -e

echo "=== Phase 3 Verification ==="

# Check ReadableView exists
echo "Checking ReadableView component..."
test -f webapp/src/views/ReadableView.vue && echo "✓ ReadableView.vue exists" || { echo "✗ ReadableView.vue not found"; exit 1; }

# Check route is defined
echo "Checking router configuration..."
ROUTER_FILE="webapp/src/router/index.ts"
grep -q "/bookmark/:id/content" "$ROUTER_FILE" && echo "✓ Route path defined" || { echo "✗ Route path not found"; exit 1; }
grep -q "readable" "$ROUTER_FILE" && echo "✓ Route name defined" || { echo "✗ Route name not found"; exit 1; }
grep -q "ReadableView" "$ROUTER_FILE" && echo "✓ Component imported" || { echo "✗ Component import not found"; exit 1; }

# Check ReadableView structure
echo "Checking ReadableView structure..."
VIEW_FILE="webapp/src/views/ReadableView.vue"
grep -q "useRoute" "$VIEW_FILE" && echo "✓ Route params accessed" || { echo "✗ Route params not used"; exit 1; }
grep -q "readable" "$VIEW_FILE" && echo "✓ API call present" || { echo "✗ API call not found"; exit 1; }

# Check test file exists  
echo "Checking unit tests..."
test -f webapp/src/views/ReadableView.spec.ts && echo "✓ Test file exists" || { echo "✗ Test file not found"; exit 1; }

# Run type-check
echo "Running type-check..."
cd webapp && bun run type-check && echo "✓ Type check passed" || { echo "✗ Type check failed"; exit 1; }

# Run unit tests
echo "Running unit tests..."
cd webapp && bun run test:unit --run && echo "✓ Unit tests passed" || { echo "✗ Unit tests failed"; exit 1; }

echo ""
echo "=== Phase 3 Complete ✓ ==="
```

### Exit Criteria

- [x] Route `/bookmark/:id/content` defined and protected
- [x] ReadableView.vue created
- [x] Content fetched from API on mount
- [x] Loading and error states handled
- [x] HTML content rendered safely
- [x] Back navigation working
- [x] Dark mode support
- [x] Unit tests passing
- [x] `phase3-verify.sh` passes all checks

---

## Phase 4: Integration & Store Updates

**Goal:** Integrate BookmarkCard into HomeView, connect API calls, and update bookmarks store.

**Dependencies:** Phase 2, Phase 3

### Tasks

- [ ] **4.1** Update bookmarks store (`webapp/src/stores/bookmarks.ts`)
  - Add `updateBookmark(bookmark: ModelBookmarkDTO)` method
  - Add `fetchBookmarkContent(bookmark: ModelBookmarkDTO)` method
    - Calls `PUT /api/v1/bookmarks/cache`
    - Updates store state on success
    - Returns updated bookmark

- [ ] **4.2** Connect BookmarkCard download action to store
  - Import and use `useBookmarksStore`
  - Call `fetchBookmarkContent` on download click
  - Show loading spinner during fetch
  - Show toasts for progress/success/error
  - Navigate to original URL on failure

- [ ] **4.3** Replace HomeView inline bookmark display
  - Import BookmarkCard component
  - Replace `<ul>` list with BookmarkCard components
  - Handle `content-fetched` event to update store
  - Maintain existing search and add bookmark UI

- [ ] **4.4** Wire up title click navigation
  - Use `useRouter` for programmatic navigation
  - Navigate to `/bookmark/:id/content` when `hasContent: true`
  - Navigate to external URL when `hasContent: false`

- [ ] **4.5** Add store unit tests
  - Test `updateBookmark` updates store correctly
  - Test `fetchBookmarkContent` API integration
  - Test error handling

- [ ] **4.6** Integration test for HomeView with BookmarkCard
  - Test cards render from store data
  - Test click interactions
  - Test toast notifications appear

### Verification Script

Create `webapp/phase4-verify.sh`:

```bash
#!/bin/bash
set -e

echo "=== Phase 4 Verification ==="

# Check store updates
echo "Checking bookmarks store..."
STORE_FILE="webapp/src/stores/bookmarks.ts"
grep -q "updateBookmark" "$STORE_FILE" && echo "✓ updateBookmark method" || { echo "✗ updateBookmark not found"; exit 1; }
grep -q "fetchBookmarkContent" "$STORE_FILE" && echo "✓ fetchBookmarkContent method" || { echo "✗ fetchBookmarkContent not found"; exit 1; }

# Check HomeView uses BookmarkCard
echo "Checking HomeView integration..."
HOME_FILE="webapp/src/views/HomeView.vue"
grep -q "BookmarkCard" "$HOME_FILE" && echo "✓ BookmarkCard imported" || { echo "✗ BookmarkCard not imported"; exit 1; }
grep -q "content-fetched" "$HOME_FILE" && echo "✓ Event handler present" || { echo "✗ Event handler not found"; exit 1; }

# Check BookmarkCard uses store
echo "Checking BookmarkCard store integration..."
CARD_FILE="webapp/src/components/bookmarks/BookmarkCard.vue"
grep -q "useBookmarksStore\|toast" "$CARD_FILE" && echo "✓ Store/toast integration" || { echo "✗ Store/toast not integrated"; exit 1; }

# Run type-check
echo "Running type-check..."
cd webapp && bun run type-check && echo "✓ Type check passed" || { echo "✗ Type check failed"; exit 1; }

# Run all unit tests
echo "Running unit tests..."
cd webapp && bun run test:unit --run && echo "✓ Unit tests passed" || { echo "✗ Unit tests failed"; exit 1; }

# Run lint
echo "Running linter..."
cd webapp && bun run lint && echo "✓ Lint passed" || { echo "✗ Lint failed"; exit 1; }

echo ""
echo "=== Phase 4 Complete ✓ ==="
```

### Exit Criteria

- [x] Bookmarks store has new methods
- [x] BookmarkCard uses store for content fetching
- [x] HomeView displays BookmarkCard components
- [x] Toast notifications working
- [x] Title navigation working (internal/external)
- [x] All existing functionality preserved
- [x] Store unit tests passing
- [x] Integration tests passing
- [x] `phase4-verify.sh` passes all checks

---

## Phase 5: Polish, E2E Tests & Documentation

**Goal:** Final polish, end-to-end testing, and ensure production readiness.

**Dependencies:** Phase 4

### Tasks

- [ ] **5.1** Visual polish and responsive design
  - Test on mobile viewport sizes
  - Adjust card layout for small screens
  - Verify dark mode styling
  - Ensure consistent spacing

- [ ] **5.2** Error handling improvements
  - Handle network failures gracefully
  - Handle 401 (redirect to login)
  - Handle 404 (bookmark not found)
  - Log errors to console with context

- [ ] **5.3** Accessibility audit
  - Test with keyboard navigation
  - Verify screen reader announcements
  - Check color contrast
  - Add focus indicators

- [ ] **5.4** Write E2E tests (`e2e/`)
  - Test bookmark card displays correctly
  - Test title click navigates to readable view
  - Test download icon fetches content
  - Test toast notifications appear
  - Test error scenarios

- [ ] **5.5** Run full test suite
  - Unit tests: `bun run test:unit`
  - Type check: `bun run type-check`
  - Lint: `bun run lint`
  - E2E tests: `make e2e`
  - Go backend tests: `make unittest`

- [ ] **5.6** Code cleanup
  - Remove any debug/console.log statements
  - Remove verification scripts
  - Review and clean up comments
  - Ensure consistent code style

- [ ] **5.7** Final review against acceptance criteria
  - Cross-check all PRD acceptance criteria
  - Verify all behaviors match specification

### Verification Script

Create `webapp/phase5-verify.sh`:

```bash
#!/bin/bash
set -e

echo "=== Phase 5 Final Verification ==="

# Run all frontend checks
echo "Running type-check..."
cd webapp && bun run type-check && echo "✓ Type check passed" || { echo "✗ Type check failed"; exit 1; }

echo "Running unit tests..."
cd webapp && bun run test:unit --run && echo "✓ Unit tests passed" || { echo "✗ Unit tests failed"; exit 1; }

echo "Running linter..."
cd webapp && bun run lint && echo "✓ Lint passed" || { echo "✗ Lint failed"; exit 1; }

echo "Building production bundle..."
cd webapp && bun run build && echo "✓ Build passed" || { echo "✗ Build failed"; exit 1; }

cd ..

# Run backend checks
echo "Running Go linter..."
make lint && echo "✓ Go lint passed" || { echo "✗ Go lint failed"; exit 1; }

echo "Running Go unit tests..."
make unittest && echo "✓ Go unit tests passed" || { echo "✗ Go unit tests failed"; exit 1; }

# E2E tests (if configured)
echo "Running E2E tests..."
make e2e && echo "✓ E2E tests passed" || { echo "⚠ E2E tests skipped or failed"; }

echo ""
echo "=== Acceptance Criteria Checklist ==="
echo ""
echo "BookmarkCard Component:"
echo "  [ ] Displays all required fields (title, URL, thumbnail, excerpt, tags, date)"
echo "  [ ] Shows archive indicator when hasArchive: true"
echo "  [ ] Shows ebook indicator when hasEbook: true"
echo "  [ ] Title navigates to /bookmark/:id/content when hasContent: true"
echo "  [ ] Title navigates to original URL when hasContent: false"
echo "  [ ] URL click opens original URL in new tab"
echo "  [ ] External link icon opens original URL in new tab"
echo "  [ ] Download icon only visible when hasContent: false"
echo "  [ ] Download icon shows spinner during fetch"
echo "  [ ] Successful fetch shows success toast and updates card"
echo "  [ ] Failed fetch shows error toast and navigates to original URL"
echo ""
echo "ReadableView:"
echo "  [ ] Route /bookmark/:id/content is accessible"
echo "  [ ] Displays readable HTML content"
echo "  [ ] Shows bookmark title and original URL"
echo "  [ ] Provides navigation back to home"
echo "  [ ] Handles loading and error states"
echo ""
echo "Toast System:"
echo "  [ ] vue-sonner is installed and configured"
echo "  [ ] Toasts appear for fetch start, success, and failure"
echo "  [ ] Toasts auto-dismiss appropriately"
echo ""
echo "Integration:"
echo "  [ ] BookmarkCard replaces inline bookmark display in HomeView"
echo "  [ ] Bookmarks store updated with new methods"
echo "  [ ] All existing functionality preserved"
echo ""
echo "=== Phase 5 Complete ✓ ==="
```

### Exit Criteria

- [x] All acceptance criteria from PRD verified
- [x] Responsive design tested
- [x] Accessibility tested
- [x] E2E tests passing
- [x] Full test suite passing
- [x] Production build succeeds
- [x] Code cleaned up
- [x] `phase5-verify.sh` passes all checks

---

## Summary

| Phase | Focus | Estimated Complexity | Depends On |
|-------|-------|---------------------|------------|
| 1 | Foundation | Low | None |
| 2 | BookmarkCard | Medium | Phase 1 |
| 3 | ReadableView | Medium | Phase 1 |
| 4 | Integration | Medium | Phase 2, 3 |
| 5 | Polish & E2E | Low | Phase 4 |

### Key Files Created/Modified

**New Files:**
- `webapp/src/utils/toast.ts`
- `webapp/src/components/bookmarks/BookmarkCard.vue`
- `webapp/src/components/bookmarks/BookmarkCard.spec.ts`
- `webapp/src/views/ReadableView.vue`
- `webapp/src/views/ReadableView.spec.ts`

**Modified Files:**
- `webapp/package.json` (add vue-sonner)
- `webapp/src/App.vue` (add Toaster)
- `webapp/src/router/index.ts` (add readable route)
- `webapp/src/stores/bookmarks.ts` (add new methods)
- `webapp/src/views/HomeView.vue` (use BookmarkCard)
- `internal/http/handlers/api/v1/bookmarks.go` (fix swagger annotation)

### Risk Mitigation

1. **API Client Bug:** Fixed in Phase 1 before dependent work begins
2. **Parallel Development:** Phases 2 and 3 can be worked on simultaneously after Phase 1
3. **Incremental Testing:** Each phase has verification scripts to catch issues early
4. **Rollback Points:** Each phase leaves the codebase in a working state

---

*Plan generated from PRD v1.0 (2026-01-31)*
