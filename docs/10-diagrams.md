# BookOwl — Diagrams Extension (draw.io)

## 1. Overview

BookOwl embeds diagrams.net (draw.io) as a custom Tiptap block type. Users can create and edit architecture diagrams, flowcharts, network diagrams, and incident timelines directly inside documents.

Diagrams are stored as XML in the document's Tiptap JSON content — no separate storage backend required. The diagram XML is rendered as SVG for display and edited via an embedded draw.io iframe.

For KRITIS and data-sovereign deployments, the draw.io editor is **self-hosted** in the same Kubernetes cluster. No data leaves the cluster during editing.

---

## 2. User Experience

### 2.1 Inserting a Diagram

- Type `/` in the editor → select "Diagram" from the slash command menu (category: Media & Content)
- An empty `DiagramBlock` is inserted with a placeholder "Click to create diagram"
- Clicking the placeholder opens the draw.io editor in a modal overlay

### 2.2 Editing a Diagram

- In edit mode: clicking any diagram opens the draw.io modal with the current XML loaded
- The draw.io editor runs full-screen as a modal (no separate tab or window)
- On "Save & Close": the updated XML is written back to the Tiptap node, the modal closes, and the document autosaves
- On "Discard": the modal closes with no changes

### 2.3 Viewing a Diagram

- In view mode: diagrams render as inline SVG
- SVG is generated client-side from the stored XML using the draw.io viewer library
- Diagrams are responsive (scale to container width, maintain aspect ratio)
- A small toolbar appears on hover: "Edit" (view→edit mode switch), "Download SVG", "Download PNG", "Fullscreen view"

### 2.4 Empty / Error States

- New diagram (no XML): placeholder with dashed border — "Click to create diagram · draw.io"
- Draw.io unavailable: "Diagram editor unavailable" with the last-rendered SVG still visible if XML exists
- Invalid XML: "Diagram could not be rendered" with an "Edit" button to open the raw XML for repair

---

## 3. Architecture

```
┌─────────────────────────────────────────┐
│  Browser                                │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │  Tiptap Editor                  │   │
│  │  DiagramBlock node              │   │
│  │  attrs: { xml, width, height }  │   │
│  │                ↕ postMessage    │   │
│  │  ┌──────────────────────────┐  │   │
│  │  │  draw.io iframe          │  │   │
│  │  │  (modal overlay)         │  │   │
│  │  │  src: /drawio/index.html │  │   │
│  │  └──────────────────────────┘  │   │
│  └─────────────────────────────────┘   │
│            ↓ SVG render                 │
│  ┌─────────────────────────────────┐   │
│  │  mxGraph client-side renderer   │   │
│  │  (draw.io viewer JS bundle)     │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘

No API calls for diagram data — XML lives in Tiptap JSON → PostgreSQL
```

The draw.io application is served from the same origin as BookOwl web (`/drawio/`). It is **not** proxied through the BookOwl API — the nginx container serves the static draw.io assets directly.

---

## 4. draw.io Embed Protocol

draw.io's embed mode communicates via `window.postMessage`. The protocol:

### 4.1 Init sequence

```
1. Parent opens iframe: src="/drawio/index.html?embed=1&proto=json&spin=1"
2. draw.io sends: { event: "init" }
3. Parent responds: { action: "load", xml: "<mxGraphModel>...</mxGraphModel>" }
   (or empty string for new diagram)
4. draw.io renders the diagram
```

### 4.2 Save

```
1. User clicks Save in draw.io
2. draw.io sends: { event: "save", xml: "<mxGraphModel>...</mxGraphModel>" }
3. Parent receives XML, updates Tiptap node attrs, closes modal
4. draw.io sends: { event: "exit" } (after save)
```

### 4.3 Exit without saving

```
1. User clicks Exit / presses Escape
2. draw.io sends: { event: "exit" }
3. Parent closes modal, no changes
```

### 4.4 Auto-save (optional)

```
draw.io sends: { event: "autosave", xml: "..." } periodically
Parent can choose to update the node silently (not closing the modal)
```

---

## 5. Tiptap Extension

