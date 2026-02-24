# CLAUDE.md — BookOwl

## Project Overview

BookOwl is a knowledge management platform built by Wisbric. It serves two roles:

1. **General wiki** — XWiki-level breadth for teams that need structured documentation (spaces, collections, pages, rich content, search, version history)
2. **NightOwl integration layer** — runbooks, post-mortems, and service docs are deeply integrated with NightOwl's alert and incident pipeline

BookOwl is a **separate microservice** from NightOwl. They share the same OIDC provider and multi-tenant architecture, communicate over REST API, and are deployed together on Kubernetes via a shared Helm chart.

BookOwl is a Wisbric product (wisbric.com). The parent brand is NightOwl — BookOwl is the documentation pillar of the NightOwl platform.

## Specifications

All design decisions are captured in `docs/`. Always read the relevant spec before implementing:

- `docs/01-requirements.md` — Product requirements and feature scope
- `docs/02-architecture.md` — System architecture, domain structure, API endpoints
- `docs/03-data-model.md` — PostgreSQL schema, multi-tenancy, full-text search
- `docs/04-editor.md` — Block-based editor spec (Tiptap), block types, live context blocks
- `docs/05-nightowl-integration.md` — Integration contract with NightOwl (REST API, runbook migration, post-mortem creation, live blocks)
- `docs/06-branding.md` — BookOwl branding (follows NightOwl design system)
- `docs/07-deployment.md` — Kubernetes deployment, Helm chart, CI/CD
- `docs/08-tasks.md` — Implementation task checklist and progress
- `docs/09-image-storage.md` — Image upload and storage backends (local, S3)
- `docs/10-diagrams.md` — draw.io diagram blocks in the editor
- `docs/11-oidc-admin.md` — OIDC/Keycloak configuration UI
- `docs/12-login-and-profile.md` — Login, local admin, user profiles, personal API tokens
- `docs/13-export.md` — PDF, Markdown, HTML export
- `docs/14-comments.md` — Threaded comments, @mentions, and notifications
- `docs/15-templates.md` — Template library (system + user templates)

## Branding

The product is called **BookOwl**. It uses the NightOwl design system from the parent product.

- Dark mode is the default (theme toggle available)
- Same color palette as NightOwl: `#0A2E24` primary, `#00e5a0` accent, severity colors
- Sidebar navigation layout with owl logo
- Footer: "A Wisbric product"
- Owl-themed naming: Spaces are "Nests", the search is "Scan", version history is "Feathers"
  (these are flavor names only — use plain terms in API and code)

## Tech Stack

### Backend

- **Language:** Go 1.25+ (module: `github.com/wisbric/bookowl`)
- **Binary:** `cmd/bookowl` with modes: `api`, `seed`, `seed-demo`
- **Router:** go-chi/chi/v5
- **Database:** PostgreSQL 16+ via jackc/pgx/v5 + sqlc
- **Migrations:** golang-migrate (SQL files in `migrations/`)
- **Cache:** Redis 7 via redis/go-redis/v9
- **Auth:** OIDC (coreos/go-oidc/v3) + API keys (SHA-256) — same provider as NightOwl
- **Full-text search:** PostgreSQL tsvector (same pattern as NightOwl incidents)
- **Metrics:** prometheus/client_golang (namespace: `bookowl`)
- **Tracing:** OpenTelemetry (OTLP gRPC)
- **Logging:** slog (structured JSON)
- **Config:** caarlos0/env/v11
- **UUIDs:** google/uuid

### Frontend

- **Framework:** React 19 + TypeScript 5.9
- **Build:** Vite 7
- **UI Kit:** shadcn/ui + Tailwind CSS 4 (same theme tokens as NightOwl)
- **State:** TanStack Query 5 + TanStack Router 1
- **Editor:** Tiptap 2 (block-based, extensible)
- **Forms:** React Hook Form + Zod
- **Icons:** lucide-react
- **Dates:** date-fns

## Code Conventions

These mirror NightOwl exactly. If in doubt, look at how NightOwl does it.

### Go

- Standard `gofmt` + `golangci-lint`
- Package names are single lowercase words matching directory name
- Table-driven tests
- Errors: `fmt.Errorf("doing X: %w", err)` — never discard
- Context: always `context.Context` as first parameter
- SQL: prefer sqlc-generated code; raw SQL for JOINs not in schema
- HTTP handlers return JSON; always use `httpserver.Respond()` and `httpserver.RespondError()`
- Domain packages follow `handler.go` / `service.go` / `store.go` / `{domain}.go` pattern
- Per-request store creation from `tenant.ConnFromContext(r.Context())`

### Frontend

- Functional components only
- TanStack Query for all API calls — never raw fetch in components
- TanStack Router for all navigation — no `useNavigate` with raw strings
- Zod schemas co-located with forms
- Dark mode first — always test in dark mode before light

## Multi-Tenancy

Schema-per-tenant isolation, identical to NightOwl. Every request resolves a tenant from JWT or API key. The middleware acquires a pooled connection and sets `search_path` before any query.

BookOwl tenants correspond 1:1 with NightOwl tenants — they share the same tenant slug. When NightOwl calls BookOwl it passes the tenant context via API key scoped to that tenant.

Never reference tenant data without going through the tenant middleware.

## Development

```bash
docker compose up -d          # PostgreSQL + Redis
make seed                     # Create "acme" dev tenant (idempotent)
go run ./cmd/bookowl          # API on :8081
cd web && npm run dev         # Frontend on :3001 (proxies /api to :8081)
```

- Dev API key: `bw_dev_seed_key_do_not_use_in_production`
- Local admin: username `admin`, password `bookowl-admin` (dev mode only; forced password change on first login)
- Login URL: `http://localhost:3001/login`
- Env vars prefix: `BOOKOWL_` (e.g., `BOOKOWL_MODE`, `BOOKOWL_PORT`)
- DB credentials (dev): `bookowl:bookowl@localhost:5433/bookowl`
- NightOwl API (dev): `BOOKOWL_NIGHTOWL_API_URL=http://localhost:8080`
- NightOwl API key (dev): `BOOKOWL_NIGHTOWL_API_KEY=ow_dev_seed_key_do_not_use_in_production`

## Testing

- Unit tests: mock dependencies via interfaces, table-driven
- Integration tests: testcontainers-go for real PostgreSQL and Redis
- Run `make test` before committing
- Run `make lint` before committing

## Commit Style

Same as NightOwl:
- Conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `chore:`
- One logical change per commit
- Reference task IDs from `docs/01-requirements.md` where applicable
