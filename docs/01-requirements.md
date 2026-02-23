# BookOwl — Product Requirements

> General Knowledge Management Platform with Deep NightOwl Integration

## 1. Problem Statement

NightOwl handles incident alerts and on-call management, but operational knowledge is scattered:

- Runbooks live inline as plain markdown blobs in NightOwl — no versioning, no rich editing, no cross-linking
- Post-mortems are either not written or live in external tools (Confluence, Notion) disconnected from the incident record
- General team documentation (architecture docs, SOPs, onboarding guides) has no home in the platform
- Engineers switch tools during incidents to find documentation — context breaks, time is lost
- KRITIS/public sector clients require a self-hosted documentation platform with audit trails

BookOwl solves this by being the documentation pillar of the NightOwl platform: a self-hosted, deeply integrated knowledge base that covers everything from runbooks to general team wikis.

## 2. Target Users

| Role | Description |
|------|-------------|
| **On-call engineer** | Opens runbooks during incidents, writes post-mortems after resolution |
| **SRE / Team Lead** | Authors SOPs, architecture docs, onboarding guides |
| **NightOwl (API)** | Fetches runbook content when an alert fires, creates post-mortems on incident close |
| **Platform admin** | Manages spaces, permissions, API keys |

## 3. Document Model

BookOwl uses a three-level hierarchy:

```
Space (e.g., "Platform Engineering", "Customer Runbooks")
└── Collection (e.g., "Kubernetes", "Post-mortems")
    └── Document (the actual content)
```

Documents contain **blocks** — structured content units (paragraphs, headings, code, checklists, callouts, live context, etc.) edited with Tiptap.

## 4. Core Feature Requirements

### 4.1 Document Management

| ID | Requirement | Priority | Status |
|----|-------------|----------|--------|
| DM-01 | Three-level hierarchy: Space → Collection → Document | Must | Not started |
| DM-02 | Block-based document editor (Tiptap) with rich content types | Must | Not started |
| DM-03 | Full-text search across all documents using PostgreSQL tsvector | Must | Not started |
| DM-04 | Version history with per-version diff (who changed what, when) | Must | Not started |
| DM-05 | Document status: draft, published, archived | Should | Not started |
| DM-06 | Tags for cross-collection organization | Should | Not started |
| DM-07 | Backlinks — show all documents that link to the current one | Should | Not started |
| DM-08 | Sidebar tree navigation with expand/collapse | Must | Not started |
| DM-09 | Breadcrumb navigation (Space → Collection → Document) | Must | Not started |
| DM-10 | Document templates (blank, runbook, post-mortem, SOP, ADR) | Should | Not started |
| DM-11 | Emoji icons on spaces, collections, documents | Could | Not started |
| DM-12 | Nested collections (one level of nesting) | Could | Not started |

### 4.2 Block Types

| ID | Block | Priority | Status |
|----|-------|----------|--------|
| BK-01 | Paragraph (plain text) | Must | Not started |
| BK-02 | Headings H1–H3 | Must | Not started |
| BK-03 | Ordered and unordered lists | Must | Not started |
| BK-04 | Checklist / task list (checkboxes, completable) | Must | Not started |
| BK-05 | Code block with syntax highlighting | Must | Not started |
| BK-06 | Callout / alert box (info, warning, danger) | Should | Not started |
| BK-07 | Image upload and embed | Should | Not started |
| BK-08 | Table (static, resizable columns) | Should | Not started |
| BK-09 | Divider | Should | Not started |
| BK-10 | **Live Context block** — embeds live NightOwl data inline | Must | Not started |
| BK-11 | Quote / blockquote | Should | Not started |
| BK-12 | Embed (URL preview card) | Could | Not started |

### 4.3 Live Context Blocks (NightOwl Integration)

Live Context blocks render real-time NightOwl data inline in a document. They are read-only in the document and refresh on load.

