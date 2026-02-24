# BookOwl — Document Comments

## 1. Overview

Users can leave threaded comments on documents. Comments are attached to the document as a whole (not inline/margin comments anchored to specific text). This covers the primary use cases:

- Annotating a post-mortem with follow-up notes after review
- Flagging a runbook step as outdated without editing the document
- Discussing a proposed change to an SOP before editing
- Acknowledging that you've read and reviewed a document

Inline/margin comments (anchored to specific paragraphs) are out of scope for v1 — they require significant Tiptap integration and are a v1.1 feature.

---

## 2. Data Model

```sql
-- migrations/000010_create_comments.up.sql

CREATE TABLE comments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    parent_id       UUID REFERENCES comments(id) ON DELETE CASCADE,  -- NULL = top-level
    author_id       UUID NOT NULL REFERENCES users(id),
    body            TEXT NOT NULL,                -- plain text, max 4000 chars
    body_rendered   TEXT NOT NULL,               -- HTML rendered from body (Markdown-lite)
    is_resolved     BOOLEAN NOT NULL DEFAULT false,
    resolved_by     UUID REFERENCES users(id),
    resolved_at     TIMESTAMPTZ,
    edited_at       TIMESTAMPTZ,                 -- set if body has been edited
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comments_document ON comments(document_id);
CREATE INDEX idx_comments_parent ON comments(parent_id);
CREATE INDEX idx_comments_author ON comments(author_id);

-- Only one level of nesting (reply to top-level, not reply to reply)
-- Enforced via CHECK or application logic
```

**Threading model:** Two levels only — top-level comments and direct replies. No infinite nesting. A reply cannot itself have replies. This keeps the UI simple and avoids deep thread management.

**Resolved state:** Top-level comments can be marked resolved (collapsed in the UI). Replies cannot be individually resolved. Resolved = "this discussion is closed / actioned".

---

## 3. Comment Body Format

Comments use a lightweight Markdown subset, not full Tiptap:

- `**bold**` and `_italic_`
- `` `inline code` ``
- ` ```code block``` ` (fenced, single language)
- `@username` mentions (renders as a link, sends notification — see section 6)
- URLs are auto-linked
- No headings, no tables, no images, no block elements

This is intentional. Comments are for discussion, not documentation. Keep them lightweight.

Server-side rendering: `body` is stored as raw text, `body_rendered` is stored as sanitised HTML generated on write. Rendering happens once on save, not on every read.

Sanitisation: strip all HTML tags except `<strong>`, `<em>`, `<code>`, `<pre>`, `<a href>`, `<br>`. Never trust client-submitted HTML.

---

## 4. API Endpoints

```
GET  /api/v1/documents/:id/comments
     → Returns all comments for the document, nested (replies under their parent)
     → Default: exclude resolved threads (include resolved if ?include_resolved=true)
     → Sorted: top-level by created_at ASC, replies by created_at ASC within thread

POST /api/v1/documents/:id/comments
     → Create a top-level comment
     → Body: { "body": "text" }
     → Returns created comment with body_rendered

POST /api/v1/documents/:id/comments/:commentId/replies
     → Reply to a top-level comment
     → Body: { "body": "text" }
     → Returns 400 if commentId is itself a reply (no nesting beyond 1 level)

PUT  /api/v1/documents/:id/comments/:commentId
     → Edit own comment body (within 15 minutes of creation, then locked)
     → Body: { "body": "text" }
     → Sets edited_at = now()
     → Admins can edit any comment at any time

DELETE /api/v1/documents/:id/comments/:commentId
     → Soft-delete: replaces body with "[comment deleted]", clears author reference
     → Keeps the record so replies remain coherent
     → Only the author or an admin can delete

POST /api/v1/documents/:id/comments/:commentId/resolve
     → Mark a top-level comment thread as resolved
     → Sets is_resolved=true, resolved_by, resolved_at
     → Requires editor role or above (or comment author)
     → Returns 400 if commentId is a reply

POST /api/v1/documents/:id/comments/:commentId/unresolve
     → Reopen a resolved thread
     → Requires editor role or above
```

### 4.1 Response Shape