### 5.1 Node Definition

```typescript
// web/src/components/editor/extensions/DiagramBlock.ts

import { Node, mergeAttributes } from '@tiptap/core'
import { ReactNodeViewRenderer } from '@tiptap/react'
import { DiagramBlockView } from './DiagramBlockView'

export interface DiagramBlockOptions {
  drawioUrl: string   // e.g. "/drawio" — injected from app config
}

export const DiagramBlock = Node.create<DiagramBlockOptions>({
  name: 'diagramBlock',
  group: 'block',
  atom: true,          // treated as a single unit, not editable inline
  draggable: true,
  selectable: true,

  addOptions() {
    return {
      drawioUrl: '/drawio',
    }
  },

  addAttributes() {
    return {
      xml: {
        default: '',
        parseHTML: el => el.getAttribute('data-xml') ?? '',
        renderHTML: attrs => ({ 'data-xml': attrs.xml }),
      },
      width: {
        default: null,
        parseHTML: el => el.getAttribute('data-width'),
        renderHTML: attrs => attrs.width ? { 'data-width': attrs.width } : {},
      },
      height: {
        default: null,
        parseHTML: el => el.getAttribute('data-height'),
        renderHTML: attrs => attrs.height ? { 'data-height': attrs.height } : {},
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="diagram-block"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-type': 'diagram-block' })]
  },

  addNodeView() {
    return ReactNodeViewRenderer(DiagramBlockView)
  },
})
```

### 5.2 React Node View

```typescript
// web/src/components/editor/extensions/DiagramBlockView.tsx

import { NodeViewWrapper } from '@tiptap/react'
import { useState, useEffect, useRef, useCallback } from 'react'
import { DiagramModal } from './DiagramModal'
import { renderDiagramSVG } from './diagramRenderer'

interface Props {
  node: { attrs: { xml: string; width: number | null; height: number | null } }
  updateAttributes: (attrs: Record<string, unknown>) => void
  selected: boolean
  editor: { isEditable: boolean }
}

export function DiagramBlockView({ node, updateAttributes, selected, editor }: Props) {
  const { xml, width, height } = node.attrs
  const [svg, setSvg] = useState<string>('')
  const [renderError, setRenderError] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const isEmpty = !xml

  // Render SVG from XML whenever XML changes
  useEffect(() => {
    if (!xml) return
    renderDiagramSVG(xml)
      .then(setSvg)
      .catch(() => setRenderError(true))
  }, [xml])

  const handleSave = useCallback((newXml: string) => {
    updateAttributes({ xml: newXml })
    setModalOpen(false)
  }, [updateAttributes])

  const handleClick = () => {
    if (editor.isEditable) setModalOpen(true)
  }

  return (
    <NodeViewWrapper>
      <div
        className={`diagram-block group relative my-4 rounded-lg border ${
          selected ? 'border-accent' : 'border-border'
        }`}
        style={{ width: width ? `${width}px` : '100%' }}
      >
        {isEmpty ? (
          <div
            className="flex h-32 cursor-pointer items-center justify-center rounded-lg border-2 border-dashed border-border text-sm text-muted-foreground hover:border-accent hover:text-accent"
            onClick={handleClick}
          >
            Click to create diagram · draw.io
          </div>
        ) : renderError ? (
          <div className="flex h-24 items-center justify-center gap-2 text-sm text-destructive">
            Diagram could not be rendered
            {editor.isEditable && (
              <button className="underline" onClick={handleClick}>Edit</button>
            )}
          </div>
        ) : (
          <div
            className={editor.isEditable ? 'cursor-pointer' : ''}
            onClick={handleClick}
            dangerouslySetInnerHTML={{ __html: svg }}
          />
        )}

        {/* Hover toolbar — view mode */}
        {!isEmpty && !editor.isEditable && (
          <DiagramToolbar xml={xml} svg={svg} />
        )}
      </div>

      {modalOpen && (
        <DiagramModal
          xml={xml}
          onSave={handleSave}
          onClose={() => setModalOpen(false)}
        />
      )}
    </NodeViewWrapper>
  )
}
```

