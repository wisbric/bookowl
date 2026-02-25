# BookOwl — Real-Time Collaborative Editing

## 1. Overview

Multiple users can edit the same document simultaneously with live cursor presence,
character-level conflict-free merging, and no "someone else is editing" locks.
The experience matches CodiMD / HedgeDoc — you see other users' cursors and
changes appear in real time without any coordination required.

**Stack:**
- **Yjs** — CRDT library for conflict-free merging (already used by Tiptap Collaboration)
- **Tiptap Collaboration** — Tiptap's first-party extension that wires Yjs into the editor
- **Hocuspocus** — WebSocket server built specifically for Tiptap + Yjs (by the Tiptap team)
- **Redis** — Shared Yjs document state across Hocuspocus replicas (via `@hocuspocus/extension-redis`)
- **PostgreSQL** — Persistent Yjs binary snapshots (via `@hocuspocus/extension-database`)

---

## 2. Architecture

```
Browser (User A)                Browser (User B)
  Tiptap + CollabExtension        Tiptap + CollabExtension
  WebSocket ──────────────────────────────── WebSocket
                        │
                 bookowl-collab
                 (Hocuspocus, Node.js)
                 ws://collab-internal/
                        │
              ┌─────────┴──────────┐
              Redis                PostgreSQL
         (live Yjs state)     (persisted snapshots)
              │
         bookowl-collab
         (replica 2, if scaled)
```

The Go API (`bookowl-api`) is not involved in the WebSocket flow at all. Hocuspocus
handles auth by validating the same `bw_session` JWT that the Go API uses.
Document saves flow through Hocuspocus → PostgreSQL directly, bypassing the Go API.
The Go API remains the source of truth for document metadata (title, type, tags,
version history) — only the Tiptap content JSON is owned by Hocuspocus.

### 2.1 Coexistence With Existing Save Flow

Currently: `PUT /api/v1/documents/:id` saves Tiptap JSON content.

With collab: Hocuspocus owns live content. The Go API save endpoint remains for:
- Non-collaborative edits (API integrations, NightOwl post-mortem creation)
- Metadata updates (title, tags, collection — these never go through Hocuspocus)

On Hocuspocus persist (every 30s of activity + on all users disconnect):
- Converts Yjs doc → Tiptap JSON
- Writes to `documents.content` in PostgreSQL directly (not via API)
- Writes Yjs binary snapshot to `document_yjs_state` table for fast rehydration

On document load by a new user:
- Hocuspocus checks `document_yjs_state` for existing Yjs snapshot
- If found: loads snapshot, new user gets current state immediately
- If not found: converts `documents.content` JSON → initial Yjs doc (bootstraps from existing content)

---

## 3. New Service: bookowl-collab

A small Node.js service using Hocuspocus server. Lives in `collab/` directory.

```
collab/
├── package.json
├── Dockerfile
├── src/
│   └── server.ts        ← everything in one file, it's small
```

### 3.1 server.ts

