# BookOwl

Knowledge management platform for operational teams. Part of the [NightOwl](https://wisbric.com) incident management platform by Wisbric.

BookOwl provides structured documentation — runbooks, post-mortems, SOPs, architecture decision records — with deep integration into NightOwl's alert and incident pipeline.

## Features

- **Spaces & Collections** — Organize documents into spaces with nested collections and drag-and-drop ordering
- **Rich Editor** — Tiptap-based block editor with slash commands, callouts, code blocks, diagrams, and image uploads
- **NightOwl Live Context** — Embed live on-call rosters, service status, and active alerts directly in runbooks
- **Full-Text Search** — PostgreSQL tsvector search with highlighting across all documents
- **Version History** — Every edit creates a version snapshot; restore any previous version
- **Templates** — System and user-created templates for runbooks, post-mortems, and other document types
- **Threaded Comments** — Document-level threaded comments with @mentions, resolve/unresolve, and in-app notifications
- **Image Storage** — Upload images via drag-and-drop or paste; S3 or local filesystem backends
- **Authentication** — Cookie-based sessions (`wisbric_session`), OIDC (Keycloak), local admin break-glass, personal API tokens — auth via shared `core/pkg/auth`
- **Multi-Tenancy** — Schema-per-tenant isolation, shared OIDC provider with NightOwl
- **Admin UI** — NightOwl connection settings, OIDC configuration, user management

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│   Frontend   │────▶│  BookOwl API  │────▶│  PostgreSQL   │
│  React/Vite  │     │   Go / Chi   │     │  (per-tenant) │
└─────────────┘     └──────┬───────┘     └──────────────┘
                           │
                    ┌──────┴───────┐
                    │    Redis     │
                    │  (cache)     │
                    └──────────────┘
                           │
                    ┌──────┴───────┐
                    │   NightOwl   │
                    │  (REST API)  │
                    └──────────────┘
```

BookOwl is a separate microservice from NightOwl. They share the same OIDC provider and multi-tenant architecture, communicate over REST API, and deploy together on Kubernetes via a shared Helm chart.

## Tech Stack

**Backend:** Go 1.25+, Chi router, PostgreSQL 16+ (pgx + sqlc), Redis 7, OIDC (go-oidc), golang-migrate

**Frontend:** React 19, TypeScript 5.9, Vite 7, Tailwind CSS 4, TanStack Query + Router, Tiptap 2, lucide-react

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 20+
- Docker & Docker Compose (for PostgreSQL + Redis)

### Development

```bash
# Start dependencies
docker compose up -d

# Seed the database (creates "acme" tenant + local admin)
make seed

# Run the API server
go run ./cmd/bookowl

# In another terminal, start the frontend
cd web && npm install && npm run dev
```

The API runs on `http://localhost:8081` and the frontend on `http://localhost:3001`.

### Default Credentials

- **Dev API key:** `bw_dev_seed_key_do_not_use_in_production`
- **Local admin:** username `admin`, password `bookowl-admin` (dev mode only; forced password change on first login)
- **Database:** `bookowl:bookowl@localhost:5433/bookowl`

### Login

Navigate to `http://localhost:3001/login` to sign in with the local admin account. In production, users authenticate via OIDC (Keycloak).

### Configuration Notes

- `BOOKOWL_SECRET_KEY` is required for cookie sessions and OIDC secret encryption (must be set in production).
- Dev-only auth shortcuts (like `X-Tenant-Slug`) are only enabled when `DEV_MODE=true`.

## Project Structure

```
cmd/bookowl/          # Binary entrypoint (modes: api, seed, seed-demo)
internal/
  app/                # Application setup and router wiring
  admin/              # Admin API handlers (config, OIDC)
  authadapter/        # Auth storage adapter (implements core/pkg/auth.Storage) + OIDC group mapping
  config/             # Environment-based configuration (extends core BaseConfig)
  db/global/          # sqlc queries for global schema (tenants, API keys)
  db/tenant/          # sqlc queries for tenant schema (documents, users, etc.)
  httpserver/         # HTTP helpers (respond, pagination, decode)
  integration/        # NightOwl integration endpoints
  seed/               # Seed data (dev tenant, demo content, templates)
pkg/
  collection/         # Collection domain (CRUD)
  comment/            # Document comments (threaded, @mentions, resolve)
  document/           # Document domain (CRUD, versions, search)
  image/              # Image upload and storage
  livecontext/        # NightOwl live context proxy + cache
  notification/       # In-app notifications (mentions, replies)
  space/              # Space domain (CRUD, members, tree)
  storage/            # Storage backends (local, S3)
  template/           # Document templates
  tenant/             # Tenant middleware and context
  user/               # User profile and personal tokens
web/                  # React frontend
  src/auth/           # Auth provider (cookie-based sessions via wisbric_session)
  src/components/     # UI components (editor, sidebar, panels)
  src/routes/         # TanStack Router pages
migrations/
  global/             # Global schema migrations
  tenant/             # Per-tenant schema migrations
deploy/helm/          # Helm chart for Kubernetes
docs/                 # Design specifications (01-15)
```

## Available Commands

```bash
make build          # Build the Go binary
make test           # Run all tests
make lint           # Run golangci-lint
make sqlc           # Generate sqlc Go code from SQL queries
make seed           # Seed dev tenant + local admin
make seed-demo      # Seed with full demo data
make docker         # Build backend Docker image
make docker-web     # Build frontend Docker image
```

## API Overview

All endpoints require authentication (OIDC token, session cookie, or API key) and are scoped to a tenant.

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/spaces` | List spaces |
| `GET /api/v1/spaces/:id/tree` | Space tree (collections + documents) |
| `POST /api/v1/documents` | Create document |
| `GET /api/v1/documents/:id` | Get document with content |
| `PUT /api/v1/documents/:id` | Update document (auto-versions) |
| `GET /api/v1/documents/:id/history` | Version history |
| `GET /api/v1/documents/:id/comments` | List comments (threaded) |
| `POST /api/v1/documents/:id/comments` | Add comment |
| `GET /api/v1/search?q=...` | Full-text search |
| `GET /api/v1/templates` | List templates |
| `GET /api/v1/notifications` | List notifications |
| `GET /api/v1/live-context/...` | NightOwl live data proxy |
| `GET /api/v1/profile` | Current user profile |
| `GET /api/v1/integration/...` | NightOwl integration endpoints |
| `GET /api/v1/admin/config` | Admin configuration |

See `docs/02-architecture.md` for the full API reference.

## Documentation

Design specifications are in `docs/`:

| Doc | Topic |
|-----|-------|
| [01-requirements](docs/01-requirements.md) | Product requirements and feature scope |
| [02-architecture](docs/02-architecture.md) | System architecture and API endpoints |
| [03-data-model](docs/03-data-model.md) | PostgreSQL schema and multi-tenancy |
| [04-editor](docs/04-editor.md) | Block-based editor (Tiptap) |
| [05-nightowl-integration](docs/05-nightowl-integration.md) | NightOwl REST API integration |
| [06-branding](docs/06-branding.md) | NightOwl design system |
| [07-deployment](docs/07-deployment.md) | Kubernetes, Helm, CI/CD |
| [08-tasks](docs/08-tasks.md) | Implementation task checklist |
| [09-image-storage](docs/09-image-storage.md) | Image upload and storage backends |
| [10-diagrams](docs/10-diagrams.md) | draw.io diagram blocks |
| [11-oidc-admin](docs/11-oidc-admin.md) | OIDC/Keycloak configuration UI |
| [12-login-and-profile](docs/12-login-and-profile.md) | Login, local admin, profiles, tokens |
| [13-export](docs/13-export.md) | PDF, Markdown, HTML export |
| [14-comments](docs/14-comments.md) | Threaded comments and notifications |
| [15-templates](docs/15-templates.md) | Template library |

## Deployment

BookOwl deploys alongside NightOwl on Kubernetes. See `docs/07-deployment.md` for the full deployment guide.

```bash
helm install bookowl deploy/helm/bookowl/ \
  --set config.dbUrl="postgres://..." \
  --set config.redisUrl="redis://..." \
  --set config.oidcIssuer="https://keycloak.example.com/realms/wisbric"
```

## License

Proprietary — Wisbric. All rights reserved.
