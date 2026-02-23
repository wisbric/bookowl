# BookOwl — NightOwl Integration Specification

## 1. Overview

BookOwl and NightOwl are two separate microservices that integrate over REST API. This document defines the full integration contract: what each service calls on the other, how auth works, and how the runbook migration happens.

```
NightOwl calls BookOwl for:
  - Fetching the runbook linked to an alert (inline display)
  - Searching for a runbook to link when creating an alert rule
  - Creating a post-mortem document when an incident closes
  - Slack /nightowl runbook <title> command

BookOwl calls NightOwl for:
  - Live Context block data (on-call, active alerts, service status)
  - User identity validation (shared OIDC, but BookOwl may check NightOwl
    for display names / timezone info if needed)
```

Both directions use API key authentication with a dedicated service account key.

## 2. Configuration

### BookOwl config (per tenant)

Stored in `public.tenants.config` JSONB:

```json
{
  "nightowl_api_url": "http://nightowl-api.nightowl.svc:8080",
  "nightowl_api_key": "ow_service_account_key_for_bookowl"
}
```

Configurable via `PUT /api/v1/admin/config` and the admin UI.

### NightOwl config (per tenant)

NightOwl needs to know BookOwl's location. Add to NightOwl's tenant config:

```json
{
  "bookowl_api_url": "http://bookowl-api.bookowl.svc:8081",
  "bookowl_api_key": "bw_service_account_key_for_nightowl"
}
```

This is a **NightOwl-side change** — see the NightOwl integration tasks section below.

## 3. BookOwl Endpoints for NightOwl

These endpoints are available at `/api/v1/integration/` and require an API key with `role=admin`.

### 3.1 List Runbooks

```
GET /api/v1/integration/runbooks
Header: X-API-Key: <nightowl-service-key>

Query params:
  - q: optional text search
  - limit: default 20
  - offset: default 0

Response:
{
  "items": [
    {
      "id": "uuid",
      "title": "Pod CrashLoopBackOff",
      "slug": "pod-crashloopbackoff",
      "tags": ["kubernetes", "pod", "crashloop"],
      "url": "https://bookowl.example.com/spaces/platform/docs/pod-crashloopbackoff",
      "updated_at": "2026-02-20T10:00:00Z"
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

### 3.2 Fetch Runbook for Inline Display

```
GET /api/v1/integration/runbooks/:id
Header: X-API-Key: <nightowl-service-key>

Response:
{
  "id": "uuid",
  "title": "Pod CrashLoopBackOff",
  "slug": "pod-crashloopbackoff",
  "content_text": "Plain text version of the runbook for inline display",
  "content_html": "<h1>Pod CrashLoopBackOff</h1><p>...</p>",  -- rendered HTML
  "url": "https://bookowl.example.com/spaces/...",
  "tags": ["kubernetes", "pod"],
  "updated_at": "2026-02-20T10:00:00Z"
}
```

NightOwl renders `content_html` in the alert detail inline panel. It also provides the `url` as a "View in BookOwl" deep link.

### 3.3 Create Post-Mortem

Called by NightOwl when an incident is resolved and the engineer clicks "Create Post-Mortem" (or automatically if the tenant config enables auto-creation).

```
POST /api/v1/integration/post-mortems
Header: X-API-Key: <nightowl-service-key>
Content-Type: application/json

Body:
{
  "title": "Post-Mortem: Pod CrashLoopBackOff — payment-gateway",
  "space_slug": "post-mortems",     -- target space (created if not exists)
  "incident": {
    "id": "nightowl-incident-uuid",
    "title": "Pod CrashLoopBackOff",
    "severity": "critical",
    "root_cause": "OOM kill due to memory leak in payment service v2.3.1",
    "solution": "Rolled back to v2.3.0, increased memory limit to 512Mi",
    "created_at": "2026-02-20T08:00:00Z",
    "resolved_at": "2026-02-20T09:42:00Z",
    "resolved_by": "Stefan K."
  }
}

