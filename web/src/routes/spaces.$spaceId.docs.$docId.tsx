import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useCallback } from 'react'
import { Pencil, Eye, MoreHorizontal, ChevronLeft } from 'lucide-react'
import { api } from '@/api/client'
import { BookOwlEditor } from '@/components/editor/BookOwlEditor'
import { formatDistanceToNow } from 'date-fns'
import type { Document, Space } from '@/api/client'

export const Route = createFileRoute('/spaces/$spaceId/docs/$docId')({
  component: DocumentPage,
})

function DocumentPage() {
  const { spaceId, docId } = Route.useParams()
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState(false)

  const { data: space } = useQuery({
    queryKey: ['space', spaceId],
    queryFn: () => api.get<Space>(`/spaces/${spaceId}`),
  })

  const { data: doc, isLoading } = useQuery({
    queryKey: ['document', docId],
    queryFn: () => api.get<Document>(`/documents/${docId}`),
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
            <button className="rounded-md border border-border p-1.5 transition-colors hover:bg-muted">
              <MoreHorizontal className="h-4 w-4" />
            </button>
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
