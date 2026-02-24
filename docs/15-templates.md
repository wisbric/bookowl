# BookOwl — Document Templates Library

## 1. Overview

Any document in BookOwl can be saved as a template. Templates are reusable starting points for new documents. A "New document" flow can start from a blank document or from any template in the library.

This replaces the current hardcoded post-mortem template (which is used only for NightOwl integration). The hardcoded template remains for the NightOwl `POST /api/v1/integration/post-mortems` endpoint, but the library adds user-facing template management on top.

---

## 2. Template Types

| Type | Who manages | Visible to |
|------|-------------|-----------|
| **System templates** | Seeded by BookOwl, read-only | All users |
| **Space templates** | Editors/admins of a space | Members of that space |
| **Global templates** | BookOwl admins | All users in the tenant |

System templates are shipped with BookOwl and cannot be deleted (but can be hidden by admins). They cover the most common document types teams need immediately after installation.

---

## 3. System Templates (Seeded)

BookOwl ships with these templates out of the box:

| Template name | Doc type | Description |
|--------------|----------|-------------|
| Post-Mortem | post-mortem | Standard 5-section post-mortem with timeline, root cause, action items |
| Runbook | runbook | Step-by-step operational runbook with prerequisites and escalation path |
| Standard Operating Procedure | sop | SOP with scope, responsibilities, procedure steps, and exceptions |
| Architecture Decision Record | adr | ADR with context, decision, consequences (Nygard format) |
| Incident Response Checklist | runbook | Task list format for live incident response |
| Onboarding Guide | document | New team member onboarding with checklist and links |
| Meeting Notes | document | Agenda, attendees, decisions, action items |
| Change Request | document | RFC-style change request for KRITIS change control processes |

The "Change Request" template is particularly important for KRITIS clients — it matches the structure expected by German IT-Grundschutz change management processes.

---

## 4. Data Model

```sql
-- migrations/000011_create_templates.up.sql

CREATE TABLE templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    description     TEXT,
    doc_type        TEXT NOT NULL,              -- document | runbook | post-mortem | sop | adr
    content         JSONB NOT NULL,             -- Tiptap JSON
    icon            TEXT,                       -- emoji or lucide icon name
    is_system       BOOLEAN NOT NULL DEFAULT false,   -- seeded by BookOwl
    is_global       BOOLEAN NOT NULL DEFAULT false,   -- visible to all spaces
    space_id        UUID REFERENCES spaces(id), -- NULL if global/system
    created_by      UUID REFERENCES users(id),  -- NULL if system
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Either global/system (no space) or space-scoped (has space_id)
    CONSTRAINT chk_scope CHECK (
        (is_global = true AND space_id IS NULL) OR
        (is_global = false AND (is_system = true OR space_id IS NOT NULL))
    )
);

CREATE INDEX idx_templates_space ON templates(space_id);
CREATE INDEX idx_templates_global ON templates(is_global) WHERE is_global = true;
CREATE INDEX idx_templates_system ON templates(is_system) WHERE is_system = true;
```

---

## 5. API Endpoints

```
# Template library
GET  /api/v1/templates
     ?space_id=<id>     — include space templates for this space
     ?doc_type=<type>   — filter by doc type
     ?q=<text>          — search by name/description
     → Returns: system templates + global templates + space templates (if space_id given)
     → Sorted: system first, then global, then space, each group by sort_order

GET  /api/v1/templates/:id
     → Single template with full content JSON

POST /api/v1/templates
     → Create a new template
     → Body: { name, description, doc_type, content, icon, is_global, space_id }
     → is_global=true requires admin role
     → space_id requires editor role in that space

PUT  /api/v1/templates/:id
     → Update template (cannot update system templates)
     → name, description, content, icon, sort_order

DELETE /api/v1/templates/:id
     → Delete template (cannot delete system templates)
     → Requires: admin (global) or editor in space (space template)

# Save-as-template action on a document
POST /api/v1/documents/:id/save-as-template
     → Body: { name, description, is_global, space_id }
     → Creates a template with the document's current content
     → Returns created template

# Create document from template
POST /api/v1/documents
     → Existing endpoint, add optional body field: template_id
     → If template_id present: pre-fill content from template
     → title defaults to template name (user can change)
```

---

## 6. Frontend

### 6.1 New Document Flow

The "New Document" button in a collection now opens a dialog with two tabs:

```
┌──────────────────────────────────────────────────────┐
│  New Document                                        │
│                                                      │
│  [ Blank ]  [ From template ]                        │
│                                                      │
│  ── From template ──────────────────────────────     │
│                                                      │
│  [ Search templates...                          ]    │
│                                                      │
│  System                                              │
│  ┌──────────────────┐  ┌──────────────────┐         │
│  │ 📋 Post-Mortem   │  │ 📖 Runbook       │         │
│  │ post-mortem      │  │ runbook          │         │
│  └──────────────────┘  └──────────────────┘         │
│  ┌──────────────────┐  ┌──────────────────┐         │
│  │ 📝 SOP           │  │ 🏛️ ADR            │         │
│  └──────────────────┘  └──────────────────┘         │
│                                                      │
│  This space                                          │
│  ┌──────────────────┐                               │
│  │ 🔒 K8s Incident  │                               │
│  │ runbook          │                               │
│  └──────────────────┘                               │
│                                                      │
│  Document title                                      │
│  ┌────────────────────────────────────────────────┐  │
│  │ Post-Mortem: <incident title>                  │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  [ Cancel ]                  [ Create document ]     │
└──────────────────────────────────────────────────────┘
```