Response 201:
{
  "id": "bookowl-document-uuid",
  "url": "https://bookowl.example.com/spaces/post-mortems/docs/post-mortem-...",
  "title": "Post-Mortem: Pod CrashLoopBackOff — payment-gateway"
}
```

BookOwl creates the document using the post-mortem template from `docs/04-editor.md`, substitutes the incident data, and returns the URL. NightOwl stores this URL on the incident record as `post_mortem_url`.

### 3.4 Search (for Runbook Linking)

Used when a NightOwl admin is configuring an alert rule and wants to link a BookOwl runbook.

```
GET /api/v1/integration/search?q=crashloop&type=runbook&limit=5
Header: X-API-Key: <nightowl-service-key>

Response:
{
  "items": [
    {
      "id": "uuid",
      "title": "Pod CrashLoopBackOff",
      "excerpt": "Check pod events and logs. If OOM kill, increase memory limit...",
      "url": "https://bookowl.example.com/...",
      "score": 0.95
    }
  ]
}
```

## 4. NightOwl Endpoints for BookOwl (Live Context)

BookOwl calls NightOwl for Live Context block data. These are existing NightOwl endpoints — no new endpoints needed in NightOwl.

### 4.1 On-Call Block

```
GET /api/v1/rosters/:id/oncall
Header: X-API-Key: <bookowl-service-key>

NightOwl returns the existing on-call response (primary + secondary + shift info)
BookOwl caches this in Redis with key: live-context:{tenant_slug}:oncall:{roster_id}
TTL: 30 seconds
```

### 4.2 Service Status Block

```
GET /api/v1/alerts?status=firing&service_name=<name>&limit=10
Header: X-API-Key: <bookowl-service-key>

BookOwl uses the existing NightOwl alert list endpoint.
Caches: live-context:{tenant_slug}:alerts:{service_name}
TTL: 30 seconds
```

### 4.3 Active Alerts Block

```
GET /api/v1/alerts?status=firing&severity=critical,major&limit=5
Header: X-API-Key: <bookowl-service-key>

Caches: live-context:{tenant_slug}:alerts:active:{severity}
TTL: 30 seconds
```

## 5. Live Context API (BookOwl proxy, called by the frontend)

The browser calls BookOwl (not NightOwl directly) for live data. BookOwl handles caching and auth.

```
GET /api/v1/live-context/oncall/:rosterId
→ Returns on-call data from cache or NightOwl

GET /api/v1/live-context/service/:serviceName
→ Returns active alerts for service

GET /api/v1/live-context/alerts?severity=critical&limit=5
→ Returns active alerts

GET /api/v1/live-context/incident/:nightowlIncidentId
→ Returns incident status/title from NightOwl
```

All these endpoints:
- Require normal OIDC/API key auth (same as any BookOwl endpoint)
- Check Redis cache first (30s TTL)
- On cache miss: call NightOwl API with the service account key
- On NightOwl unreachable: return last cached value if available, else `{ "status": "unavailable" }`
- Return a consistent envelope: `{ "data": {...}, "cached_at": "...", "source": "cache|live" }`

## 6. Runbook Migration

When BookOwl is first deployed alongside an existing NightOwl instance, existing NightOwl runbooks need to be imported.

### 6.1 Migration Command

A one-time migration command in the BookOwl binary:

```bash
go run ./cmd/bookowl --mode=migrate-nightowl-runbooks \
  --nightowl-url=http://nightowl-api:8080 \
  --nightowl-api-key=<key> \
  --target-space=runbooks \
  --target-collection=imported
```

### 6.2 Migration Logic

```
1. GET /api/v1/runbooks from NightOwl (all runbooks, paginated)
2. For each runbook:
   a. Create a BookOwl document with doc_type=runbook
   b. Convert markdown content to Tiptap JSON
      (use goldmark or similar to parse MD → HTML → Tiptap JSON)
   c. Copy: title, tags, category→collection, is_template
   d. Set status=published
   e. Store original NightOwl runbook ID in document metadata
