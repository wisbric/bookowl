import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useCallback, useRef, useEffect } from 'react'
import { Pencil, Eye, Clock, ChevronLeft, MoreHorizontal, LayoutTemplate, X, MessageSquare } from 'lucide-react'
import { api } from '@/api/client'
import { BookOwlEditor } from '@/components/editor/BookOwlEditor'
import { VersionHistory } from '@/components/VersionHistory'
import { CommentPanel } from '@/components/CommentPanel'
import { formatDistanceToNow } from 'date-fns'
import type { Document, Space, Template, CommentCount } from '@/api/client'

export const Route = createFileRoute('/spaces/$spaceId/docs/$docId')({
  component: DocumentPage,
})

function DocumentPage() {
  const { spaceId, docId } = Route.useParams()
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [showVersions, setShowVersions] = useState(false)
  const [showComments, setShowComments] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const [showSaveTemplate, setShowSaveTemplate] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const { data: space } = useQuery({
    queryKey: ['space', spaceId],
    queryFn: () => api.get<Space>(`/spaces/${spaceId}`),
  })

  const { data: doc, isLoading } = useQuery({
    queryKey: ['document', docId],
    queryFn: () => api.get<Document>(`/documents/${docId}`),
  })

  const { data: commentCount } = useQuery({
    queryKey: ['comment-count', docId],
    queryFn: () => api.get<CommentCount>(`/documents/${docId}/comments/count`),
  })

  const saveMutation = useMutation({
    mutationFn: (content: Record<string, unknown>) =>
      api.put<Document>(`/documents/${docId}`, {
        title: doc!.title,
        slug: doc!.slug,
        content,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['document', docId] })
    },
  })

  const handleSave = useCallback(
    (content: Record<string, unknown>) => {
      saveMutation.mutate(content)
    },
    [saveMutation],
  )

  // Close menu on outside click.
  useEffect(() => {
    if (!showMenu) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [showMenu])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20 text-muted-foreground">
        Loading...
      </div>
    )
  }

  if (!doc) {
    return (
      <div className="flex items-center justify-center py-20 text-muted-foreground">
        Document not found.
      </div>
    )
  }

  const docTypeColor = getDocTypeColor(doc.doc_type)

  return (
    <div className="flex h-full">
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-4xl px-8 py-6">
          {/* Breadcrumb */}
          <div className="mb-4 flex items-center gap-1 text-sm text-muted-foreground">
            <Link
              to="/spaces/$spaceId"
              params={{ spaceId }}
              className="hover:text-foreground"
            >
              <ChevronLeft className="mr-1 inline h-4 w-4" />
              {space?.name || 'Space'}
            </Link>
          </div>

          {/* Document header */}
          <div className="mb-6">
            <div className="flex items-start justify-between">
              <h1 className="text-2xl font-bold tracking-tight">
                {doc.icon && <span className="mr-2">{doc.icon}</span>}
                {doc.title}
              </h1>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setEditing(!editing)}
                  className="inline-flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-sm transition-colors hover:bg-muted"
                >
                  {editing ? (
                    <><Eye className="h-4 w-4" /> View</>
                  ) : (
                    <><Pencil className="h-4 w-4" /> Edit</>
                  )}
                </button>
                <button
                  onClick={() => {
                    setShowComments(!showComments)
                    if (!showComments) setShowVersions(false)
                  }}
                  className={`inline-flex items-center gap-1 rounded-md border border-border px-2 py-1.5 text-sm transition-colors hover:bg-muted ${showComments ? 'bg-muted text-accent' : ''}`}
                  title="Comments"
                >
                  <MessageSquare className="h-4 w-4" />
                  {(commentCount?.count ?? 0) > 0 && (
                    <span className="text-xs">{commentCount?.count}</span>
                  )}
                </button>
                <button
                  onClick={() => {
                    setShowVersions(!showVersions)
                    if (!showVersions) setShowComments(false)
                  }}
                  className={`rounded-md border border-border p-1.5 transition-colors hover:bg-muted ${showVersions ? 'bg-muted text-accent' : ''}`}
                  title="Version history"
                >
                  <Clock className="h-4 w-4" />
                </button>
                {/* More menu */}
                <div className="relative" ref={menuRef}>
                  <button
                    onClick={() => setShowMenu(!showMenu)}
                    className="rounded-md border border-border p-1.5 transition-colors hover:bg-muted"
                    title="More actions"
                  >
                    <MoreHorizontal className="h-4 w-4" />
                  </button>
                  {showMenu && (
                    <div className="absolute right-0 top-full mt-1 w-48 rounded-md border border-border bg-card py-1 shadow-lg z-20">
                      <button
                        onClick={() => {
                          setShowMenu(false)
                          setShowSaveTemplate(true)
                        }}
                        className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-muted"
                      >
                        <LayoutTemplate className="h-4 w-4" />
                        Save as Template
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>
            <div className="mt-1 flex items-center gap-2 text-sm text-muted-foreground">
              <span
                className="inline-block h-2 w-2 rounded-full"
                style={{ backgroundColor: docTypeColor }}
              />
              <span className="capitalize">{doc.doc_type}</span>
              <span>·</span>
              <span className="capitalize">{doc.status}</span>
              <span>·</span>
              <span>v{doc.version}</span>
              <span>·</span>
              <span>Updated {formatDistanceToNow(new Date(doc.updated_at))} ago</span>
            </div>
            {saveMutation.isPending && (
              <div className="mt-2 text-xs text-accent">Saving...</div>
            )}
            {saveMutation.isSuccess && (
              <div className="mt-2 text-xs text-muted-foreground">Saved</div>
            )}
          </div>

          {/* Editor / Viewer */}
          <BookOwlEditor
            content={doc.content}
            editable={editing}
            onSave={handleSave}
          />
        </div>
      </div>

      {/* Comment panel */}
      {showComments && (
        <div className="w-80 shrink-0">
          <CommentPanel
            documentId={docId}
            onClose={() => setShowComments(false)}
          />
        </div>
      )}

      {/* Version history panel */}
      {showVersions && (
        <div className="w-80 shrink-0">
          <VersionHistory
            documentId={docId}
            currentVersion={doc.version}
            onClose={() => setShowVersions(false)}
          />
        </div>
      )}

      {/* Save as Template dialog */}
      {showSaveTemplate && doc && (
        <SaveAsTemplateDialog
          documentId={docId}
          documentTitle={doc.title}
          spaceId={spaceId}
          spaceName={space?.name}
          onClose={() => setShowSaveTemplate(false)}
        />
      )}
    </div>
  )
}

