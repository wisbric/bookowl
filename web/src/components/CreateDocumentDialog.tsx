import { useState, useMemo } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { X, FileText, LayoutTemplate, Search } from 'lucide-react'
import { api } from '@/api/client'
import type { Document, Template } from '@/api/client'

interface CreateDocumentDialogProps {
  open: boolean
  spaceId: string
  collectionId?: string
  onClose: () => void
}

const DOC_TYPES = [
  { value: 'page', label: 'Page' },
  { value: 'runbook', label: 'Runbook' },
  { value: 'post-mortem', label: 'Post-Mortem' },
  { value: 'sop', label: 'SOP' },
  { value: 'adr', label: 'ADR' },
]

type Tab = 'blank' | 'template'

export function CreateDocumentDialog({
  open,
  spaceId,
  collectionId,
  onClose,
}: CreateDocumentDialogProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<Tab>('blank')
  const [title, setTitle] = useState('')
  const [docType, setDocType] = useState('page')
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)
  const [templateSearch, setTemplateSearch] = useState('')

  const { data: templates } = useQuery({
    queryKey: ['templates', spaceId],
    queryFn: () => api.get<Template[]>(`/templates?space_id=${spaceId}`),
    enabled: open,
  })

  const filtered = useMemo(() => {
    if (!templates) return []
    if (!templateSearch) return templates
    const q = templateSearch.toLowerCase()
    return templates.filter(
      (t) =>
        t.name.toLowerCase().includes(q) ||
        t.doc_type.toLowerCase().includes(q) ||
        t.description?.toLowerCase().includes(q),
    )
  }, [templates, templateSearch])

  const grouped = useMemo(() => {
    const system: Template[] = []
    const global: Template[] = []
    const space: Template[] = []
    for (const t of filtered) {
      if (t.is_system) system.push(t)
      else if (t.is_global) global.push(t)
      else space.push(t)
    }
    return { system, global, space }
  }, [filtered])

  const handleSelectTemplate = (t: Template) => {
    setSelectedTemplate(t)
    if (!title) setTitle(t.name)
    setDocType(t.doc_type)
  }

  const createMutation = useMutation({
    mutationFn: () => {
      const slug = title
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '')
      if (tab === 'template' && selectedTemplate) {
        return api.post<Document>('/documents', {
          space_id: spaceId,
          collection_id: collectionId,
          title,
          slug,
          doc_type: selectedTemplate.doc_type,
          template_id: selectedTemplate.id,
          content: selectedTemplate.content,
        })
      }
      return api.post<Document>('/documents', {
        space_id: spaceId,
        collection_id: collectionId,
        title,
        slug,
        doc_type: docType,
        content: { type: 'doc', content: [{ type: 'paragraph' }] },
      })
    },
    onSuccess: (doc) => {
      queryClient.invalidateQueries({ queryKey: ['space-tree', spaceId] })
      resetState()
      onClose()
      navigate({
        to: '/spaces/$spaceId/docs/$docId',
        params: { spaceId, docId: doc.id },
      })
    },
  })

  const resetState = () => {
    setTitle('')
    setDocType('page')
    setSelectedTemplate(null)
    setTemplateSearch('')
    setTab('blank')
  }

  const handleClose = () => {
    resetState()
    onClose()
  }

  if (!open) return null

  const canSubmit =
    title.trim() &&
    !createMutation.isPending &&
    (tab === 'blank' || selectedTemplate)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/60" onClick={handleClose} />
      <div className="relative w-full max-w-lg rounded-xl border border-border bg-card p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">New Document</h2>
          <button
            onClick={handleClose}
            className="rounded p-1 text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Tabs */}
        <div className="mb-4 flex gap-1 rounded-lg bg-muted p-1">
          <button
            type="button"
            onClick={() => {
              setTab('blank')
              setSelectedTemplate(null)
            }}
            className={`flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              tab === 'blank'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <FileText className="h-3.5 w-3.5" />
            Blank
          </button>
          <button
            type="button"
            onClick={() => setTab('template')}
            className={`flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              tab === 'template'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <LayoutTemplate className="h-3.5 w-3.5" />
            From template
          </button>
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (canSubmit) createMutation.mutate()
          }}
          className="space-y-4"
        >
          {tab === 'template' && (
            <>
              {/* Template search */}
              <div className="relative">
                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  type="text"
                  value={templateSearch}
                  onChange={(e) => setTemplateSearch(e.target.value)}
                  placeholder="Search templates..."
                  className="w-full rounded-md border border-border bg-background pl-9 pr-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
                />
              </div>

              {/* Template grid */}
              <div className="max-h-56 overflow-y-auto space-y-3">
                {grouped.system.length > 0 && (
                  <TemplateGroup
                    label="System"
                    templates={grouped.system}
                    selected={selectedTemplate}
                    onSelect={handleSelectTemplate}
                  />
                )}
                {grouped.global.length > 0 && (
                  <TemplateGroup
                    label="Global"
                    templates={grouped.global}
                    selected={selectedTemplate}
                    onSelect={handleSelectTemplate}
                  />
                )}
                {grouped.space.length > 0 && (
                  <TemplateGroup
                    label="This space"
                    templates={grouped.space}
                    selected={selectedTemplate}
                    onSelect={handleSelectTemplate}
                  />
                )}
                {filtered.length === 0 && (
                  <p className="py-4 text-center text-sm text-muted-foreground">
                    No templates found.
                  </p>
                )}
              </div>
            </>
          )}

          {/* Title */}
          <div>
            <label className="mb-1 block text-sm font-medium">Title</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Document title"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              autoFocus={tab === 'blank'}
            />
          </div>

          {/* Type selector (blank tab only) */}
          {tab === 'blank' && (
            <div>
              <label className="mb-1 block text-sm font-medium">Type</label>
              <select
                value={docType}
                onChange={(e) => setDocType(e.target.value)}
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              >
                {DOC_TYPES.map((dt) => (
                  <option key={dt.value} value={dt.value}>
                    {dt.label}
                  </option>
                ))}
              </select>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={handleClose}
              className="rounded-md border border-border px-4 py-2 text-sm hover:bg-muted"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!canSubmit}
              className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-foreground hover:bg-accent/90 disabled:opacity-50"
            >
              {createMutation.isPending ? 'Creating...' : 'Create'}
            </button>
          </div>

          {createMutation.isError && (
            <p className="text-sm text-destructive">
              Failed to create document. Please try again.
            </p>
          )}
        </form>
      </div>
    </div>
  )
}

