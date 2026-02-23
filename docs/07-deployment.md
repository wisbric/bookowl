# BookOwl — Deployment & CI/CD

## 1. Container Images

BookOwl produces two images from multi-stage builds:

| Image | Contents | Purpose |
|-------|----------|---------|
| `ghcr.io/wisbric/bookowl` | Go binary | API server (all modes) |
| `ghcr.io/wisbric/bookowl-web` | Nginx + React SPA | Frontend |

Tags: `latest` (main branch), `v0.1.0` (release), `sha-abc1234` (commit SHA).
Platforms: `linux/amd64` and `linux/arm64`.

---

## 2. Backend Dockerfile

`Dockerfile` (repo root):

```dockerfile
# Build stage
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/wisbric/bookowl/internal/version.Version=${VERSION} -X github.com/wisbric/bookowl/internal/version.Commit=${COMMIT}" \
    -o /bookowl ./cmd/bookowl

# Production stage
FROM gcr.io/distroless/static-debian12
COPY --from=build /bookowl /bookowl
COPY --from=build /app/migrations /migrations
ENTRYPOINT ["/bookowl"]
```

---

## 3. Frontend Dockerfile

`web/Dockerfile`:

```dockerfile
# Build stage
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Production stage
FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

`web/nginx.conf`:

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    # SPA routing
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API proxy to BookOwl API service
    location /api/ {
        proxy_pass http://bookowl-api:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        client_max_body_size 20m;
    }

    location /healthz {
        return 200 'ok';
        add_header Content-Type text/plain;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff2?)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

---

## 4. Development: docker-compose.yml

`docker-compose.yml` (repo root) — for local development:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: bookowl
      POSTGRES_PASSWORD: bookowl
      POSTGRES_DB: bookowl
    ports:
      - "5433:5432"   # 5433 to avoid colliding with NightOwl on 5432
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bookowl"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6380:6379"   # 6380 to avoid colliding with NightOwl on 6379
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

`docker-compose.demo.yml` — full-stack one-command demo:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: bookowl
      POSTGRES_PASSWORD: bookowl
      POSTGRES_DB: bookowl
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bookowl"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  api:
    image: bookowl:dev
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      BOOKOWL_MODE: api
      BOOKOWL_PORT: "8081"
      BOOKOWL_DB_URL: postgres://bookowl:bookowl@postgres:5432/bookowl?sslmode=disable
      BOOKOWL_REDIS_URL: redis://redis:6379/1
      BOOKOWL_DEV_MODE: "true"
      BOOKOWL_NIGHTOWL_API_URL: ""   # leave blank to disable live context in demo
      BOOKOWL_NIGHTOWL_API_KEY: ""
    ports:
      - "8081:8081"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  seed:
    image: bookowl:dev
    environment:
      BOOKOWL_MODE: seed-demo
      BOOKOWL_DB_URL: postgres://bookowl:bookowl@postgres:5432/bookowl?sslmode=disable
    depends_on:
      - api
    restart: "no"

  web:
    image: bookowl-web:dev
    build:
      context: ./web
      dockerfile: Dockerfile
    ports:
      - "3001:80"
    depends_on:
      - api

volumes:
  postgres_data:
```

Run the demo:

```bash
make docker
docker compose -f docker-compose.demo.yml up
# BookOwl at http://localhost:3001
# API at http://localhost:8081
```

---

## 5. Makefile

