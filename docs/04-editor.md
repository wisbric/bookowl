# BookOwl — Editor Specification

## 1. Editor Choice: Tiptap 2

BookOwl uses **Tiptap 2** as its block-based editor. Tiptap is built on ProseMirror and is MIT-licensed.

Why Tiptap:
- MIT license — suitable for self-hosted KRITIS environments
- Excellent React integration
- Extensible — custom block types (Live Context blocks) are first-class
- Stores content as JSON (ProseMirror doc) — easy to serialize to JSONB
- Active development, good TypeScript support

**Install:**

```bash
npm install @tiptap/react @tiptap/pm @tiptap/starter-kit @tiptap/extension-task-list @tiptap/extension-task-item @tiptap/extension-code-block-lowlight @tiptap/extension-image @tiptap/extension-table @tiptap/extension-table-row @tiptap/extension-table-cell @tiptap/extension-table-header @tiptap/extension-placeholder
npm install lowlight  # syntax highlighting for code blocks
```

## 2. Block Types

### 2.1 Standard Blocks (Tiptap built-ins + extensions)

| Block | Tiptap Extension | Notes |
|-------|-----------------|-------|
| Paragraph | `StarterKit` | Default block |
| Heading H1–H3 | `StarterKit` | Only 3 levels |
| Bullet list | `StarterKit` | Unordered |
| Ordered list | `StarterKit` | Numbered |
| Task list / checklist | `TaskList` + `TaskItem` | Checkboxes, completable |
| Code block | `CodeBlockLowlight` | Syntax highlighting via lowlight |
| Blockquote | `StarterKit` | |
| Horizontal rule | `StarterKit` | Divider |
| Image | `Image` | Upload or URL |
| Table | `Table` + `TableRow` + `TableCell` + `TableHeader` | |

### 2.2 Custom Blocks

#### Callout Block

A colored info/warning/danger box. Implemented as a custom Tiptap node.

```typescript
// CalloutBlock: wraps content in a styled container
// attrs: { type: 'info' | 'warning' | 'danger' }
// renders: <div class="callout callout-{type}">...</div>
```

Example usage in editor:
```
⚠️ Warning: Always drain the node before rescheduling critical pods.
```

#### Live Context Block

A read-only block that fetches and displays live NightOwl data. Implemented as a custom Tiptap node with a React `NodeView`.

```typescript
// LiveContextBlock: read-only, fetches from BookOwl live-context API
// attrs:
//   subtype: 'oncall' | 'alerts' | 'service-status' | 'incident-link'
//   rosterId?: string       (for oncall subtype)
//   serviceName?: string    (for service-status, alerts)
//   alertId?: string        (for incident-link)
//   label?: string          (optional display label)
```

**On-Call Live Context Block — rendered output:**

```
┌─────────────────────────────────────────────┐
│ 🦉 On-Call — DE On-Call                     │
│  Primary:   Stefan K.  (UTC+1)              │
│  Secondary: Max M.     (UTC+1)              │
│  Week of Feb 24 · Next handoff: Mar 3 09:00 │
│  [last refreshed 12s ago]                   │
└─────────────────────────────────────────────┘
```

**Service Status Live Context Block — rendered output:**

```
┌─────────────────────────────────────────────┐
│ 🔴 payment-gateway                          │
│  3 active alerts  ·  2 critical, 1 warning  │
│  Last alert: 14m ago                        │
└─────────────────────────────────────────────┘
```

Live Context blocks in the editor show a configuration popover when clicked (not when viewing). In view mode they are fully read-only and auto-refresh every 30s.

### 2.3 Slash Command Menu

Typing `/` in the editor opens a block picker. Categories:

**Basic**
- `/paragraph` or just type
- `/heading1`, `/heading2`, `/heading3`
- `/bullet-list`
- `/numbered-list`
- `/task-list`
- `/divider`

**Media & Content**
- `/code`
- `/table`
- `/image`
- `/quote`

**Callouts**
- `/info`
- `/warning`
- `/danger`

**NightOwl (Live)**
- `/oncall` — Insert On-Call block
- `/service` — Insert Service Status block
- `/alerts` — Insert Active Alerts block

## 3. Editor Component Structure