### 5.3 Modal

```typescript
// web/src/components/editor/extensions/DiagramModal.tsx

import { useEffect, useRef } from 'react'

interface Props {
  xml: string
  onSave: (xml: string) => void
  onClose: () => void
}

export function DiagramModal({ xml, onSave, onClose }: Props) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const initialized = useRef(false)

  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      // Only accept messages from same origin
      if (event.origin !== window.location.origin) return

      let msg: { event: string; xml?: string }
      try {
        msg = typeof event.data === 'string' ? JSON.parse(event.data) : event.data
      } catch {
        return
      }

      if (msg.event === 'init') {
        // draw.io is ready — send the current XML
        initialized.current = true
        iframeRef.current?.contentWindow?.postMessage(
          JSON.stringify({ action: 'load', xml: xml || '' }),
          window.location.origin
        )
      }

      if (msg.event === 'save' && msg.xml) {
        onSave(msg.xml)
      }

      if (msg.event === 'exit') {
        onClose()
      }
    }

    window.addEventListener('message', handleMessage)
    return () => window.removeEventListener('message', handleMessage)
  }, [xml, onSave, onClose])

  // Close on Escape
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [onClose])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80">
      <div className="relative h-[90vh] w-[95vw] overflow-hidden rounded-xl border border-border bg-background shadow-2xl">
        <iframe
          ref={iframeRef}
          src="/drawio/index.html?embed=1&proto=json&spin=1&libraries=1&lang=en"
          className="h-full w-full border-0"
          title="draw.io diagram editor"
        />
      </div>
    </div>
  )
}
```

### 5.4 SVG Renderer

```typescript
// web/src/components/editor/extensions/diagramRenderer.ts

// draw.io viewer script (loaded once, cached)
// Source: draw.io GitHub releases — copy viewer.min.js to web/public/drawio-viewer.min.js
let viewerLoaded = false

function loadViewer(): Promise<void> {
  if (viewerLoaded) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = '/drawio-viewer.min.js'
    script.onload = () => { viewerLoaded = true; resolve() }
    script.onerror = reject
    document.head.appendChild(script)
  })
}

export async function renderDiagramSVG(xml: string): Promise<string> {
  await loadViewer()
  // mxGraph is exposed globally by viewer.min.js
  const graph = (window as unknown as { mxGraph: unknown })
  if (!graph) throw new Error('mxGraph not loaded')

  // Use the draw.io utility to convert XML → SVG
  // This is synchronous once the viewer is loaded
  return new Promise((resolve, reject) => {
    try {
      const div = document.createElement('div')
      div.style.visibility = 'hidden'
      div.style.position = 'absolute'
      document.body.appendChild(div)

      // @ts-expect-error — mxGraph global from viewer bundle
      const g = new window.mxGraph(div)
      // @ts-expect-error
      const codec = new window.mxCodec()
      // @ts-expect-error
      const doc = window.mxUtils.parseXml(xml)
      codec.decode(doc.documentElement, g.getModel())
      // @ts-expect-error
      const svgStr = window.mxUtils.getSvg(g, null, null, null, null, true)
      document.body.removeChild(div)
      resolve(new XMLSerializer().serializeToString(svgStr))
    } catch (err) {
      reject(err)
    }
  })
}
```

> **Note:** The mxGraph global API is somewhat unwieldy. An alternative is to use the draw.io export API via the self-hosted container (POST `/export` endpoint) which returns SVG server-side. This is simpler and more reliable for complex diagrams — see section 8 for details.

---

## 6. Slash Command Entry

Add to `SlashCommandMenu.tsx`:

```typescript
{
  title: 'Diagram',
  description: 'Draw.io diagram or flowchart',
  icon: <GitBranch className="h-4 w-4" />,
  category: 'Media & Content',
  command: ({ editor, range }) => {
    editor
      .chain()
      .focus()
      .deleteRange(range)
      .insertContent({ type: 'diagramBlock', attrs: { xml: '' } })
      .run()
  },
},
```

---

## 7. HTML Renderer Update