```makefile
.PHONY: build run test lint sqlc seed seed-demo docker docker-web migrate migrate-runbooks

build:
	go build -o bin/bookowl ./cmd/bookowl

run:
	BOOKOWL_MODE=api \
	BOOKOWL_DB_URL=postgres://bookowl:bookowl@localhost:5433/bookowl?sslmode=disable \
	BOOKOWL_REDIS_URL=redis://localhost:6380/1 \
	BOOKOWL_DEV_MODE=true \
	go run ./cmd/bookowl

seed:
	BOOKOWL_MODE=seed \
	BOOKOWL_DB_URL=postgres://bookowl:bookowl@localhost:5433/bookowl?sslmode=disable \
	go run ./cmd/bookowl

seed-demo:
	BOOKOWL_MODE=seed-demo \
	BOOKOWL_DB_URL=postgres://bookowl:bookowl@localhost:5433/bookowl?sslmode=disable \
	go run ./cmd/bookowl

test:
	go test ./... -count=1

lint:
	golangci-lint run ./...

sqlc:
	sqlc generate

docker:
	docker build -t bookowl:dev .

docker-web:
	docker build -t bookowl-web:dev ./web

migrate-runbooks:
	BOOKOWL_MODE=migrate-nightowl-runbooks \
	BOOKOWL_DB_URL=postgres://bookowl:bookowl@localhost:5433/bookowl?sslmode=disable \
	BOOKOWL_NIGHTOWL_API_URL=http://localhost:8080 \
	BOOKOWL_NIGHTOWL_API_KEY=ow_dev_seed_key_do_not_use_in_production \
	go run ./cmd/bookowl
```

---

## 6. GitHub Actions

### CI — `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  backend:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: bookowl
          POSTGRES_PASSWORD: bookowl
          POSTGRES_DB: bookowl
        ports: ["5432:5432"]
        options: --health-cmd pg_isready --health-interval 10s --health-timeout 5s --health-retries 5
      redis:
        image: redis:7-alpine
        ports: ["6379:6379"]
        options: --health-cmd "redis-cli ping" --health-interval 10s --health-timeout 5s --health-retries 5

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
          cache: true
      - name: Install sqlc
        run: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
      - name: Verify sqlc diff
        run: sqlc diff
      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
      - name: Test
        run: make test
        env:
          BOOKOWL_DB_URL: postgres://bookowl:bookowl@localhost:5432/bookowl?sslmode=disable
          BOOKOWL_REDIS_URL: redis://localhost:6379/1
      - name: Build
        run: make build

  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "22"
          cache: npm
          cache-dependency-path: web/package-lock.json
      - run: cd web && npm ci
      - run: cd web && npm run lint
      - run: cd web && npx tsc --noEmit
      - run: cd web && npm run build
```

### Release — `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    branches: [main]
    tags: ["v*"]

env:
  REGISTRY: ghcr.io
  IMAGE_API: ghcr.io/wisbric/bookowl
  IMAGE_WEB: ghcr.io/wisbric/bookowl-web

jobs:
  build-api:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4
      - uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ${{ env.IMAGE_API }}
          tags: |
            type=ref,event=branch
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=sha,prefix=sha-
      - uses: docker/setup-buildx-action@v3
      - uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          platforms: linux/amd64,linux/arm64
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ github.ref_name }}
            COMMIT=${{ github.sha }}

  build-web:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4
      - uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ${{ env.IMAGE_WEB }}
          tags: |
            type=ref,event=branch
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=sha,prefix=sha-
      - uses: docker/setup-buildx-action@v3
      - uses: docker/build-push-action@v6
        with:
          context: ./web
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          platforms: linux/amd64,linux/arm64
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

---

## 7. Helm Chart

### Chart Structure

```
deploy/helm/bookowl/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── deployment-api.yaml
│   ├── deployment-web.yaml
│   ├── service-api.yaml
│   ├── service-web.yaml
│   ├── ingress.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── serviceaccount.yaml
│   ├── servicemonitor.yaml
│   └── pdb.yaml
```

### `Chart.yaml`

```yaml
apiVersion: v2
name: bookowl
description: BookOwl — Knowledge management platform by Wisbric
type: application
version: 0.1.0
appVersion: "0.1.0"
keywords: [knowledge-base, documentation, runbooks, wisbric, nightowl]
maintainers:
  - name: Wisbric
    url: https://wisbric.com
```

### `values.yaml`