```typescript
import { Server } from '@hocuspocus/server'
import { Redis } from '@hocuspocus/extension-redis'
import { Database } from '@hocuspocus/extension-database'
import { Logger } from '@hocuspocus/extension-logger'
import * as Y from 'yjs'
import { Pool } from 'pg'
import * as jwt from 'jsonwebtoken'

const pool = new Pool({ connectionString: process.env.DATABASE_URL })
const SECRET = process.env.BOOKOWL_SECRET_KEY!

const server = Server.configure({
  port: 1234,
  address: '0.0.0.0',

  // Auth: validate bw_session JWT passed as token param
  async onAuthenticate({ token, documentName }) {
    if (!token) throw new Error('unauthenticated')
    try {
      const claims = jwt.verify(token, SECRET) as any
      // documentName format: "{tenantSlug}/{documentId}"
      const [tenantSlug] = documentName.split('/')
      if (claims.tenant !== tenantSlug) throw new Error('wrong tenant')
      return { user: claims }  // attached to context
    } catch {
      throw new Error('invalid token')
    }
  },

  // Awareness: broadcast cursor positions and user info
  async onConnect({ documentName, context }) {
    return {
      user: {
        id: context.user.sub,
        name: context.user.name,
        color: userColor(context.user.sub),  // deterministic color from user ID
      }
    }
  },

  extensions: [
    new Logger(),

    // Redis: share document state across replicas
    new Redis({
      host: process.env.REDIS_HOST!,
      port: parseInt(process.env.REDIS_PORT || '6379'),
      password: process.env.REDIS_PASSWORD,
    }),

    // Database: persist to PostgreSQL
    new Database({
      // Load: try Yjs snapshot first, fall back to converting existing content JSON
      async fetch({ documentName }) {
        const [tenantSlug, documentId] = documentName.split('/')
        const schema = `tenant_${tenantSlug}`

        // Try Yjs snapshot
        const snap = await pool.query(
          `SELECT yjs_state FROM ${schema}.document_yjs_state WHERE document_id = $1`,
          [documentId]
        )
        if (snap.rows[0]?.yjs_state) {
          return snap.rows[0].yjs_state  // Buffer
        }

        // Bootstrap from existing Tiptap JSON content
        const doc = await pool.query(
          `SELECT content FROM ${schema}.documents WHERE id = $1`,
          [documentId]
        )
        if (!doc.rows[0]) return null

        // Convert Tiptap JSON → Yjs doc
        const ydoc = new Y.Doc()
        const { prosemirrorJSONToYDoc } = await import('y-prosemirror')
        prosemirrorJSONToYDoc(schema, doc.rows[0].content, ydoc)
        return Buffer.from(Y.encodeStateAsUpdate(ydoc))
      },

      // Store: persist Yjs state and update documents.content
      async store({ documentName, state, document }) {
        const [tenantSlug, documentId] = documentName.split('/')
        const schema = `tenant_${tenantSlug}`

        // Save Yjs binary snapshot
        await pool.query(
          `INSERT INTO ${schema}.document_yjs_state (document_id, yjs_state, updated_at)
           VALUES ($1, $2, now())
           ON CONFLICT (document_id) DO UPDATE SET yjs_state = $2, updated_at = now()`,
          [documentId, Buffer.from(state)]
        )

        // Convert Yjs → Tiptap JSON and update documents.content
        const { yDocToProsemirrorJSON } = await import('y-prosemirror')
        const content = yDocToProsemirrorJSON(document)
        await pool.query(
          `UPDATE ${schema}.documents SET content = $1, updated_at = now() WHERE id = $2`,
          [JSON.stringify(content), documentId]
        )
      },
    }),
  ],
})

server.listen()

// Deterministic color from user ID for cursor display
function userColor(userId: string): string {
  const colors = ['#3B82F6','#10B981','#F59E0B','#EF4444','#8B5CF6','#EC4899','#06B6D4','#84CC16']
  const hash = userId.split('').reduce((a, c) => a + c.charCodeAt(0), 0)
  return colors[hash % colors.length]
}
```

### 3.2 package.json

```json
{
  "name": "bookowl-collab",
  "version": "0.1.0",
  "scripts": {
    "start": "node dist/server.js",
    "build": "tsc",
    "dev": "ts-node src/server.ts"
  },
  "dependencies": {
    "@hocuspocus/server": "^2.13.0",
    "@hocuspocus/extension-redis": "^2.13.0",
    "@hocuspocus/extension-database": "^2.13.0",
    "@hocuspocus/extension-logger": "^2.13.0",
    "pg": "^8.11.0",
    "y-prosemirror": "^1.2.5",
    "yjs": "^13.6.0",
    "jsonwebtoken": "^9.0.0"
  },
  "devDependencies": {
    "@types/pg": "^8.11.0",
    "@types/jsonwebtoken": "^9.0.0",
    "typescript": "^5.3.0",
    "ts-node": "^10.9.0"
  }
}
```

