# BookOwl — Branding & Design System

> BookOwl uses the NightOwl design system. Read NightOwl's `docs/06-branding.md` for the full design token reference. This document covers BookOwl-specific additions only.

## 1. Product Identity

**BookOwl** — the knowledge management pillar of the NightOwl platform.

- Part of the NightOwl product family by Wisbric
- Positioned as the "book" to NightOwl's "watch" — structured knowledge vs. real-time alerting
- Tagline: "Where your operational knowledge lives."

### Logo

- Use the same owl icon as NightOwl, with "BookOwl" wordmark
- Accent color is still Wisbric green (`#00e5a0`)
- Consider an open-book motif alongside the owl for the BookOwl mark — but keep it consistent with NightOwl's geometric minimal style

### Footer

```
BookOwl v0.1.0 — A Wisbric product · Part of the NightOwl platform
```

## 2. Color Palette

Identical to NightOwl. Copy the full CSS variable set from NightOwl's `web/src/index.css` into BookOwl's `web/src/index.css`.

No new color tokens — BookOwl uses the same design system as NightOwl so the two feel like one platform.

## 3. Document Type Colors

Documents in the sidebar and search results have a colored left border by type:

| Type | Color | Token |
|------|-------|-------|
| Runbook | `#8B5CF6` (purple) | `--doc-runbook` |
| Post-mortem | `#DC2626` (red) | `--doc-postmortem` |
| SOP | `#3B82F6` (blue) | `--doc-sop` |
| ADR | `#F59E0B` (amber) | `--doc-adr` |
| Document (general) | `#6B7280` (gray) | `--doc-default` |

## 4. Editor Color Schemes

### Callout Block Colors

| Type | Background | Border | Icon |
|------|-----------|--------|------|
| Info | `#EFF6FF` / `#1E3A5F` (dark) | `#3B82F6` | ℹ️ |
| Warning | `#FFFBEB` / `#3D2E00` (dark) | `#F59E0B` | ⚠️ |
| Danger | `#FEF2F2` / `#3D0000` (dark) | `#DC2626` | 🚨 |

### Live Context Block Colors

| State | Background | Border |
|-------|-----------|--------|
| Loaded | `--card` | `--accent` (`#00e5a0`) thin left border |
| Loading | `--muted` | `--border` |
| Error/Unavailable | `--muted` | `--status-firing` dashed |

## 5. Layout

### Sidebar

BookOwl's sidebar is wider than NightOwl's (320px vs 240px) because it contains the document tree, which needs space for nested items.

```
┌──────────────────────────────────────────────────────────────┐
│ 🦉 BookOwl                              [user] [◐]           │
├─────────────────────────────────────────────────────────────┤
│  [🔍 Search docs... Cmd+K]                                   │
│                                                              │
│  SPACES                                           [+ New]   │
│  ├ 🏗️  Platform Engineering              ▾        ...       │
│  │  ├ 📁 Kubernetes                      ▾                  │
│  │  │  ├ 📄 Pod CrashLoopBackOff         ← runbook (purple) │
│  │  │  ├ 📄 OOM Kill Response            ← runbook          │
│  │  │  └ 📄 Node Not Ready               ← runbook          │
│  │  ├ 📁 Post-mortems                    ▾                  │
│  │  │  └ 📄 2026-02-20 Payment Gateway   ← postmortem (red) │
│  │  └ 📁 Architecture                   ▾                  │
│  │     └ 📄 Cluster Overview             ← document (gray)  │
│  │                                                          │
│  ├ 🇩🇪 Customer: TechGmbH               ▾                   │
│  │  └ 📁 SOPs                           ▾                   │
│  │     └ 📄 Change Management Process                        │
│  │                                                          │
│  └ 🔴 On-Call Runbooks                  ▾                   │
│     └ 📁 Alerts                         ▾                   │
│        └ 📄 Alertmanager Runbook                             │
│                                                              │
│  ─────────────────────────────────                          │
│  [⚙️ Admin]  [NightOwl ↗]                                   │
└──────────────────────────────────────────────────────────────┘
```

### Document View

```
┌─────────────────────────────────────────────────────────────┐
│  ← Platform Engineering / Kubernetes                        │
│                                                             │
│  [🦉] Pod CrashLoopBackOff Runbook       [Edit] [⋯]  [🔗]  │
│       📘 Runbook · Published · v4 · Updated 2h ago          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  [Live Context Block — On-Call]                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🦉 On-Call — DE On-Call                             │   │
│  │  Primary:   Stefan K.  · Secondary: Max M.          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  # Pod CrashLoopBackOff Runbook                             │
│                                                             │
│  ⚠️  Check logs before restarting the pod.                 │
│                                                             │
│  ## Diagnosis Steps                                         │
│  ☐ Check pod events: `kubectl describe pod ...`            │
│  ☐ Check previous logs: `kubectl logs --previous`          │
│  ☐ Check resource limits: `kubectl top pod`                │
│                                                             │
│  ...                                                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 6. Dark Mode

Dark mode is the default — identical policy to NightOwl. All colors use CSS variables. Toggle persists to localStorage.

## 7. Implementation Notes for Claude Code

1. **Copy NightOwl's design tokens** — duplicate `web/src/index.css` CSS variables from NightOwl as the starting point
2. **Configure Tailwind identically** — use the same `tailwind.config.ts` pattern from NightOwl's `docs/06-branding.md`
3. **Install Inter + JetBrains Mono** — same fonts as NightOwl
4. **Sidebar is wider** — 320px for BookOwl vs NightOwl's narrower sidebar
5. **Document type color** — add `--doc-*` tokens and use left-border color in sidebar and search results
6. **Live Context block styling** — accent green left border, subtle background from `--card`
7. **"BookOwl" wordmark** — same owl icon asset as NightOwl, different wordmark text
8. **Page titles** — `"Page Title — BookOwl"`