| ID | Block Subtype | Shows | Priority |
|----|---------------|-------|----------|
| LC-01 | On-Call Block | Current primary + secondary on-call for a named roster | Must |
| LC-02 | Service Status Block | Current alert status for a named service | Must |
| LC-03 | Active Alerts Block | List of currently firing alerts, filterable by severity/service | Should |
| LC-04 | Incident Link Block | Link to a NightOwl incident with live status badge | Should |

Example: A runbook for a payment gateway outage contains an On-Call Block showing who's currently on-call, and a Service Status Block for `payment-gateway`. The engineer reads the runbook and immediately sees the live situation without leaving the page.

### 4.4 NightOwl Integration

| ID | Requirement | Priority | Status |
|----|-------------|----------|--------|
| NI-01 | Runbook migration: import all existing NightOwl runbooks into BookOwl at setup | Must | Not started |
| NI-02 | NightOwl can fetch a rendered read-only document by ID or slug | Must | Not started |
| NI-03 | NightOwl can search BookOwl documents (for runbook linking on alert fire) | Must | Not started |
| NI-04 | NightOwl can create a post-mortem document from an incident template on incident close | Must | Not started |
| NI-05 | Documents can be tagged as `type:runbook` or `type:post-mortem` for NightOwl filtering | Must | Not started |
| NI-06 | BookOwl exposes a `/api/v1/live-context` endpoint for NightOwl to query on-call/alert data to power Live Context blocks | Must | Not started |
| NI-07 | Alert fire → NightOwl deep-links to the linked BookOwl runbook | Must | Not started |
| NI-08 | Slack `/nightowl runbook <title>` searches BookOwl and returns the top match | Should | Not started |

### 4.5 Search

| ID | Requirement | Priority | Status |
|----|-------------|----------|--------|
| SE-01 | Full-text search across document titles, content blocks, tags | Must | Not started |
| SE-02 | Search scoped to a space, collection, or global | Must | Not started |
| SE-03 | Highlighted snippet in search results | Must | Not started |
| SE-04 | Filter by document type (runbook, post-mortem, etc.) | Should | Not started |
| SE-05 | Search keyboard shortcut (Cmd/Ctrl+K) command palette | Should | Not started |

### 4.6 Multi-Tenancy & Data Sovereignty

Same requirements as NightOwl:

| ID | Requirement | Priority |
|----|-------------|----------|
| MT-01 | Schema-per-tenant isolation (same pattern as NightOwl) | Must |
| MT-02 | Self-hosted — all data on-premises | Must |
| MT-03 | RBAC: admin, editor, viewer roles per tenant | Must |
| MT-04 | Audit log: document create/edit/delete with user + timestamp | Must |
| MT-05 | API key management for NightOwl service-to-service auth | Must |

### 4.7 Admin

| ID | Requirement | Priority |
|----|-------------|----------|
| AD-01 | Space management (create, rename, archive, set icon) | Must |
| AD-02 | Member management per space (add, remove, set role) | Must |
| AD-03 | API key management (create, list masked, revoke) | Must |
| AD-04 | NightOwl connection settings (API URL, API key, test connection) | Must |
| AD-05 | System status page (DB health, NightOwl connectivity, uptime) | Should |

## 5. Non-Functional Requirements

| Category | Requirement |
|----------|-------------|
| **Deployment** | Kubernetes-native, Helm chart, single namespace alongside NightOwl |
| **Database** | PostgreSQL 16+, schema-per-tenant |
| **Authentication** | Shared OIDC provider with NightOwl, API keys for service-to-service |
| **Search latency** | Document search < 500ms p99 |
| **Editor** | Save on every change (debounced 1s), optimistic UI |
| **Compliance** | GDPR-compatible, audit trail for KRITIS |

## 6. Out of Scope (v1)

- Real-time collaborative editing (single-user editing with last-write-wins is fine for v1)
- Comments and inline annotations
- Public document sharing (all docs are tenant-private)
- PDF export
- Diagrams-as-code (Mermaid, PlantUML) — could add as a block type post-v1
- AI-assisted writing