3. After migration: NightOwl runbooks remain in place (read-only mode)
   NightOwl admin must manually switch to BookOwl links after verifying
4. Log migration summary: N runbooks imported, M failed with reasons
```

### 6.3 Post-Migration NightOwl Changes

After verifying the migration, NightOwl's `pkg/runbook` domain is deprecated:

**In NightOwl:**
- `incidents.runbook_id` (FK to NightOwl runbooks) is replaced with `incidents.runbook_url` (TEXT, pointing to BookOwl)
- The `runbook-*` API endpoints return 410 Gone with a message pointing to BookOwl
- NightOwl alert detail page: "View Runbook" button links to the BookOwl URL instead of rendering inline markdown
- NightOwl alert fires: enrichment still attaches the solution text (from the incident record) inline, but the full runbook link goes to BookOwl

**This is a NightOwl task, not a BookOwl task.** BookOwl owns the migration command; NightOwl owns the deprecation of its own runbook system.

## 7. NightOwl Changes Required

The following changes need to be made in the NightOwl codebase to complete the integration. These are tasks for the NightOwl project, tracked in NightOwl's `docs/05-tasks.md`:

### 7.1 Tenant Config

Add BookOwl connection settings to tenant config:
```go
type TenantConfig struct {
    // existing fields...
    BookOwlAPIURL  string `json:"bookowl_api_url"`
    BookOwlAPIKey  string `json:"bookowl_api_key"`
    BookOwlEnabled bool   `json:"bookowl_enabled"`
}
```

### 7.2 Alert Enrichment

In `pkg/alert/enrich.go`, after KB enrichment, if BookOwl is enabled:
- Attach `runbook_url` (fetched from BookOwl search for matching runbook)
- Or if alert rule has a manually linked `runbook_url`, use that directly

### 7.3 Incident Resolution

In the incident resolution flow, if BookOwl is enabled:
- Add "Create Post-Mortem in BookOwl" option alongside the existing Slack prompt
- On confirmation: `POST /api/v1/integration/post-mortems` on BookOwl
- Store returned URL as `incidents.post_mortem_url`

### 7.4 Slack Commands

In `pkg/slack/commands.go`, handle `/nightowl runbook <title>`:
```
/nightowl runbook crashloop
→ GET /api/v1/integration/search?q=crashloop&type=runbook&limit=3 on BookOwl
→ Return top 3 results as Slack blocks with direct links
```

### 7.5 Alert Detail Page

In the NightOwl frontend alert detail view, if the alert has a `runbook_url`:
- Show a "Runbook" panel with the BookOwl rendered HTML (fetched via `GET /api/v1/integration/runbooks/:id`)
- "Open in BookOwl" button that deep-links to the full editor

## 8. Slack Integration (BookOwl side)

BookOwl does not have its own Slack app. Slack integration goes through NightOwl. NightOwl proxies runbook searches and post-mortem links to BookOwl.

Future: BookOwl could add its own slash command `/bookowl search <query>` for direct document search without going through NightOwl. Out of scope for v1.

## 9. Acceptance Criteria

- [ ] NightOwl can fetch a BookOwl runbook and display it inline in alert detail
- [ ] NightOwl can search BookOwl runbooks when linking to an alert rule
- [ ] NightOwl creates a post-mortem in BookOwl when an incident is resolved
- [ ] BookOwl Live Context blocks show real NightOwl on-call and alert data
- [ ] Live Context blocks degrade gracefully when NightOwl is unreachable
- [ ] Live Context blocks refresh every 30s without page reload
- [ ] Runbook migration command imports all NightOwl runbooks as BookOwl documents
- [ ] Migrated runbooks are tagged `doc_type=runbook` and appear in integration search
- [ ] BookOwl admin UI shows NightOwl connection status and allows configuring the API key
