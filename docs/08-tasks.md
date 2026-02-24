# BookOwl — Implementation Tasks

## Phase 1: Foundation (Weeks 1–2)

### 1.1 Project Scaffolding
- [x] Initialize Go module (`github.com/wisbric/bookowl`)
- [x] Set up project structure (`pkg/`, `cmd/`, `internal/`, `migrations/`)
- [x] Configure `sqlc` with PostgreSQL schema (`sqlc.yaml`)
- [x] Set up `golang-migrate` migration framework
- [x] Docker build (multi-stage: build → distroless)
- [x] Frontend scaffold (Vite + React 19 + TypeScript + Tailwind + shadcn/ui)
- [x] Makefile (build, test, lint, sqlc, seed, docker)
- [ ] CI: GitHub Actions (ci.yml + release.yml)
- [ ] Helm chart skeleton (`deploy/helm/bookowl/`)

### 1.2 Database & Multi-Tenancy
- [x] Global schema migrations (tenants, api_keys)
- [x] Tenant schema migrations (users, spaces, space_members, collections, documents, document_versions, audit_log)
- [x] FTS trigger on documents (`update_document_search_vector`)
- [x] Tenant provisioning: create schema + run migrations
- [x] `search_path` middleware: resolve tenant → set schema
- [x] Image storage migration (see `docs/09-image-storage.md`): `document_images` table + `storage_objects` table
- [x] Trigram search migration (`migrations/tenant/000010_add_trigram_search`)

### 1.3 Authentication & RBAC
- [x] OIDC middleware: validate JWT, extract claims
- [x] API key middleware: SHA-256 hash lookup
- [x] RBAC middleware: admin / editor / viewer roles
- [x] Dev fallback: `X-Tenant-Slug` header (dev only, grants admin)
- [x] Health endpoints: `/healthz`, `/readyz`

### 1.4 Core API Framework
- [x] Chi router with middleware chain (RequestID, logging, recovery, CORS, metrics, auth, tenant)
- [x] `httpserver.Respond()` and `httpserver.RespondError()`
- [x] Pagination helper (offset-based)
- [x] Prometheus metrics middleware

---

## Phase 2: Document Management (Weeks 3–4)

### 2.1 Spaces
- [x] `POST /api/v1/spaces` — create with name, slug, description, icon, is_private
- [x] `GET /api/v1/spaces` — list all spaces the user has access to
- [x] `GET /api/v1/spaces/:id` — detail
- [x] `PUT /api/v1/spaces/:id` — update
- [x] `DELETE /api/v1/spaces/:id` — archive (soft delete)
- [x] `GET /api/v1/spaces/:id/tree` — full tree: collections + document titles + icons, positions
- [x] Space membership: `GET/POST/DELETE /api/v1/spaces/:id/members`
- [x] Auto-add creator as space admin on creation

### 2.2 Collections
- [x] `POST /api/v1/spaces/:spaceId/collections` — create with name, slug, icon, parent_id, position
- [x] `GET /api/v1/spaces/:spaceId/collections` — list
- [x] `GET /api/v1/spaces/:spaceId/collections/:id` — detail
- [x] `PUT /api/v1/spaces/:spaceId/collections/:id` — update including reorder (position)
- [x] `DELETE /api/v1/spaces/:spaceId/collections/:id` — delete (cascade documents)

### 2.3 Documents CRUD
- [x] `POST /api/v1/documents` — create with space_id, collection_id, title, doc_type, content
- [x] `GET /api/v1/documents` — list with filters: space_id, collection_id, doc_type, status, tags
- [x] `GET /api/v1/documents/:id` — detail (full content JSON)
- [x] `PUT /api/v1/documents/:id` — update content + metadata; auto-saves a new version snapshot
- [x] `DELETE /api/v1/documents/:id` — soft delete (set status=archived)
- [x] Extract `content_text` from Tiptap JSON on every save (walk all text nodes)
- [x] Increment `version` counter on every save
- [x] Set `word_count` from extracted text on every save

