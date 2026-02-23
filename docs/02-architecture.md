# BookOwl — Architecture Specification

## 1. System Overview

BookOwl is a standalone Go API + React frontend deployed as a separate Kubernetes service alongside NightOwl. They share the same OIDC provider, the same tenant slugs, and communicate via REST API.

```
┌──────────────────────────────────────────────────────────────────────┐
│                          Ingress (TLS)                               │
│   nightowl.example.com          bookowl.example.com                  │
├───────────────────────────────────┬──────────────────────────────────┤
│         NightOwl                  │            BookOwl               │
│  ┌─────────────────────────┐      │   ┌────────────────────────────┐ │
│  │   Go API (chi)          │      │   │   Go API (chi)             │ │
│  │  alert / roster /       │      │   │  document / space /        │ │
│  │  incident / runbook*    │◄─────┤───│  collection / search /     │ │
│  └────────────┬────────────┘      │   │  livecontext               │ │
│               │                   │   └──────────────┬─────────────┘ │
│  ┌────────────┴────────────┐      │   ┌──────────────┴─────────────┐ │
│  │   PostgreSQL (NightOwl) │      │   │   PostgreSQL (BookOwl)     │ │
│  │   schema-per-tenant     │      │   │   schema-per-tenant        │ │
│  └─────────────────────────┘      │   └────────────────────────────┘ │
│               │                   │               │                  │
│  ┌────────────┴────────────┐      │   ┌──────────────────────────┐  │
│  │         Redis           │◄─────┤───│       Redis              │  │
│  └─────────────────────────┘      │   │  (can share instance)    │  │
│                                   │   └──────────────────────────┘  │
│  ┌─────────────────────────┐      │   ┌────────────────────────────┐ │
│  │   React SPA (nginx)     │      │   │   React SPA (nginx)        │ │
│  └─────────────────────────┘      │   └────────────────────────────┘ │
└───────────────────────────────────┴──────────────────────────────────┘
                    Both services share same OIDC provider
```

*NightOwl's `pkg/runbook` domain is deprecated once BookOwl is deployed. Runbook records migrate to BookOwl. NightOwl keeps only a reference field `runbook_url` on incidents and alerts.

## 2. Domain Architecture

```
pkg/
├── space/           # Space CRUD + membership
│   ├── handler.go
│   ├── service.go
│   ├── store.go
│   └── space.go
├── collection/      # Collection CRUD within a space
│   ├── handler.go
│   ├── service.go
│   ├── store.go
│   └── collection.go
├── document/        # Document CRUD, versions, search
│   ├── handler.go
│   ├── service.go
│   ├── store.go
│   ├── document.go
│   └── search.go    # Full-text search queries
├── block/           # Block content serialization/validation
│   └── block.go     # Block type definitions, ProseMirror JSON schema
├── livecontext/     # Proxy to NightOwl for live data blocks
│   ├── handler.go   # GET /api/v1/live-context/oncall, /alerts, /services
│   ├── client.go    # NightOwl REST client
│   └── cache.go     # Redis cache for live context (30s TTL)
├── tenant/          # Multi-tenancy (same pattern as NightOwl)
│   ├── tenant.go
│   ├── middleware.go
│   └── provisioner.go
├── user/            # User CRUD (synced from OIDC, not manually managed)
│   ├── handler.go
│   ├── service.go
│   ├── store.go
│   └── user.go
└── apikey/          # API key management (for NightOwl service account)
    ├── handler.go
    ├── service.go
    ├── store.go
    └── apikey.go

internal/
├── app/             # Application orchestrator (api, seed, seed-demo)
├── auth/            # OIDC + API key middleware (same as NightOwl)
├── audit/           # Async audit log writer
├── config/          # Env-based config
├── db/              # sqlc-generated models and queries
├── docs/            # OpenAPI/Swagger UI
├── httpserver/      # Chi server, middleware, response helpers, pagination
├── platform/        # PostgreSQL pool, Redis client, migration runner
├── seed/            # Dev seed + demo seed
├── telemetry/       # Logger, metrics, tracing
└── version/         # Version + commit (ldflags)
```

## 3. API Design

All endpoints under `/api/v1/` require authentication except health checks.