### 3.3 Dockerfile

```dockerfile
FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
COPY --from=build /app/dist ./dist
COPY --from=build /app/node_modules ./node_modules
EXPOSE 1234
CMD ["node", "dist/server.js"]
```

---

## 4. Database Changes

```sql
-- migrations/000012_create_yjs_state.up.sql

CREATE TABLE document_yjs_state (
    document_id   UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    yjs_state     BYTEA NOT NULL,     -- Yjs binary update encoded with Y.encodeStateAsUpdate
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

No other schema changes. The existing `documents.content` column remains and stays
in sync via Hocuspocus's store hook.

---

## 5. Frontend Changes

### 5.1 New Dependencies

```bash
npm install @tiptap/extension-collaboration @tiptap/extension-collaboration-cursor
npm install @hocuspocus/provider yjs
```

### 5.2 Collab Provider Hook

```typescript
// web/src/hooks/useCollabProvider.ts

import { HocuspocusProvider } from '@hocuspocus/provider'
import { useEffect, useRef } from 'react'
import * as Y from 'yjs'

export function useCollabProvider(documentId: string, tenantSlug: string, sessionToken: string) {
  const ydoc = useRef(new Y.Doc())
  const provider = useRef<HocuspocusProvider | null>(null)

  useEffect(() => {
    const wsUrl = import.meta.env.VITE_COLLAB_WS_URL || 'wss://collab.bookowl.example.com'

    provider.current = new HocuspocusProvider({
      url: wsUrl,
      name: `${tenantSlug}/${documentId}`,
      document: ydoc.current,
      token: sessionToken,                  // bw_session JWT value
      onAuthenticationFailed: () => {
        console.error('Collab auth failed')
      },
    })

    return () => {
      provider.current?.destroy()
    }
  }, [documentId, tenantSlug, sessionToken])

  return { ydoc: ydoc.current, provider: provider.current }
}
```

### 5.3 Updated BookOwlEditor

```typescript
// web/src/components/editor/BookOwlEditor.tsx

import { Collaboration } from '@tiptap/extension-collaboration'
import { CollaborationCursor } from '@tiptap/extension-collaboration-cursor'
import { useCollabProvider } from '../../hooks/useCollabProvider'
import { useAuth } from '../../hooks/useAuth'

export function BookOwlEditor({ document, spaceId }: Props) {
  const { user, sessionToken, tenant } = useAuth()
  const { ydoc, provider } = useCollabProvider(document.id, tenant, sessionToken)

  const editor = useEditor({
    extensions: [
      // Remove StarterKit history — Yjs handles undo/redo
      StarterKit.configure({ history: false }),
      TaskList,
      TaskItem.configure({ nested: false }),
      CodeBlockLowlight.configure({ lowlight: createLowlight(common) }),
      Placeholder.configure({ placeholder: "Type '/' for commands…" }),
      CalloutBlock,
      LiveContextBlock,

      // Collab extensions — replace the old content prop
      Collaboration.configure({ document: ydoc }),
      CollaborationCursor.configure({
        provider,
        user: {
          name: user.displayName,
          color: user.cursorColor,  // pre-assigned server-side, stored in user context
        },
      }),
    ],
    // No content prop — Yjs owns the content now
  })

  // ... rest of editor
}
```

**Key change:** Remove the `content: document.content` prop from `useEditor`. Yjs
bootstraps the content from the server state via Hocuspocus. Also remove the
existing autosave `onUpdate` debounced PUT — Hocuspocus handles persistence now.

### 5.4 Presence Avatars

Show who is currently viewing/editing the document in the document header, right of the title:

```
┌──────────────────────────────────────────────────────────────────────┐
│  Pod CrashLoopBackOff Runbook          [SK] [MK] +1   [Edit] [⋯]   │
└──────────────────────────────────────────────────────────────────────┘
```

Each avatar is a coloured circle with initials, matching that user's cursor colour.
Tooltip on hover: "Stefan Kloppenborg — editing now".

```typescript
// web/src/components/editor/PresenceAvatars.tsx