### 6.2 Template Preview

Hovering over a template card shows a popover preview of the template content (rendered read-only, truncated to ~300px height with a fade).

Clicking a template card selects it (highlighted border). The title field auto-fills with the template name.

### 6.3 Templates Library Page

Route: `/templates`

An admin/editor view of all templates in the tenant, grouped by scope. Accessible from the sidebar footer or from Admin settings.

```
┌─────────────────────────────────────────────────────────────┐
│  Templates                          [ + New template ]      │
│                                                             │
│  ── System (8) ─────────────────────────────────────────   │
│  Post-Mortem · post-mortem                         [ Use ]  │
│  Runbook · runbook                                 [ Use ]  │
│  SOP · sop                                         [ Use ]  │
│  ADR · adr                                         [ Use ]  │
│  Incident Response Checklist · runbook             [ Use ]  │
│  Onboarding Guide · document                       [ Use ]  │
│  Meeting Notes · document                          [ Use ]  │
│  Change Request · document                         [ Use ]  │
│                                                             │
│  ── Global (2) ─────────────────────────────────────────   │
│  Weekly Status Update · document          [ Edit ] [ Delete ]│
│  Vendor RFP Template · document           [ Edit ] [ Delete ]│
│                                                             │
│  ── Platform Engineering (1) ────────────────────────────  │
│  K8s Incident Runbook · runbook           [ Edit ] [ Delete ]│
└─────────────────────────────────────────────────────────────┘
```

### 6.4 Save as Template

In the document `⋯` menu, add:

```
Save as Template
```

Opens a dialog:

```
┌────────────────────────────────────────────────┐
│  Save as Template                               │
│                                                 │
│  Template name                                  │
│  ┌──────────────────────────────────────────┐  │
│  │ K8s Incident Runbook                     │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  Description (optional)                         │
│  ┌──────────────────────────────────────────┐  │
│  │                                          │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  Make available to                              │
│  ● This space only (Platform Engineering)       │
│  ○ All spaces (global)       ← admins only      │
│                                                 │
│  [ Cancel ]         [ Save as template ]        │
└────────────────────────────────────────────────┘
```

### 6.5 Template Variables (Optional Enhancement)

Templates can include variable placeholders in the format `{{variable_name}}` that are highlighted in the editor after creation from a template. Users fill them in before saving.

Example post-mortem template with variables:
```
# Post-Mortem: {{incident_title}}

**Date:** {{date}}
**Severity:** {{severity}}
**Resolved by:** {{resolved_by}}
```

When a document is created from a template containing `{{...}}` patterns, the editor automatically selects the first placeholder and cycles through them on Tab key. This is a lightweight alternative to a form-based template system.

Variable detection: scan the Tiptap JSON content for text nodes matching `{{word_chars}}`. Highlight them with a custom mark in the editor. On first load, jump to the first variable.

---

## 7. NightOwl Integration Update

The `POST /api/v1/integration/post-mortems` endpoint currently uses a hardcoded template. Update it to look up the system "Post-Mortem" template from the templates table, falling back to the hardcoded template if not found:

```go
func (s *Service) CreatePostMortem(ctx context.Context, req PostMortemRequest) (*Document, error) {
    // Try to find the system post-mortem template
    tmpl, err := s.templates.GetSystemTemplate(ctx, "post-mortem")
    if err != nil {
        // Fall back to hardcoded template
        tmpl = defaultPostMortemTemplate()
    }
    content := substituteVariables(tmpl.Content, req.IncidentData)
    // ... create document
}
```

This means operators can customise the post-mortem template used by NightOwl by editing the system template in BookOwl's admin interface, without code changes.

---

## 8. Configuration

No new environment variables. Templates are per-tenant data, managed entirely through the API and UI.

The system templates are seeded on tenant creation via the seed command. They are stored in the tenant schema just like user-created templates, with `is_system=true`.

System template content is stored in `internal/seed/templates/` as JSON files (one per template), loaded at seed time:

```
internal/seed/templates/
├── post-mortem.json
├── runbook.json
├── sop.json
├── adr.json
├── incident-response-checklist.json
├── onboarding-guide.json
├── meeting-notes.json
└── change-request.json
```

---

## 9. Tasks

Add to `docs/08-tasks.md` as Phase 15:

### Phase 15: Document Templates

**Backend:**
- [ ] Migration: `templates` table
- [ ] `pkg/template` package: template.go, handler.go, service.go, store.go
- [ ] `GET /api/v1/templates` with space_id + doc_type + search filters
- [ ] `POST /api/v1/templates` — create template
- [ ] `PUT /api/v1/templates/:id` — update (block system template edits)
- [ ] `DELETE /api/v1/templates/:id` — delete (block system templates)
- [ ] `POST /api/v1/documents/:id/save-as-template`
- [ ] Update `POST /api/v1/documents` to accept `template_id`
- [ ] 8 system template JSON files in `internal/seed/templates/`
- [ ] Seed: insert system templates on tenant creation
- [ ] Update NightOwl integration post-mortem to use template table with hardcoded fallback
- [ ] Unit tests: template scope validation, system template protection

**Frontend:**
- [ ] Update "New Document" dialog to two-tab flow (Blank / From template)
- [ ] Template card grid with hover preview popover
- [ ] Template selection and title auto-fill
- [ ] `/templates` library page with grouped list (system / global / space)
- [ ] "Save as Template" in document `⋯` menu
- [ ] Save-as-template dialog with scope selection
- [ ] Template variable highlighting in editor (`{{variable}}` detection)
- [ ] Tab-cycle through template variables on new document creation