// --- Save as Template Dialog ---

function SaveAsTemplateDialog({
  documentId,
  documentTitle,
  spaceId,
  spaceName,
  onClose,
}: {
  documentId: string
  documentTitle: string
  spaceId: string
  spaceName?: string
  onClose: () => void
}) {
  const queryClient = useQueryClient()
  const [name, setName] = useState(documentTitle)
  const [description, setDescription] = useState('')
  const [scope, setScope] = useState<'space' | 'global'>('space')

  const saveMutation = useMutation({
    mutationFn: () =>
      api.post<Template>(`/documents/${documentId}/save-as-template`, {
        name,
        description,
        is_global: scope === 'global',
        space_id: scope === 'space' ? spaceId : undefined,
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
          <h2 className="text-lg font-semibold">Save as Template</h2>
          <button onClick={onClose} className="rounded p-1 text-muted-foreground hover:text-foreground">
            <X className="h-4 w-4" />
          </button>
        </div>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (name.trim()) saveMutation.mutate()
          }}
          className="space-y-4"
        >
          <div>
            <label className="mb-1 block text-sm font-medium">Template name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">Description (optional)</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What is this template for?"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <fieldset className="space-y-2">
            <legend className="mb-1 text-sm font-medium">Make available to</legend>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="scope"
                checked={scope === 'space'}
                onChange={() => setScope('space')}
                className="accent-accent"
              />
              This space only{spaceName ? ` (${spaceName})` : ''}
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="scope"
                checked={scope === 'global'}
                onChange={() => setScope('global')}
                className="accent-accent"
              />
              All spaces (global)
            </label>
          </fieldset>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose}
              className="rounded-md border border-border px-4 py-2 text-sm hover:bg-muted">
              Cancel
            </button>
            <button type="submit" disabled={!name.trim() || saveMutation.isPending}
              className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-foreground hover:bg-accent/90 disabled:opacity-50">
              {saveMutation.isPending ? 'Saving...' : 'Save as template'}
            </button>
          </div>
          {saveMutation.isError && (
            <p className="text-sm text-destructive">Failed to save as template.</p>
          )}
        </form>
      </div>
    </div>
  )
}

function getDocTypeColor(docType: string): string {
  switch (docType) {
    case 'runbook': return '#8B5CF6'
    case 'post-mortem': return '#DC2626'
    case 'sop': return '#3B82F6'
    case 'adr': return '#F59E0B'
    default: return '#6B7280'
  }
}