### 2.4 Version History
- [x] Auto-snapshot on every `PUT /api/v1/documents/:id`: insert into `document_versions`
- [x] `GET /api/v1/documents/:id/history` — list versions (id, version #, changed_by, change_summary, created_at)
- [x] `GET /api/v1/documents/:id/history/:versionId` — fetch full content snapshot
- [x] `POST /api/v1/documents/:id/restore/:versionId` — restore version as new current (increments version, creates a new snapshot)

### 2.5 Search
- [x] `GET /api/v1/search?q=<text>&space_id=<id>&doc_type=<type>&limit=20&offset=0`
- [x] PostgreSQL FTS: `ts_rank` scoring, `ts_headline` highlighting for title and content_text
- [x] Return: id, title, doc_type, space_name, collection_name, title_highlight, content_highlight, rank
- [x] Filter to `status != 'archived'`

### 2.6 Audit Logging
- [x] Async buffered writer (same pattern as NightOwl: channel cap 256, flush at 32 or 2s)
- [x] Log: document create, update, delete, restore, publish, archive
- [x] Log: space create, update, delete
- [ ] `GET /api/v1/audit-log` with filtering (resource, user, date range)

---

## Phase 3: Image Storage (Week 5)

See `docs/09-image-storage.md` for the full spec.

- [x] Add image storage migrations (`document_images`, `storage_objects`)
- [x] Implement storage backend interface (`pkg/storage/storage.go`)
- [x] Implement local filesystem backend (dev) — `pkg/storage/local.go`
- [x] Implement S3-compatible backend (production) — `pkg/storage/s3.go`
- [x] `POST /api/v1/images` — upload image, return URL
- [x] `GET /api/v1/images/:id` — serve image (local backend) or redirect (S3 backend)
- [x] `DELETE /api/v1/images/:id` — delete image (editor removes image block → client calls this)
- [x] Wire Tiptap `Image` extension to upload endpoint in the frontend
- [x] Orphan cleanup: background job (weekly) removes images not referenced in any document

---

## Phase 4: NightOwl Integration (Week 6)

### 4.1 Live Context Proxy
- [x] NightOwl REST client (`pkg/livecontext/client.go`) — reads `nightowl_api_url` + `nightowl_api_key` from tenant config
- [x] Redis cache layer (30s freshness / 5m stale TTL)
- [x] `GET /api/v1/live-context/oncall/:rosterId`
- [x] `GET /api/v1/live-context/service/:serviceName`
- [x] `GET /api/v1/live-context/alerts?severity=<sev>&service=<n>&limit=5`
- [x] `GET /api/v1/live-context/incident/:nightowlIncidentId`
- [x] Graceful degradation: return stale cache + `"source": "stale"` if NightOwl is unreachable
- [ ] Metrics: `bookowl_livecontext_cache_hits_total{subtype}`, `bookowl_livecontext_nightowl_errors_total`

### 4.2 Integration Endpoints (for NightOwl to call)
- [x] `GET /api/v1/integration/runbooks` — list published runbook documents
- [x] `GET /api/v1/integration/runbooks/:id` — fetch with `content_html` (render Tiptap JSON → HTML)
- [x] `POST /api/v1/integration/post-mortems` — create post-mortem from template + incident data
- [x] `GET /api/v1/integration/search?q=<text>&type=runbook` — search for runbook linking
- [x] All integration endpoints require API key with `role=admin`
- [x] Tiptap JSON → HTML renderer (`internal/integration/postmortem.go`) — walk JSON nodes, produce clean HTML

### 4.3 Admin Config
- [x] `GET /api/v1/admin/config` — includes NightOwl connection settings
- [x] `PUT /api/v1/admin/config` — update NightOwl API URL + key
- [x] `POST /api/v1/admin/config/test-nightowl` — ping NightOwl, return connectivity status

### 4.4 Runbook Migration Command
- [x] `--mode=migrate-nightowl-runbooks` binary mode
- [x] Fetch all runbooks from NightOwl `GET /api/v1/runbooks` (paginated)
- [x] Convert markdown → Tiptap JSON
- [x] Create documents with `doc_type=runbook`, `status=published`
- [x] Idempotent: check if document with same NightOwl runbook ID already exists before creating
- [x] Log migration summary on completion

---

## Phase 5: Frontend — Layout & Navigation (Week 7)

### 5.1 Setup
- [x] Vite + React 19 + TypeScript scaffold
- [x] Tailwind CSS 4 + shadcn/ui with NightOwl design tokens
- [x] TanStack Router with route definitions
- [x] TanStack Query client
- [x] Dark mode default with theme toggle
- [x] Inter + JetBrains Mono fonts (self-hosted in `web/public/fonts/`)

### 5.2 Layout & Sidebar
- [x] App shell: 320px sidebar + main content area
- [x] Sidebar: space tree with expand/collapse per space and collection
- [x] Sidebar tree fetched from `GET /api/v1/spaces/:id/tree` per space
- [x] Active document highlighted in tree
- [x] Sidebar search shortcut: `Cmd/Ctrl+K` opens command palette
- [x] "New Document" button at top of each collection
- [x] Sidebar footer: Admin link, NightOwl link, theme toggle
- [ ] Collapsed sidebar (icon-only) on narrow screens

### 5.3 Command Palette (Cmd+K)
- [x] Modal search overlay
- [x] Searches `GET /api/v1/search` as user types (debounced 300ms)
- [x] Shows: title highlight, space/collection path, doc type badge
- [x] Keyboard navigable (arrow keys, Enter to open)
- [ ] Recent documents section when query is empty

---

## Phase 6: Frontend — Editor (Week 8)

### 6.1 Tiptap Setup
- [x] Install Tiptap packages (see `docs/04-editor.md` for full list)
- [x] Configure editor with all extensions
- [x] Autosave: debounced 1s, `PUT /api/v1/documents/:id`
- [x] "Saving…" / "Saved" indicator in document header

### 6.2 Slash Command Menu
- [x] `/` triggers block picker overlay
- [x] Categories: Basic, Media & Content, Callouts, NightOwl (Live)
- [x] Keyboard navigation + fuzzy filter

### 6.3 Custom Extensions
- [x] `CalloutBlock` — custom node with `type` attr (info/warning/danger)
- [x] `LiveContextBlock` — custom node with `subtype`, `rosterId`, `serviceName` attrs
  - [x] View mode: fetches from `/api/v1/live-context/...`, auto-refreshes every 30s
  - [x] Loading state: shows loading text
  - [x] Error state: "Unavailable" with source info
  - [ ] Edit mode: config popover on click (roster picker, service name input)

### 6.4 Image Upload
- [x] Tiptap `Image` extension wired to `POST /api/v1/images`
- [x] Drag-and-drop onto editor uploads image and inserts block
- [x] Paste image from clipboard uploads and inserts
- [ ] Progress indicator during upload

### 6.5 Document Header
- [x] Breadcrumb: Space → Document
- [x] Doc type badge with color (runbook=purple, post-mortem=red, etc.)
- [x] Status badge (draft/published/archived)
- [x] Version info: "v4 · Updated 2h ago"
- [x] Edit / View mode toggle button
- [ ] Document title (editable inline, updates on blur)
- [ ] `⋯` menu: Version History, Duplicate, Move, Archive, Delete

---

## Phase 7: Frontend — Views (Week 9)

### 7.1 Document View
- [x] Read-only rendered Tiptap content (view mode)
- [ ] Task list checkboxes interactive in view mode (check off runbook steps during incident)
- [x] Syntax-highlighted code blocks

### 7.2 Version History Panel
- [x] Side panel: list all versions with author + timestamp
- [x] Click version to preview content snapshot (read-only)
- [x] "Restore this version" button

### 7.3 Space & Collection Management
- [x] Space list page (sidebar lists all spaces)
- [x] Create space dialog: name, slug (auto-generated), description, icon
- [x] Create document dialog per collection in sidebar tree
- [ ] Drag-and-drop reordering of collections and documents (updates `position`)

### 7.4 Admin Pages
- [x] NightOwl connection settings: URL + API key input, "Test Connection" button
- [x] API key info section
- [ ] Image storage settings (see `docs/09-image-storage.md`): provider, bucket/path config
- [ ] Audit log viewer: filterable table

### 7.5 System Status Page
- [x] DB health, Redis health (via /health endpoint)
- [x] NightOwl connectivity status
- [ ] Image storage connectivity status
- [ ] Uptime, version, commit SHA

---

## Phase 8: Deployment & Operations (Week 10)

### 8.1 Container Images
- [x] `Dockerfile` (backend, distroless)
- [x] `web/Dockerfile` (frontend, nginx)
- [x] `web/nginx.conf` (SPA routing + API proxy)
- [x] `docker-compose.yml` (dev)
- [x] `docker-compose.demo.yml` (full-stack demo)
- [x] `make docker` and `make docker-web` both build successfully

### 8.2 Helm Chart
- [x] `deploy/helm/bookowl/Chart.yaml`
- [x] `deploy/helm/bookowl/values.yaml`
- [x] All templates: deployment-api, deployment-web, services, ingress, configmap, secret, serviceaccount, servicemonitor, pdb
- [x] `helm lint deploy/helm/bookowl/` passes

### 8.3 CI/CD
- [x] `.github/workflows/ci.yml` — backend test + lint + build, frontend lint + typecheck + build
- [x] `.github/workflows/release.yml` — build + push both images to GHCR

### 8.4 Seed Data
- [x] `make seed` — creates "acme" tenant, 3 users (Stefan, Max, Anna), seed API key `bw_dev_seed_key_do_not_use_in_production`
- [x] `make seed-demo` — full demo data:
  - Space: "Platform Engineering" with collections: Kubernetes, Post-mortems, Architecture
  - Space: "On-Call Runbooks" with collection: Alerts
  - 6 runbook documents (pod crashloop, OOM, cert expiry, node not ready, PVC stuck, etcd)
  - 2 post-mortem documents
  - 1 document with Live Context blocks configured
  - All documents `status=published`

### 8.5 Testing
- [ ] Integration tests for document CRUD with real PostgreSQL (testcontainers)
- [ ] Integration tests for FTS search
- [x] Unit tests for Tiptap JSON → plain text extraction
- [x] Unit tests for Tiptap JSON → HTML rendering (integration endpoint)
- [x] Unit tests for live context cache key functions
- [x] `make test` passes (8 packages, 0 failures)
- [x] `make lint` passes (0 issues)

---

## Phase 9: Image Storage (docs/09-image-storage.md)

- [x] Add image storage migrations (`document_images`, `storage_objects`)
- [x] Implement storage backend interface (`pkg/storage/storage.go`)
- [x] Implement local filesystem backend (dev) — `pkg/storage/local.go`
- [x] Implement S3-compatible backend (production) — `pkg/storage/s3.go`
- [x] `POST /api/v1/images` — upload image, return URL
- [x] `GET /api/v1/images/:id` — serve image (local) or redirect (S3)
- [x] `DELETE /api/v1/images/:id` — delete image
- [x] Wire Tiptap `Image` extension to upload endpoint in the frontend
- [x] Orphan cleanup: background job removes unreferenced images

---

## Phase 10: Diagrams (docs/10-diagrams.md)

- [x] `DiagramBlock` custom Tiptap extension (`web/src/components/editor/extensions/DiagramBlock.ts`)
- [x] `DiagramBlockView` React node view with inline draw.io editor (`DiagramBlockView.tsx`)
- [x] `DiagramModal` full-screen editor modal (`DiagramModal.tsx`)
- [x] Slash command entry: `/diagram` inserts a diagram block
- [x] Diagrams stored as SVG/XML within Tiptap JSON (no separate storage)

---

## Phase 11: OIDC Admin (docs/11-oidc-admin.md)

- [x] `GET /api/v1/admin/oidc` — get OIDC configuration
- [x] `PUT /api/v1/admin/oidc` — save OIDC settings (issuer, client ID, client secret, redirect URI)
- [x] `POST /api/v1/admin/oidc/test` — test OIDC connectivity
- [x] Admin UI: OIDC configuration form in admin page (`internal/admin/oidc.go`)
- [x] Unit tests for OIDC config handlers (`internal/admin/oidc_test.go`)

---

## Phase 12: Login & Profile (docs/12-login-and-profile.md)

### 12.1 Authentication
- [x] Local admin table and migrations (`migrations/global/000003_create_local_admins`)
- [x] `POST /auth/local` — local admin login with session cookie
- [x] `POST /auth/logout` — destroy session
- [x] `POST /auth/change-password` — forced password change on first login
- [x] JWT session management (`internal/session/`)
- [x] Auth handler package (`internal/authhandler/`)
- [x] Rate limiting on local login (429 with `retry_after`)
- [x] Auth middleware updated to support session cookies (`internal/auth/auth.go`)
- [x] Unit tests for auth middleware (`internal/auth/auth_test.go`)

### 12.2 User Profile
- [x] `GET /api/v1/profile` — current user profile
- [x] `PUT /api/v1/profile` — update display name, job title, avatar URL
- [x] User profile fields migration (`migrations/tenant/000011_add_user_profile_fields`)
- [x] Profile page (`web/src/routes/profile.tsx`)
- [x] `UserAvatar` component (`web/src/components/UserAvatar.tsx`)

### 12.3 Personal API Tokens
- [x] `GET /api/v1/profile/tokens` — list user's personal tokens
- [x] `POST /api/v1/profile/tokens` — create personal token
- [x] `DELETE /api/v1/profile/tokens/:id` — revoke token
- [x] Personal tokens page (`web/src/routes/profile.tokens.tsx`)

### 12.4 Frontend
- [x] Login page (`web/src/routes/login.tsx`) — OIDC button + local admin form
- [x] OIDC callback page (`web/src/routes/callback.tsx`)
- [x] Change password page (`web/src/routes/change-password.tsx`)
- [x] Auth provider with OIDC + session support (`web/src/auth/`)
- [x] User menu in sidebar with profile, tokens, admin, sign out links

---

## Phase 13: Export (docs/13-export.md)

- [ ] `GET /api/v1/documents/:id/export?format=markdown` — export as Markdown
- [ ] `GET /api/v1/documents/:id/export?format=html` — export as HTML
- [ ] `GET /api/v1/documents/:id/export?format=pdf` — export as PDF
- [ ] Tiptap JSON → Markdown converter
- [ ] Tiptap JSON → HTML converter (reuse integration renderer)
- [ ] PDF generation (HTML → PDF via headless browser or wkhtmltopdf)
- [ ] Frontend: export button in document header `⋯` menu

---

## Phase 14: Comments & Notifications (docs/14-comments.md)

### 14.1 Backend
- [x] Comments migration (`migrations/tenant/000013_create_comments`)
- [x] Notifications migration (`migrations/tenant/000014_create_notifications`)
- [x] sqlc queries: comments (`internal/db/tenant/query/comments.sql`)
- [x] sqlc queries: notifications (`internal/db/tenant/query/notifications.sql`)
- [x] sqlc query: `GetUsersByEmails` added to users.sql for @mention resolution
- [x] `pkg/comment/` — renderer, service, store, handler, types, tests
  - [x] Markdown-lite renderer (bold, italic, code, @mentions, URLs)
  - [x] Two-level threading (top-level + replies, no infinite nesting)
  - [x] 15-minute edit window (admin bypasses)
  - [x] Soft-delete (body replaced with `[deleted]`)
  - [x] Resolve/unresolve top-level comments
  - [x] @mention extraction → notification creation
- [x] `pkg/notification/` — service, store, handler, types, tests
  - [x] Unread count endpoint
  - [x] List with actor info
  - [x] Mark read (individual + all)
- [x] Wired in `internal/app/app.go`

### 14.2 Frontend
- [x] `CommentPanel` component — right panel with threaded comments
- [x] `NotificationBell` component — bell icon with unread count badge, 60s polling
- [x] Comment types added to `web/src/api/client.ts`
- [x] Document page updated with comment toggle button + panel
- [x] Sidebar updated with notification bell

---

## Phase 15: Templates (docs/15-templates.md)

- [x] Templates migration (`migrations/tenant/000012_create_templates`)
- [x] sqlc queries: templates (`internal/db/tenant/query/templates.sql`)
- [x] `pkg/template/` — service, store, handler, types
- [x] System templates seeded (`internal/seed/templates/`)
- [x] `GET /api/v1/templates` — list templates
- [x] `POST /api/v1/templates` — create user template
- [x] `PUT /api/v1/templates/:id` — update template
- [x] `DELETE /api/v1/templates/:id` — delete template
- [x] `POST /api/v1/documents/:id/save-as-template` — save document as template
- [x] Templates page (`web/src/routes/templates.tsx`)
- [x] Create document dialog: template picker

---

## Implementation Notes

### What to implement first
Phase 1 (foundation) and Phase 2 (document management) are the critical path. Everything else depends on having working CRUD and search. Don't start Phase 5 (frontend) until the full backend API is working and tested.

### Patterns to follow
Always follow NightOwl conventions exactly. When in doubt, look at how NightOwl implements the equivalent feature. The codebases should feel like they were written by the same person.

### Document content flow
Every document save goes through this sequence:
1. Receive Tiptap JSON in request body
2. Extract plain text (walk all text nodes recursively)
3. Count words
4. Set `content_text`, `word_count` on the document record
5. FTS trigger fires on INSERT/UPDATE, rebuilds `search_vector`
6. Insert version snapshot into `document_versions`
7. Return updated document

Never store HTML — always store Tiptap JSON and render to HTML on demand (for the integration endpoint). This keeps the data clean and format-agnostic.