function TemplateGroup({
  label,
  templates,
  selected,
  onSelect,
}: {
  label: string
  templates: Template[]
  selected: Template | null
  onSelect: (t: Template) => void
}) {
  return (
    <div>
      <div className="mb-1.5 text-xs font-medium uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="grid grid-cols-2 gap-2">
        {templates.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => onSelect(t)}
            className={`rounded-lg border p-3 text-left text-sm transition-colors ${
              selected?.id === t.id
                ? 'border-accent bg-accent/10 ring-1 ring-accent'
                : 'border-border hover:border-accent/50 hover:bg-muted'
            }`}
          >
            <div className="flex items-center gap-1.5">
              <span className="text-base">{getTemplateIcon(t)}</span>
              <span className="font-medium truncate">{t.name}</span>
            </div>
            <div className="mt-0.5 text-xs text-muted-foreground">
              {t.doc_type}
            </div>
          </button>
        ))}
      </div>
    </div>
  )
}

function getTemplateIcon(t: Template): string {
  switch (t.doc_type) {
    case 'post-mortem':
      return '\uD83D\uDD25'
    case 'runbook':
      return '\uD83D\uDCD6'
    case 'sop':
      return '\uD83D\uDCCB'
    case 'adr':
      return '\uD83C\uDFDB\uFE0F'
    default:
      return '\uD83D\uDCC4'
  }
}