```
web/src/
├── components/
│   └── editor/
│       ├── BookOwlEditor.tsx        # Main editor component (Tiptap instance)
│       ├── EditorToolbar.tsx        # Formatting toolbar
│       ├── SlashCommandMenu.tsx     # / command picker
│       ├── extensions/
│       │   ├── CalloutBlock.ts      # Custom callout node
│       │   └── LiveContextBlock.tsx # Custom live context node (NodeView)
│       └── live-context/
│           ├── OnCallBlock.tsx      # Renders on-call live data
│           ├── ServiceStatusBlock.tsx
│           └── ActiveAlertsBlock.tsx
```

## 4. Editor Configuration

```typescript
// BookOwlEditor.tsx
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import { common, createLowlight } from 'lowlight'
import Placeholder from '@tiptap/extension-placeholder'
import { CalloutBlock } from './extensions/CalloutBlock'
import { LiveContextBlock } from './extensions/LiveContextBlock'

const editor = useEditor({
  extensions: [
    StarterKit,
    TaskList,
    TaskItem.configure({ nested: false }),
    CodeBlockLowlight.configure({ lowlight: createLowlight(common) }),
    Placeholder.configure({ placeholder: "Type '/' for commands…" }),
    CalloutBlock,
    LiveContextBlock,
  ],
  content: document.content,  // ProseMirror JSON from API
  onUpdate: ({ editor }) => {
    debouncedSave(editor.getJSON())  // debounced 1s
  },
})
```

## 5. Autosave

Documents save automatically:

- Debounced 1 second after last keystroke
- `PUT /api/v1/documents/:id` with the new content JSON
- Optimistic UI: show "Saving…" → "Saved" indicator in the header
- On error: show "Save failed — click to retry"
- A new `document_versions` snapshot is saved on every PUT (the server handles versioning)
- Version history accessible from the document header

## 6. View Mode vs Edit Mode

Documents have two modes:

**View mode** (default):
- Read-only rendered content
- Live Context blocks refresh automatically (30s)
- Task list checkboxes are interactive even in view mode (checking off steps during an incident is a core workflow)
- "Edit" button in the header switches to edit mode

**Edit mode:**
- Full Tiptap editor
- Live Context blocks show a settings icon to configure them
- Slash command menu active
- Autosave enabled

## 7. Document Header

```
┌──────────────────────────────────────────────────────────────┐
│  ← Platform Engineering / Kubernetes                         │
│                                                              │
│  [🦉] Pod CrashLoopBackOff Runbook              [Edit] [⋯]  │
│        Runbook · Published · Updated 2h ago by Stefan K.     │
└──────────────────────────────────────────────────────────────┘
```

The `[⋯]` menu contains: Version History, Duplicate, Move to Collection, Archive, Delete.

## 8. Post-Mortem Template

When NightOwl creates a post-mortem via `POST /api/v1/integration/post-mortems`, the content is pre-filled from this template:

```json
{
  "type": "doc",
  "content": [
    { "type": "heading", "attrs": {"level": 1}, "content": [{"type": "text", "text": "Post-Mortem: {incident.title}"}] },
    { "type": "paragraph", "content": [{"type": "text", "marks": [{"type": "bold"}], "text": "Date:"}, {"type": "text", "text": " {incident.created_at}"}] },
    { "type": "paragraph", "content": [{"type": "text", "marks": [{"type": "bold"}], "text": "Severity:"}, {"type": "text", "text": " {incident.severity}"}] },
    { "type": "paragraph", "content": [{"type": "text", "marks": [{"type": "bold"}], "text": "Resolved by:"}, {"type": "text", "text": " {incident.resolved_by}"}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Summary"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "{incident.solution}"}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Timeline"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "Add key events from the incident timeline here."}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Root Cause"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "{incident.root_cause}"}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Impact"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "Describe the business/user impact."}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Action Items"}] },
    { "type": "taskList", "content": [
      { "type": "taskItem", "attrs": {"checked": false}, "content": [{"type": "text", "text": "Add follow-up actions here"}] }
    ]},
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Lessons Learned"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "What did we learn? What do we do differently next time?"}] }
  ]
}
```

Template variables (`{incident.title}` etc.) are substituted server-side before the document is created.