```
# Health
GET  /healthz
GET  /readyz
GET  /metrics
GET  /api/docs

# System
GET  /api/v1/status
GET  /api/v1/ping

# Spaces
POST   /api/v1/spaces
GET    /api/v1/spaces
GET    /api/v1/spaces/:id
PUT    /api/v1/spaces/:id
DELETE /api/v1/spaces/:id
GET    /api/v1/spaces/:id/members
POST   /api/v1/spaces/:id/members
DELETE /api/v1/spaces/:id/members/:userId
GET    /api/v1/spaces/:id/tree          # Full space tree (collections + doc titles)

# Collections
POST   /api/v1/spaces/:spaceId/collections
GET    /api/v1/spaces/:spaceId/collections
GET    /api/v1/spaces/:spaceId/collections/:id
PUT    /api/v1/spaces/:spaceId/collections/:id
DELETE /api/v1/spaces/:spaceId/collections/:id

# Documents
POST   /api/v1/documents
GET    /api/v1/documents                         # List (filterable by space, collection, type, tags)
GET    /api/v1/documents/:id
PUT    /api/v1/documents/:id
DELETE /api/v1/documents/:id
GET    /api/v1/documents/:id/history             # Version history
GET    /api/v1/documents/:id/history/:versionId  # Specific version content
POST   /api/v1/documents/:id/restore/:versionId  # Restore to version

# Search
GET    /api/v1/search?q=<text>&space=<id>&type=<runbook|post-mortem>

# Live Context (proxy to NightOwl, cached 30s)
GET    /api/v1/live-context/oncall/:rosterId
GET    /api/v1/live-context/alerts?service=<name>&severity=<sev>
GET    /api/v1/live-context/service/:serviceName

# NightOwl integration endpoints (API key auth, service-to-service)
GET    /api/v1/integration/runbooks              # List runbook documents
GET    /api/v1/integration/runbooks/:id          # Fetch runbook for inline display
POST   /api/v1/integration/post-mortems          # Create post-mortem from template
GET    /api/v1/integration/search?q=<text>&type=runbook  # Search for runbook linking

# Users
GET    /api/v1/users
GET    /api/v1/users/:id

# API Keys
POST   /api/v1/api-keys
GET    /api/v1/api-keys
DELETE /api/v1/api-keys/:id

# Admin
GET    /api/v1/admin/config
PUT    /api/v1/admin/config              # NightOwl API URL + key settings

# Audit Log
GET    /api/v1/audit-log
```

## 4. Authentication

Identical pattern to NightOwl:

- **OIDC/JWT** — web users, same OIDC provider as NightOwl
- **API Key** (`X-API-Key` header) — NightOwl service account, other integrations
- **Dev header** (`X-Tenant-Slug`) — development fallback

NightOwl authenticates to BookOwl using a dedicated service account API key configured per tenant via `BOOKOWL_NIGHTOWL_API_KEY`.

## 5. Live Context Architecture

Live Context blocks in documents need real NightOwl data. The flow:

```
Browser renders document
  → Document JSON contains LiveContext block { type: "live-context", subtype: "oncall", rosterId: "uuid" }
  → Frontend calls GET /api/v1/live-context/oncall/:rosterId
  → BookOwl checks Redis cache (key: live-context:{tenant}:oncall:{rosterId}, TTL: 30s)
  → If miss: BookOwl calls NightOwl GET /api/v1/rosters/:id/oncall
  → BookOwl caches + returns the result
  → Frontend renders On-Call block with live data
```

This means:
- BookOwl never stores NightOwl data — it only caches transiently
- Cache TTL is 30s — live enough for incident response, not hammering NightOwl
- If NightOwl is unreachable: return cached data if available, else show "unavailable" state in the block
- NightOwl API key is tenant-scoped: BookOwl uses the key stored in the tenant config

## 6. Deployment Architecture

```yaml
# Helm chart produces (in addition to NightOwl resources):
Deployment:  bookowl-api       (2+ replicas, stateless)
Deployment:  bookowl-web       (nginx serving React SPA)
ConfigMap:   bookowl-config
Secret:      bookowl-secrets   (DB creds, NightOwl API key)
Ingress:     bookowl           (TLS via cert-manager)
Service:     bookowl-api
Service:     bookowl-web
ServiceMonitor: bookowl        (Prometheus scraping)
```

BookOwl can share the same PostgreSQL instance as NightOwl (in a separate database) or use a separate instance. Redis can be shared.

## 7. Observability

Prometheus namespace: `bookowl`. Metrics mirroring NightOwl patterns:

```
bookowl_api_request_duration_seconds{method, path, status}
bookowl_documents_created_total{type}
bookowl_search_queries_total
bookowl_livecontext_cache_hits_total{subtype}
bookowl_livecontext_nightowl_errors_total
```

Structured JSON logging via `slog`, OpenTelemetry traces via OTLP gRPC.