import { useEffect, useState } from 'react'
import type { HocuspocusProvider } from '@hocuspocus/provider'

interface ActiveUser {
  id: string
  name: string
  color: string
}

export function PresenceAvatars({ provider }: { provider: HocuspocusProvider | null }) {
  const [users, setUsers] = useState<ActiveUser[]>([])

  useEffect(() => {
    if (!provider) return

    const update = () => {
      const states = Array.from(provider.awareness.getStates().values())
      setUsers(states.map(s => s.user).filter(Boolean))
    }

    provider.awareness.on('change', update)
    return () => provider.awareness.off('change', update)
  }, [provider])

  const visible = users.slice(0, 3)
  const overflow = users.length - 3

  return (
    <div className="flex items-center gap-1">
      {visible.map(u => (
        <div
          key={u.id}
          className="w-7 h-7 rounded-full flex items-center justify-center text-xs text-white font-medium"
          style={{ backgroundColor: u.color }}
          title={`${u.name} — editing now`}
        >
          {initials(u.name)}
        </div>
      ))}
      {overflow > 0 && (
        <div className="w-7 h-7 rounded-full bg-muted flex items-center justify-center text-xs">
          +{overflow}
        </div>
      )}
    </div>
  )
}
```

### 5.5 Collab Undo/Redo

Yjs has its own undo manager scoped per user — each user's undo only undoes their
own changes, not other users' changes. This is the correct behaviour for collaborative
editors (same as Google Docs).

```typescript
// Already handled by Tiptap's Collaboration extension
// Cmd+Z undoes your own changes, Cmd+Shift+Z redoes them
// No additional code needed
```

---

## 6. Nginx / Ingress Changes

The collab WebSocket needs its own subdomain or path. Recommended: separate subdomain.

```yaml
# Ingress for collab WebSocket
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: bookowl-collab
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-http-version: "1.1"
    nginx.ingress.kubernetes.io/configuration-snippet: |
      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection "upgrade";
spec:
  rules:
    - host: collab.bookowl.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: bookowl-collab
                port:
                  number: 1234
```

**Session affinity:** Hocuspocus + Redis handles cross-replica state, so session
affinity is NOT required. Any replica can serve any WebSocket connection.

---

## 7. Helm Chart Additions

```yaml
# values.yaml additions

collab:
  enabled: true
  image:
    repository: ghcr.io/wisbric/bookowl-collab
    tag: latest
  replicas: 2
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi
  wsUrl: wss://collab.bookowl.example.com  # exposed to frontend via ConfigMap
  ingress:
    host: collab.bookowl.example.com
```

Add to `deploy/helm/bookowl/templates/`:
- `deployment-collab.yaml`
- `service-collab.yaml`
- `ingress-collab.yaml`

---

## 8. docker-compose.yml Addition

For local development:

```yaml
# Add to docker-compose.yml

bookowl-collab:
  build:
    context: ./collab
    dockerfile: Dockerfile
  ports:
    - "1234:1234"
  environment:
    DATABASE_URL: postgres://bookowl:bookowl@postgres:5432/bookowl
    REDIS_HOST: redis
    REDIS_PORT: 6379
    BOOKOWL_SECRET_KEY: dev-secret-key
  depends_on:
    - postgres
    - redis
```

Frontend dev: set `VITE_COLLAB_WS_URL=ws://localhost:1234` in `.env.local`.

---

## 9. Migration From Autosave to Hocuspocus Persistence

The existing autosave (`onUpdate` debounce → `PUT /api/v1/documents/:id`) must be
removed from `BookOwlEditor.tsx` when collab is enabled. Hocuspocus's `store` hook
replaces it.

