import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo } from 'react'
import { Plus, Pencil, Trash2, X } from 'lucide-react'
import { api } from '@/api/client'
import type { Template } from '@/api/client'

export const Route = createFileRoute('/templates')({
  component: TemplatesPage,
})

function TemplatesPage() {
  const queryClient = useQueryClient()
  const [showCreate, setShowCreate] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<Template | null>(null)

  const { data: templates, isLoading } = useQuery({
    queryKey: ['templates'],
    queryFn: () => api.get<Template[]>('/templates'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.del(`/templates/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] })
    },
  })

  const grouped = useMemo(() => {
    if (!templates) return { system: [], global: [], spaces: new Map<string, Template[]>() }
    const system: Template[] = []
    const global: Template[] = []
    const spaces = new Map<string, Template[]>()
    for (const t of templates) {
      if (t.is_system) {
        system.push(t)
      } else if (t.is_global) {
        global.push(t)
      } else {
        const key = t.space_id || 'unassigned'
        const list = spaces.get(key) || []
        list.push(t)
        spaces.set(key, list)
      }
    }
    return { system, global, spaces }
  }, [templates])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20 text-muted-foreground">
        Loading templates...
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl px-8 py-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-tight">Templates</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-sm font-medium text-accent-foreground hover:bg-accent/90"
        >
          <Plus className="h-4 w-4" />
          New template
        </button>
      </div>

      {/* System templates */}
      {grouped.system.length > 0 && (
        <TemplateSection title={`System (${grouped.system.length})`}>
          {grouped.system.map((t) => (
            <TemplateRow key={t.id} template={t} />
          ))}
        </TemplateSection>
      )}

      {/* Global templates */}
      {grouped.global.length > 0 && (
        <TemplateSection title={`Global (${grouped.global.length})`}>
          {grouped.global.map((t) => (
            <TemplateRow
              key={t.id}
              template={t}
              onEdit={() => setEditingTemplate(t)}
              onDelete={() => {
                if (confirm(`Delete "${t.name}"?`)) deleteMutation.mutate(t.id)
              }}
            />
          ))}
        </TemplateSection>
      )}

      {/* Space templates */}
      {Array.from(grouped.spaces.entries()).map(([spaceId, list]) => (
        <TemplateSection key={spaceId} title={`Space (${list.length})`}>
          {list.map((t) => (
            <TemplateRow
              key={t.id}
              template={t}
              onEdit={() => setEditingTemplate(t)}
              onDelete={() => {
                if (confirm(`Delete "${t.name}"?`)) deleteMutation.mutate(t.id)
              }}
            />
          ))}
        </TemplateSection>
      ))}

      {!templates?.length && (
        <div className="py-12 text-center text-muted-foreground">
          No templates yet. Create one to get started.
        </div>
      )}

      {showCreate && (
        <CreateTemplateDialog
          onClose={() => setShowCreate(false)}
        />
      )}

      {editingTemplate && (
        <EditTemplateDialog
          template={editingTemplate}
          onClose={() => setEditingTemplate(null)}
        />
      )}
    </div>
  )
}

function TemplateSection({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <div className="mb-6">
      <div className="mb-2 border-b border-border pb-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
        {title}
      </div>
      <div className="divide-y divide-border">{children}</div>
    </div>
  )
}

function TemplateRow({
  template,
  onEdit,
  onDelete,
}: {
  template: Template
  onEdit?: () => void
  onDelete?: () => void
}) {
  return (
    <div className="flex items-center justify-between py-2.5">
      <div className="flex items-center gap-3">
        <span className="text-lg">{getDocTypeEmoji(template.doc_type)}</span>
        <div>
          <div className="text-sm font-medium">{template.name}</div>
          <div className="text-xs text-muted-foreground">
            {template.doc_type}
            {template.description && ` \u00B7 ${template.description}`}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-1">
        {onEdit && (
          <button
            onClick={onEdit}
            className="rounded p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
            title="Edit"
          >
            <Pencil className="h-3.5 w-3.5" />
          </button>
        )}
        {onDelete && (
          <button
            onClick={onDelete}
            className="rounded p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
            title="Delete"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        )}
      </div>
    </div>
  )
}

function getDocTypeEmoji(docType: string): string {
  switch (docType) {
    case 'post-mortem': return '\uD83D\uDD25'
    case 'runbook': return '\uD83D\uDCD6'
    case 'sop': return '\uD83D\uDCCB'
    case 'adr': return '\uD83C\uDFDB\uFE0F'
    default: return '\uD83D\uDCC4'
  }
}

// --- Create Template Dialog ---

function CreateTemplateDialog({ onClose }: { onClose: () => void }) {
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [docType, setDocType] = useState('document')
  const [isGlobal, setIsGlobal] = useState(true)

  const createMutation = useMutation({
    mutationFn: () =>
      api.post<Template>('/templates', {
        name,
        description,
        doc_type: docType,
        is_global: isGlobal,
        content: { type: 'doc', content: [{ type: 'heading', attrs: { level: 1 }, content: [{ type: 'text', text: name }] }, { type: 'paragraph' }] },
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      onClose()
    },
  })

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/60" onClick={onClose} />
      <div className="relative w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">New Template</h2>
          <button onClick={onClose} className="rounded p-1 text-muted-foreground hover:text-foreground">
            <X className="h-4 w-4" />
          </button>
        </div>
        <form
          onSubmit={(e) => { e.preventDefault(); if (name.trim()) createMutation.mutate() }}
          className="space-y-4"
        >
          <div>
            <label className="mb-1 block text-sm font-medium">Name</label>
            <input type="text" value={name} onChange={(e) => setName(e.target.value)}
              placeholder="Template name" autoFocus
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent" />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">Description</label>
            <input type="text" value={description} onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent" />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">Document type</label>
            <select value={docType} onChange={(e) => setDocType(e.target.value)}
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent">
              <option value="document">Document</option>
              <option value="runbook">Runbook</option>
              <option value="post-mortem">Post-Mortem</option>
              <option value="sop">SOP</option>
              <option value="adr">ADR</option>
            </select>
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={isGlobal} onChange={(e) => setIsGlobal(e.target.checked)}
              className="rounded border-border" />
            Global (visible to all spaces)
          </label>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose}
              className="rounded-md border border-border px-4 py-2 text-sm hover:bg-muted">Cancel</button>
            <button type="submit" disabled={!name.trim() || createMutation.isPending}
              className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-foreground hover:bg-accent/90 disabled:opacity-50">
              {createMutation.isPending ? 'Creating...' : 'Create'}
            </button>
          </div>
          {createMutation.isError && (
            <p className="text-sm text-destructive">Failed to create template.</p>
          )}
        </form>
      </div>
    </div>
  )
}

// --- Edit Template Dialog ---

function EditTemplateDialog({ template, onClose }: { template: Template; onClose: () => void }) {
  const queryClient = useQueryClient()
  const [name, setName] = useState(template.name)
  const [description, setDescription] = useState(template.description || '')

  const updateMutation = useMutation({
    mutationFn: () =>
      api.put<Template>(`/templates/${template.id}`, {
        name,
        description,
        content: template.content,
        sort_order: template.sort_order,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      onClose()
    },
  })

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/60" onClick={onClose} />
      <div className="relative w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">Edit Template</h2>
          <button onClick={onClose} className="rounded p-1 text-muted-foreground hover:text-foreground">
            <X className="h-4 w-4" />
          </button>
        </div>
        <form
          onSubmit={(e) => { e.preventDefault(); if (name.trim()) updateMutation.mutate() }}
          className="space-y-4"
        >
          <div>
            <label className="mb-1 block text-sm font-medium">Name</label>
            <input type="text" value={name} onChange={(e) => setName(e.target.value)}
              autoFocus
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent" />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">Description</label>
            <input type="text" value={description} onChange={(e) => setDescription(e.target.value)}
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent" />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose}
              className="rounded-md border border-border px-4 py-2 text-sm hover:bg-muted">Cancel</button>
            <button type="submit" disabled={!name.trim() || updateMutation.isPending}
              className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-foreground hover:bg-accent/90 disabled:opacity-50">
              {updateMutation.isPending ? 'Saving...' : 'Save'}
            </button>
          </div>
          {updateMutation.isError && (
            <p className="text-sm text-destructive">Failed to update template.</p>
          )}
        </form>
      </div>
    </div>
  )
}
