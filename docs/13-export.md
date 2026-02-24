# BookOwl — PDF & Document Export

## 1. Overview

Users can export any BookOwl document as a PDF or Markdown file. This is a hard requirement for German public sector and KRITIS clients — auditors, change control boards, and compliance officers need documents they can attach to tickets, print, or archive outside the system.

Export is triggered from the document `⋯` menu and from a keyboard shortcut. It runs server-side to ensure consistent rendering regardless of the user's browser or screen settings.

---

## 2. Export Formats

| Format | Use case | Implementation |
|--------|----------|----------------|
| PDF | Printing, email attachment, compliance archive | Server-side via Chromium headless (via `chromedp` Go library) |
| Markdown | Portability, Git storage, migration | Server-side render from Tiptap JSON |
| HTML | Email embedding, static archive | Server-side render, same renderer as NightOwl integration |

---

## 3. PDF Generation

### 3.1 Approach

PDF is generated server-side by rendering the document to HTML and printing it via Chromium headless. This produces pixel-perfect output matching the BookOwl UI, handles code blocks, tables, and callout styling correctly, and requires no third-party SaaS.

**Go library:** `github.com/chromedp/chromedp` — controls a headless Chromium instance via the Chrome DevTools Protocol.

**Chromium in the container:** The `bookowl` Docker image includes Chromium:

```dockerfile
# Add to Dockerfile build stage dependencies
FROM golang:1.25-alpine AS build
RUN apk add --no-cache chromium chromium-chromedriver
```

The binary path is passed to chromedp: `chromedp.ExecPath("/usr/bin/chromium-browser")`.

Alternatively, for environments where adding Chromium to the API container is undesirable, a separate `bookowl-renderer` sidecar (same image, `--mode=renderer`) can be deployed and called internally. For v0.1, embed Chromium in the API container — it's simpler and the image size increase (~150MB) is acceptable.

### 3.2 PDF Generation Flow

```
POST /api/v1/documents/:id/export?format=pdf
  → Fetch document from DB
  → Render Tiptap JSON → HTML (internal/integration renderer + print CSS)
  → Launch chromedp context with timeout (30s)
  → chromedp.Navigate to data URL (inline HTML)
  → chromedp.WaitReady
  → chromedp.ActionFunc: Page.printToPDF with options
  → Stream PDF bytes to response
  → Content-Disposition: attachment; filename="<title>.pdf"
```

### 3.3 Print CSS

The HTML rendered for PDF uses a separate print stylesheet (`internal/export/print.css`) that:
- Sets page size to A4
- Removes sidebar, toolbar, header UI chrome
- Shows document title, space/collection path, export date, and BookOwl logo in header
- Shows page numbers in footer
- Breaks page before H1/H2 headings
- Prevents orphaned headings (keep-with-next)
- Renders code blocks with monospace, light background, full-width
- Renders callout blocks with their colored left border (no background fill for print economy)
- Hides Live Context blocks (replaces with static placeholder: "Live data — view in BookOwl")
- Hides interactive elements (edit buttons, autosave indicator)

```css
/* internal/export/print.css */

@page {
  size: A4;
  margin: 2cm 2.5cm;
}

@page :first {
  margin-top: 3cm;
}

/* PDF header */
.pdf-header {
  position: running(header);
  font-size: 9pt;
  color: #666;
  border-bottom: 1px solid #ddd;
  padding-bottom: 4pt;
}

@page {
  @top-right {
    content: element(header);
  }
  @bottom-center {
    content: "Page " counter(page) " of " counter(pages);
    font-size: 8pt;
    color: #999;
  }
}

h1, h2, h3 { break-after: avoid; }
h1, h2 { break-before: auto; }
pre, code { font-family: 'JetBrains Mono', 'Courier New', monospace; }
pre { break-inside: avoid; background: #f5f5f5; padding: 12pt; border-radius: 4pt; }
.callout { border-left: 4pt solid; padding: 8pt 12pt; break-inside: avoid; }
.callout-info { border-color: #3B82F6; }
.callout-warning { border-color: #F59E0B; }
.callout-danger { border-color: #DC2626; }
.live-context-block::before { content: "[Live data — view in BookOwl]"; color: #999; font-style: italic; }
.live-context-block > * { display: none; }
```

### 3.4 PDF Page Header Content

Every PDF page header includes:
- Left: 🦉 BookOwl · `{Space name}` / `{Collection name}` / `{Document title}`
- Right: Exported `{date}` · `{user display name}`

