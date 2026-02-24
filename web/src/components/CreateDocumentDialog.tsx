import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { X } from 'lucide-react'
import { api } from '@/api/client'
import type { Document } from '@/api/client'

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

export function CreateDocumentDialog({
  open,
  spaceId,
  collectionId,
  onClose,
}: CreateDocumentDialogProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [title, setTitle] = useState('')
  const [docType, setDocType] = useState('page')

  const createMutation = useMutation({
    mutationFn: () =>
      api.post<Document>('/documents', {
        space_id: spaceId,
        collection_id: collectionId,
        title,
        slug: title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, ''),
        doc_type: docType,
        content: { type: 'doc', content: [{ type: 'paragraph' }] },
      }),
    onSuccess: (doc) => {
      queryClient.invalidateQueries({ queryKey: ['space-tree', spaceId] })
      setTitle('')
      setDocType('page')
      onClose()
      navigate({
        to: '/spaces/$spaceId/docs/$docId',
        params: { spaceId, docId: doc.id },
      })
    },
  })

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/60" onClick={onClose} />
      <div className="relative w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">New Document</h2>
          <button
            onClick={onClose}
            className="rounded p-1 text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (title.trim()) createMutation.mutate()
          }}
          className="space-y-4"
        >
          <div>
            <label className="mb-1 block text-sm font-medium">Title</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Document title"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              autoFocus
            />
          </div>

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

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md border border-border px-4 py-2 text-sm hover:bg-muted"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!title.trim() || createMutation.isPending}
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