Keep the PUT endpoint functional for:
- NightOwl post-mortem creation (already uses the API directly)
- Any future non-browser clients
- Metadata-only updates (title, tags, collection — Hocuspocus doesn't touch these)

---

## 10. Conflict With Existing Version History

Currently every `PUT /api/v1/documents/:id` creates a `document_versions` snapshot.

With Hocuspocus, content saves happen via the `store` hook bypassing the Go API.
Two options:

**Option A (recommended for v1):** Call the Go API's internal version snapshot
function from Hocuspocus's `store` hook using an internal service-to-service call
with a special `X-Internal-Token` header. Triggered on every `store` call.

**Option B:** Drop per-save versioning, replace with scheduled snapshots (e.g.
every hour via a background job). Simpler but loses fine-grained version history.

Go with Option A — the version history is a valued feature for KRITIS compliance
(document audit trail), so preserving it is worth the extra call.

---

## 11. Environment Variables

```bash
# collab service
DATABASE_URL=postgres://...           # same DB as bookowl-api
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=                       # from secret
BOOKOWL_SECRET_KEY=                   # same key as bookowl-api (JWT validation)
PORT=1234

# bookowl-api (new)
BOOKOWL_COLLAB_ENABLED=true
BOOKOWL_COLLAB_INTERNAL_URL=http://bookowl-collab:1234  # for version snapshot calls

# frontend (new)
VITE_COLLAB_WS_URL=wss://collab.bookowl.example.com
```

---

## 12. Tasks

Add to `docs/08-tasks.md` as Phase 17:

### Phase 17: Real-Time Collaborative Editing

**New service (collab/):**
- [ ] `collab/package.json` — hocuspocus server, redis extension, database extension, pg, yjs, jsonwebtoken
- [ ] `collab/src/server.ts` — Hocuspocus server with auth, Redis, Database extensions
- [ ] `collab/Dockerfile` — Node 22 alpine, two-stage build
- [ ] JWT validation in `onAuthenticate` — reuse same secret as Go API
- [ ] Database `fetch` — load Yjs snapshot or bootstrap from existing Tiptap JSON
- [ ] Database `store` — persist Yjs snapshot + convert back to Tiptap JSON for documents table
- [ ] Version snapshot: call Go API internal endpoint on each `store` to preserve version history
- [ ] Add to `docker-compose.yml`

**Database:**
- [ ] Migration: `document_yjs_state` table (per-tenant schema)

**Frontend:**
- [ ] `npm install @tiptap/extension-collaboration @tiptap/extension-collaboration-cursor @hocuspocus/provider yjs`
- [ ] `web/src/hooks/useCollabProvider.ts` — Hocuspocus provider, connect with JWT token
- [ ] Update `BookOwlEditor.tsx` — add Collaboration + CollaborationCursor extensions, remove content prop, remove autosave onUpdate
- [ ] `web/src/components/editor/PresenceAvatars.tsx` — live presence avatars in document header
- [ ] Wire `VITE_COLLAB_WS_URL` via env + Helm configmap
- [ ] Collab-aware undo/redo (per-user, already handled by Tiptap Collaboration extension)

**Infrastructure:**
- [ ] `deploy/helm/bookowl/templates/deployment-collab.yaml`
- [ ] `deploy/helm/bookowl/templates/service-collab.yaml`
- [ ] `deploy/helm/bookowl/templates/ingress-collab.yaml` with WebSocket annotations
- [ ] `values.yaml` — collab section with replicas, resources, wsUrl
- [ ] `.github/workflows/release.yml` — add collab Docker build + push

**Testing:**
- [ ] Manual: open same document in two browsers, type simultaneously — no conflicts
- [ ] Manual: disconnect one browser, reconnect — gets latest state
- [ ] Manual: offline edit, reconnect — changes merge correctly
- [ ] Manual: presence avatars appear/disappear as users join/leave
- [ ] Manual: version history still shows snapshots after collab edits
- [ ] Manual: reload page after collab edit — content persisted correctly