Page footer: Page N of M

First page: Full document metadata block below the title:
```
Type: Runbook   Status: Published   Version: v4
Last updated: Feb 24, 2026 by Stefan Kloppenborg
Tags: kubernetes, incident, pod
```

### 3.5 chromedp Options

```go
// internal/export/pdf.go

func RenderPDF(ctx context.Context, html string) ([]byte, error) {
    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", true),
        chromedp.Flag("disable-gpu", true),
        chromedp.Flag("no-sandbox", true),        // required in containers
        chromedp.Flag("disable-dev-shm-usage", true),
        chromedp.Flag("disable-extensions", true),
        chromedp.ExecPath(chromiumPath()),         // from BOOKOWL_CHROMIUM_PATH or auto-detect
    )

    allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
    defer cancel()

    taskCtx, cancel := chromedp.NewContext(allocCtx)
    defer cancel()

    timeoutCtx, cancel := context.WithTimeout(taskCtx, 30*time.Second)
    defer cancel()

    var pdfBuf []byte
    dataURL := "data:text/html;charset=utf-8," + url.QueryEscape(html)

    err := chromedp.Run(timeoutCtx,
        chromedp.Navigate(dataURL),
        chromedp.WaitReady("body"),
        chromedp.ActionFunc(func(ctx context.Context) error {
            var err error
            pdfBuf, _, err = page.PrintToPDF().
                WithPrintBackground(false).
                WithPaperWidth(8.27).   // A4 width in inches
                WithPaperHeight(11.69). // A4 height in inches
                WithMarginTop(0.79).    // 2cm
                WithMarginBottom(0.79).
                WithMarginLeft(0.98).   // 2.5cm
                WithMarginRight(0.98).
                Do(ctx)
            return err
        }),
    )
    return pdfBuf, err
}
```

---

## 4. Markdown Export

Simple conversion from Tiptap JSON to Markdown. Uses the same node-walker as the plain text extractor but emits Markdown syntax:

```go
// internal/export/markdown.go

func RenderMarkdown(doc TiptapDoc) string {
    var sb strings.Builder
    renderMarkdownNode(&sb, doc.Content, 0)
    return sb.String()
}
```

Node mapping:

| Tiptap node | Markdown output |
|-------------|-----------------|
| heading h1 | `# text` |
| heading h2 | `## text` |
| heading h3 | `### text` |
| paragraph | `text\n\n` |
| bulletList | `- item\n` |
| orderedList | `1. item\n` |
| taskList | `- [ ] item` / `- [x] item` |
| codeBlock | ` ```lang\ncode\n``` ` |
| blockquote | `> text` |
| horizontalRule | `---` |
| image | `![alt](url)` |
| table | GFM table syntax |
| callout (info) | `> **ℹ️ Info:** text` |
| callout (warning) | `> **⚠️ Warning:** text` |
| callout (danger) | `> **🚨 Danger:** text` |
| liveContextBlock | `<!-- Live context: {subtype} — view in BookOwl -->` |
| diagramBlock | `<!-- Diagram — view in BookOwl -->` + fenced XML block |
| bold | `**text**` |
| italic | `_text_` |
| code (inline) | `` `text` `` |
| strike | `~~text~~` |
| link | `[text](url)` |

Image URLs in Markdown export: if the storage backend is local, image URLs are absolute (`https://bookowl.example.com/api/v1/images/:id`). This is set from `BOOKOWL_PUBLIC_URL` at export time.

---

## 5. API Endpoints

```
GET /api/v1/documents/:id/export?format=pdf
    → Content-Type: application/pdf
    → Content-Disposition: attachment; filename="{slug}.pdf"
    → Requires viewer role or above

GET /api/v1/documents/:id/export?format=markdown
    → Content-Type: text/markdown
    → Content-Disposition: attachment; filename="{slug}.md"
    → Requires viewer role or above

GET /api/v1/documents/:id/export?format=html
    → Content-Type: text/html
    → Content-Disposition: attachment; filename="{slug}.html"
    → Requires viewer role or above
```

All export endpoints:
- Require authentication (no public export links)
- Respect space visibility (private spaces require membership)
- Log to audit log: `document.exported` with format and user
- Add response header `X-Export-Version: {document_version}` for cache validation

---

## 6. Frontend — Export UI

### 6.1 Document `⋯` Menu Addition

```
[ ⋯ ]
  Version History
  Duplicate
  Move to Collection
  ─────────────────
  Export as PDF        ← new
  Export as Markdown   ← new
  ─────────────────
  Archive
  Delete
```