The integration renderer (`internal/integration/renderer.go`) needs to handle `diagramBlock` nodes for runbooks served to NightOwl.

For HTML export, diagrams render as a note pointing to the source document rather than inline SVG (server-side SVG generation from XML is non-trivial without a headless browser):

```go
case "diagramBlock":
    xml := ""
    if v, ok := node.Attrs["xml"].(string); ok {
        xml = v
    }
    if xml == "" {
        return ""
    }
    // Render as a placeholder — inline SVG generation requires headless browser
    return `<div class="diagram-block-placeholder" style="border:1px dashed #666;padding:12px;border-radius:4px;color:#999;font-size:13px;">` +
        `[Diagram — view in BookOwl for interactive version]` +
        `</div>`
```

If inline SVG in NightOwl is required in the future, the draw.io container's `/export` endpoint can be called server-side from the BookOwl API at render time.

---

## 8. Self-Hosted draw.io Deployment

### 8.1 Option A: Static Assets in bookowl-web (Recommended)

Copy the draw.io webapp static files into `web/public/drawio/`. These are the files from the `war/` directory of the draw.io GitHub release.

```dockerfile
# web/Dockerfile — add before the nginx stage
FROM node:22-alpine AS drawio
ARG DRAWIO_VERSION=24.7.17
RUN apk add --no-cache curl unzip && \
    curl -L "https://github.com/jgraph/drawio/releases/download/v${DRAWIO_VERSION}/draw.io-${DRAWIO_VERSION}.war" \
    -o drawio.war && \
    unzip -q drawio.war -d /drawio-assets

# In the nginx stage:
COPY --from=drawio /drawio-assets /usr/share/nginx/html/drawio
```

draw.io is then served from `/drawio/` by the same nginx container. No additional service, no ingress rule, no separate deployment.

**Tradeoff:** The draw.io war file is ~30MB, which increases the frontend image size. Only downloaded at build time.

### 8.2 Option B: Separate draw.io Service

For environments where image size is constrained or where draw.io needs independent updates:

```yaml
# docker-compose.demo.yml addition
drawio:
  image: jgraph/drawio:24.7.17
  ports:
    - "8082:8080"
  environment:
    DRAWIO_BASE_URL: http://localhost:8082
```

```yaml
# Helm values.yaml addition
drawio:
  enabled: true
  image:
    repository: jgraph/drawio
    tag: "24.7.17"
  resources:
    requests:
      cpu: 50m
      memory: 256Mi
    limits:
      cpu: 200m
      memory: 512Mi
  ingress:
    # Typically not needed — served internally
    enabled: false
```

With Option B, set `BOOKOWL_DRAWIO_URL=http://drawio:8080` and update the iframe src accordingly.

### 8.3 Recommendation

Use **Option A** for simplicity — embedding the static assets in the web container eliminates an extra service and makes the deployment self-contained. The 30MB image size increase is negligible.

The draw.io release to pin: check https://github.com/jgraph/drawio/releases for the latest stable. Pin the version in the Dockerfile ARG to avoid unexpected updates.

---

## 9. viewer.min.js

The `viewer.min.js` file (for client-side SVG rendering of stored XML) is a separate lighter bundle from the full editor. Get it from the draw.io GitHub repo:

```
https://github.com/jgraph/drawio/blob/dev/src/main/webapp/js/viewer.min.js
```

Copy to `web/public/drawio-viewer.min.js`. This is ~600KB and loaded lazily (only when a page contains a diagram block).

**Alternative — skip client-side rendering entirely:** Have the draw.io embed send the SVG back alongside the XML on save (`{ event: "save", xml: "...", svg: "..." }`). Store both in the node attributes. Render the stored SVG directly. No viewer bundle needed. This is simpler but increases document size if diagrams are complex.

The recommendation is to **store both XML and SVG** and skip the viewer bundle:

```typescript
// Updated node attributes
addAttributes() {
  return {
    xml: { default: '' },
    svg: { default: '' },   // cached SVG from last save
    width: { default: null },
    height: { default: null },
  }
}
```