```yaml
image:
  repository: ghcr.io/wisbric/bookowl
  tag: ""   # defaults to Chart.appVersion
  pullPolicy: IfNotPresent

web:
  image:
    repository: ghcr.io/wisbric/bookowl-web
    tag: ""
    pullPolicy: IfNotPresent
  replicas: 2
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 200m
      memory: 128Mi

api:
  replicas: 2
  port: 8081
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi

database:
  url: ""   # set via external secret

redis:
  url: ""   # set via external secret

oidc:
  issuerUrl: ""
  clientId: bookowl

nightowl:
  apiUrl: ""
  apiKey: ""   # set via external secret

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-body-size: "20m"
  hosts:
    - host: bookowl.example.com
      paths:
        - path: /
          pathType: Prefix
          service: web
        - path: /api
          pathType: Prefix
          service: api
  tls:
    - secretName: bookowl-tls
      hosts:
        - bookowl.example.com

metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s

pdb:
  enabled: true
  minAvailable: 1

serviceAccount:
  create: true
  name: ""

imagePullSecrets: []
extraEnv: []
```

### External Secrets (preferred for production)

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: bookowl-secrets
  namespace: bookowl
spec:
  secretStoreRef:
    name: vault
    kind: ClusterSecretStore
  target:
    name: bookowl-secrets
  data:
    - secretKey: database-url
      remoteRef:
        key: bookowl/database-url
    - secretKey: redis-url
      remoteRef:
        key: bookowl/redis-url
    - secretKey: nightowl-api-key
      remoteRef:
        key: bookowl/nightowl-api-key
```

---

## 8. Namespace Layout (running alongside NightOwl)

```
namespace: nightowl    → NightOwl API + worker + web
namespace: bookowl     → BookOwl API + web
namespace: data        → PostgreSQL (CNPG), Redis (shared)
namespace: monitoring  → Prometheus, Grafana, Alertmanager
```

Both services share the same CNPG PostgreSQL cluster and Redis instance — separate databases and key prefixes. Same OIDC realm.

Install:

```bash
helm upgrade --install bookowl ./deploy/helm/bookowl \
  --namespace bookowl --create-namespace \
  -f deploy/helm/bookowl/values-production.yaml
```

---

## 9. Deployment Checklist

**Pre-deploy**
- [ ] PostgreSQL database `bookowl` created in CNPG cluster
- [ ] Redis DB index 1 available (separate from NightOwl's index 0)
- [ ] DNS record for `bookowl.example.com` → ingress
- [ ] OIDC client `bookowl` registered in Keycloak/Dex
- [ ] BookOwl service account API key created in NightOwl (role: engineer)
- [ ] NightOwl service account API key created in BookOwl (role: admin)
- [ ] External secrets configured

**Deploy**
- [ ] `helm install` or `helm upgrade` applied
- [ ] All pods Running: `kubectl get pods -n bookowl`
- [ ] `curl https://bookowl.example.com/healthz` → 200
- [ ] `curl https://bookowl.example.com/readyz` → 200
- [ ] Frontend loads, OIDC login works

**Post-deploy**
- [ ] Run runbook migration:
  ```bash
  kubectl exec -n bookowl deploy/bookowl-api -- \
    /bookowl --mode=migrate-nightowl-runbooks
  ```
- [ ] Verify migrated runbooks appear in BookOwl UI
- [ ] Configure NightOwl tenant with BookOwl URL + API key
- [ ] Test NightOwl alert detail shows inline runbook from BookOwl
- [ ] Test "Create Post-Mortem" creates document in BookOwl
- [ ] Test Live Context block shows live NightOwl on-call
- [ ] `bookowl_livecontext_nightowl_errors_total` metric = 0
- [ ] ServiceMonitor scraped by Prometheus

**Rollback**

```bash
helm rollback bookowl 1 --namespace bookowl
```

Migrations are forward-only. Rollback restores the previous binary but does not reverse schema changes — design migrations to be backward-compatible (additive only).