### 6.2 Export Behaviour

Clicking "Export as PDF":
1. Button shows spinner: "Generating PDF…"
2. Calls `GET /api/v1/documents/:id/export?format=pdf`
3. Response streams into a Blob → `URL.createObjectURL` → trigger download
4. Button resets after download starts

Keyboard shortcut: no default shortcut (avoid conflicting with browser's Ctrl+P). Users who want print can use the browser's built-in print, which will use the print CSS.

### 6.3 Export Error States

- PDF generation timeout (>30s): "Export failed — document may be too large. Try Markdown export instead."
- Chromium not available: "PDF export is not available in this deployment. Try Markdown export instead." (shown when `BOOKOWL_EXPORT_PDF_ENABLED=false`)

---

## 7. Configuration

```bash
# PDF export
BOOKOWL_EXPORT_PDF_ENABLED=true              # set false if Chromium not available
BOOKOWL_CHROMIUM_PATH=/usr/bin/chromium      # auto-detected if not set
BOOKOWL_EXPORT_PDF_TIMEOUT=30s

# Used for absolute image URLs in Markdown/HTML export
BOOKOWL_PUBLIC_URL=https://bookowl.example.com
```

### 7.1 Helm Values

```yaml
export:
  pdf:
    enabled: true
    chromiumPath: /usr/bin/chromium
    timeout: 30s
  publicUrl: https://bookowl.example.com

# If using the sidecar renderer pattern:
renderer:
  enabled: false   # set true to use a separate renderer pod
  replicas: 1
  resources:
    requests:
      cpu: 200m
      memory: 512Mi    # Chromium needs memory
    limits:
      cpu: 1000m
      memory: 1Gi
```

### 7.2 Dockerfile Note

Add to `Dockerfile`:

```dockerfile
FROM golang:1.25-alpine AS build
# Add Chromium for PDF export
RUN apk add --no-cache chromium

# ... rest of build ...

FROM gcr.io/distroless/static-debian12
# Chromium can't run in distroless — switch to debian-slim for PDF support
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /bookowl /bookowl
COPY --from=build /app/migrations /migrations
ENTRYPOINT ["/bookowl"]
```

**Note:** Chromium requires glibc — it cannot run in `distroless/static`. Switch the production stage to `debian:bookworm-slim` when PDF export is enabled. If PDF export is disabled (`BOOKOWL_EXPORT_PDF_ENABLED=false`), the distroless image can still be used.

Provide two Dockerfile targets:

```dockerfile
# Default (no PDF): distroless, smaller image
FROM gcr.io/distroless/static-debian12 AS production-slim

# With PDF: debian-slim + chromium
FROM debian:bookworm-slim AS production-full
RUN apt-get update && apt-get install -y --no-install-recommends chromium \
    && rm -rf /var/lib/apt/lists/*

# Use ARG to select:
ARG PDF_SUPPORT=false
FROM production-${PDF_SUPPORT:+full}${PDF_SUPPORT:-slim}
```

---

## 8. Tasks

Add to `docs/08-tasks.md` as Phase 13:

### Phase 13: PDF & Document Export

- [ ] Add `chromedp` dependency (`go get github.com/chromedp/chromedp`)
- [ ] `internal/export/pdf.go` — chromedp PDF renderer with 30s timeout
- [ ] `internal/export/print.css` — A4 print stylesheet
- [ ] `internal/export/markdown.go` — Tiptap JSON → Markdown converter (all node types)
- [ ] `GET /api/v1/documents/:id/export` endpoint (pdf / markdown / html)
- [ ] Audit log entry: `document.exported`
- [ ] Dockerfile: add Chromium for `production-full` target; keep slim for `production-slim`
- [ ] `BOOKOWL_EXPORT_PDF_ENABLED` config flag
- [ ] `BOOKOWL_PUBLIC_URL` config for absolute image URLs in exports
- [ ] Frontend: Export options in document `⋯` menu
- [ ] Frontend: download via Blob + createObjectURL
- [ ] Frontend: spinner state + error handling for slow/failed PDF
- [ ] Helm chart: `export.pdf.enabled` value + renderer sidecar option
- [ ] Unit tests: Markdown renderer for all node types
- [ ] Integration test: export endpoint returns correct Content-Type and Content-Disposition
- [ ] Manual test: generated PDF renders correctly (code blocks, tables, callouts, page numbers)