On save, draw.io sends both `xml` and the rendered SVG. On load, render the cached SVG immediately (no async render needed), and regenerate via viewer only if SVG is missing.

---

## 10. Configuration

```bash
# Optional — only needed for Option B (separate draw.io service)
BOOKOWL_DRAWIO_URL=http://drawio:8080

# Leave empty to use bundled draw.io from /drawio/ path (Option A, default)
BOOKOWL_DRAWIO_URL=
```

Frontend reads this from a `/api/v1/config` endpoint (public, no auth) that returns client-side config including `drawiUrl`.

---

## 11. Data Model Changes

No new tables required. The diagram XML (and optionally SVG) is stored as node attributes in the `documents.content` JSONB column alongside all other Tiptap content.

Example document content with a diagram block:

```json
{
  "type": "doc",
  "content": [
    { "type": "heading", "attrs": { "level": 2 }, "content": [{ "type": "text", "text": "Architecture" }] },
    {
      "type": "diagramBlock",
      "attrs": {
        "xml": "<mxGraphModel><root><mxCell id=\"0\"/>...</root></mxGraphModel>",
        "svg": "<svg xmlns=\"http://www.w3.org/2000/svg\" ...>...</svg>",
        "width": null,
        "height": null
      }
    }
  ]
}
```

The `content_text` extraction (for FTS) skips `diagramBlock` nodes — there is no meaningful plain text to extract from XML.

---

## 12. Security Considerations

**XML sanitisation:** draw.io XML can contain `<script>` tags and external image references. Before storing the XML and before rendering the SVG, sanitise:

- Strip `<script>` elements
- Strip `javascript:` hrefs
- Disallow external image URLs that reference off-cluster hosts (for KRITIS — only allow relative URLs or data URIs)

Apply sanitisation in the Tiptap extension on save, before calling `updateAttributes`:

```typescript
function sanitizeDiagramXml(xml: string): string {
  const parser = new DOMParser()
  const doc = parser.parseFromString(xml, 'text/xml')
  // Remove script tags
  doc.querySelectorAll('script').forEach(el => el.remove())
  // Strip javascript: hrefs
  doc.querySelectorAll('[href^="javascript:"]').forEach(el => el.removeAttribute('href'))
  return new XMLSerializer().serializeToString(doc)
}
```

**iframe sandbox:** The draw.io iframe should run with a relaxed but bounded sandbox:

```typescript
// In DiagramModal.tsx
<iframe
  sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
  ...
/>
```

`allow-same-origin` is required for the postMessage origin check to work correctly.

---

## 13. Tasks

Add to `docs/08-tasks.md` as Phase 9:

### Phase 9: Diagrams Extension

- [ ] Download draw.io war and embed in `web/Dockerfile` (Option A)
- [ ] Download `viewer.min.js` to `web/public/drawio-viewer.min.js`
- [ ] Implement `DiagramBlock` Tiptap node (`web/src/components/editor/extensions/DiagramBlock.ts`)
- [ ] Implement `DiagramBlockView` React component with empty/render/error states
- [ ] Implement `DiagramModal` with full postMessage protocol (init, save, exit, autosave)
- [ ] Update node attrs to store both `xml` and `svg` (skip viewer bundle)
- [ ] Add XML sanitisation on save (strip scripts, javascript: hrefs)
- [ ] Add "Diagram" entry to slash command menu
- [ ] Implement `DiagramToolbar` (Download SVG, Download PNG, Fullscreen)
- [ ] Update `internal/integration/renderer.go` to handle `diagramBlock` nodes
- [ ] Update `content_text` extractor to skip `diagramBlock` nodes
- [ ] Add `BOOKOWL_DRAWIO_URL` env var + `GET /api/v1/config` public endpoint
- [ ] Update Helm chart with optional draw.io sidecar (Option B values)
- [ ] Test: create diagram, save, reload — XML and SVG persist
- [ ] Test: diagram renders correctly in view mode
- [ ] Test: NightOwl integration endpoint returns placeholder for diagram blocks
- [ ] Test: malicious XML (script tags) is stripped on save