```json
{
  "id": "uuid",
  "document_id": "uuid",
  "parent_id": null,
  "author": {
    "id": "uuid",
    "display_name": "Stefan Kloppenborg",
    "initials": "SK",
    "avatar_url": null
  },
  "body": "The escalation path in step 3 is outdated — we now use PagerDuty.",
  "body_rendered": "<p>The escalation path in step 3 is outdated — we now use PagerDuty.</p>",
  "is_resolved": false,
  "edited_at": null,
  "created_at": "2026-02-24T11:30:00Z",
  "replies": [
    {
      "id": "uuid",
      "parent_id": "uuid",
      "author": { ... },
      "body": "Updated in v5, see the runbook update I just made.",
      "body_rendered": "...",
      "created_at": "2026-02-24T14:00:00Z"
    }
  ]
}
```

---

## 5. Frontend

### 5.1 Comment Panel

Comments live in a right-side panel, toggled from the document header. The panel is 320px wide and overlays the content (does not push the editor).

```
┌─────────────────────────────────────────┐
│  Comments (3)                      [✕]  │
├─────────────────────────────────────────┤
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ SK  Stefan Kloppenborg          │   │
│  │     Feb 24, 11:30               │   │
│  │                                 │   │
│  │  The escalation path in step 3  │   │
│  │  is outdated — we now use       │   │
│  │  PagerDuty.                     │   │
│  │                                 │   │
│  │  [ Reply ]  [ Resolve ]         │   │
│  │                                 │   │
│  │  ┌───────────────────────────┐ │   │
│  │  │ MK  Max Keller · 2h ago   │ │   │
│  │  │  Updated in v5, see the   │ │   │
│  │  │  runbook update I made.   │ │   │
│  │  └───────────────────────────┘ │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ ✓ Resolved · 1 reply    [Open]  │   │ ← collapsed resolved thread
│  └─────────────────────────────────┘   │
│                                         │
├─────────────────────────────────────────┤
│  Add a comment...                       │
│  ┌─────────────────────────────────┐   │
│  │                                 │   │
│  │                                 │   │
│  └─────────────────────────────────┘   │
│  Markdown supported · @mention users   │
│  [ Post comment ]                       │
└─────────────────────────────────────────┘
```

### 5.2 Comment Count Badge

The document header shows a comment count badge on the Comments toggle button:

```
[Edit]  [💬 3]  [⋯]
```

Badge updates optimistically when a new comment is posted.

### 5.3 Reply Flow

Clicking "Reply" on a top-level comment:
1. Focuses the reply input that appears inline below the comment thread
2. Reply input is a smaller text area (not the full add-comment box at the bottom)
3. `POST /api/v1/documents/:id/comments/:commentId/replies`
4. New reply appears immediately (optimistic insert)

### 5.4 Resolved Threads

Resolved threads collapse to a single row showing:
- ✓ icon + "Resolved"
- Number of replies
- "Open" link to expand

When expanded, the thread shows with a muted/greyed style to distinguish from active threads.

### 5.5 Edit Window

Within 15 minutes of posting, an "Edit" link appears on the user's own comments. After 15 minutes it disappears (server also enforces this). Edited comments show "(edited)" after the timestamp.

### 5.6 @mentions Autocomplete

Typing `@` in the comment input opens a small dropdown listing space members (fetched from `GET /api/v1/spaces/:id/members`). Selecting a user inserts `@username`. The rendered HTML links to the user's profile.

---

## 6. Notifications

When a user is @mentioned in a comment, or when someone replies to their comment, a notification is created.

### 6.1 Data Model

```sql
CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    type            TEXT NOT NULL,   -- 'comment_mention' | 'comment_reply' | 'document_comment'
    document_id     UUID REFERENCES documents(id),
    comment_id      UUID REFERENCES comments(id),
    actor_id        UUID REFERENCES users(id),  -- who triggered the notification
    is_read         BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user ON notifications(user_id, is_read);
```

Notification types:
- `comment_mention` — someone @mentioned you in a comment
- `comment_reply` — someone replied to your comment
- `document_comment` — someone commented on a document you authored (optional, user-configurable)

### 6.2 Notification Bell

The app header shows a notification bell icon next to the user avatar:

```
[ 🔔 2 ]  [ SK ▾ ]
```

Clicking opens a dropdown of recent unread notifications:

```
┌────────────────────────────────────────┐
│  Notifications                         │
├────────────────────────────────────────┤
│  💬 Max Keller replied to your comment  │
│     Pod CrashLoopBackOff Runbook        │
│     2h ago                              │
├────────────────────────────────────────┤
│  💬 Anna W. mentioned you              │
│     Post-Mortem: DB Outage Feb 2026     │
│     Yesterday                           │
├────────────────────────────────────────┤
│  [ Mark all as read ]  [ See all ]      │
└────────────────────────────────────────┘
```

### 6.3 Notification Delivery

v1: In-app notifications only (the bell in the header). No email, no Slack push.

v1.1: Email notifications (one email per mention/reply, not batched). Respects user notification preferences.

### 6.4 Polling

The frontend polls `GET /api/v1/notifications/unread-count` every 60 seconds to update the bell badge. This is intentionally not a WebSocket — polling is simpler, sufficient for notification latency, and easier to reason about in a multi-replica deployment.

```
GET /api/v1/notifications/unread-count   → { "count": 2 }
GET /api/v1/notifications                → list with pagination
POST /api/v1/notifications/mark-read     → { "ids": ["uuid", ...] } or { "all": true }
```

---

## 7. Backend Implementation Notes

### 7.1 Package Structure

```
pkg/comment/
├── comment.go      — Comment, Reply types
├── handler.go      — HTTP handlers
├── service.go      — business logic, @mention extraction
├── store.go        — sqlc queries
└── renderer.go     — Markdown-lite → HTML for comment bodies

pkg/notification/
├── notification.go
├── handler.go
├── service.go
└── store.go
```

### 7.2 @mention Extraction

The comment service extracts @mentions from the body text after saving and creates notification records:

```go
func extractMentions(body string) []string {
    // Match @username patterns (letters, numbers, dots, underscores, hyphens)
    re := regexp.MustCompile(`@([a-zA-Z0-9._-]+)`)
    matches := re.FindAllStringSubmatch(body, -1)
    usernames := make([]string, 0, len(matches))
    for _, m := range matches {
        usernames = append(usernames, m[1])
    }
    return usernames
}
```

Usernames are resolved against `users.username` (or `preferred_username` from OIDC). Non-existent usernames are silently ignored — no error, just not linked.

### 7.3 Edit Enforcement

The 15-minute edit window is enforced server-side:

```go
if time.Since(comment.CreatedAt) > 15*time.Minute && !isAdmin {
    return httpserver.RespondError(w, http.StatusForbidden, "edit window has expired")
}
```

### 7.4 Audit Log

Log these events:
- `comment.created` — document_id, comment_id, author_id
- `comment.edited` — comment_id, author_id
- `comment.deleted` — comment_id, author_id
- `comment.resolved` — comment_id, resolved_by
- `comment.replied` — parent_id, reply_id, author_id

---

## 8. Tasks

Add to `docs/08-tasks.md` as Phase 14:

### Phase 14: Comments

**Backend:**
- [ ] Migration: `comments` table
- [ ] Migration: `notifications` table
- [ ] `pkg/comment` package: comment.go, handler.go, service.go, store.go, renderer.go
- [ ] `pkg/notification` package
- [ ] Markdown-lite renderer for comment bodies (bold, italic, inline code, code block, auto-link, @mention link)
- [ ] HTML sanitiser for body_rendered
- [ ] @mention extraction → notification creation
- [ ] All comment CRUD endpoints
- [ ] Resolve / unresolve endpoints
- [ ] 15-minute edit window enforcement
- [ ] Soft-delete (replace body with "[comment deleted]")
- [ ] Notification endpoints (count, list, mark-read)
- [ ] Audit log entries for all comment actions
- [ ] Unit tests: Markdown renderer, @mention extraction, edit window logic
- [ ] Unit tests: reply-to-reply returns 400

**Frontend:**
- [ ] Comment panel component (320px right panel, toggle from header)
- [ ] Comment count badge in document header
- [ ] Top-level comment display with author avatar, timestamp, edit/delete/reply/resolve actions
- [ ] Reply thread display (indented, muted)
- [ ] Resolved thread collapsed state with expand toggle
- [ ] Add comment textarea with Markdown-lite hints
- [ ] Reply inline input
- [ ] @mention autocomplete (fetch space members on `@` keypress)
- [ ] Edit comment inline (within 15 min, shows "(edited)" after)
- [ ] Notification bell in app header with unread count badge
- [ ] Notification dropdown (recent unread, mark all read)
- [ ] 60-second polling for unread count
- [ ] Optimistic UI: new comment appears immediately before API confirms
