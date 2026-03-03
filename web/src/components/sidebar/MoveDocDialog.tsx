import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { X, FolderOpen, ChevronRight } from 'lucide-react'
import { api } from '@/api/client'
import type { Space, SpaceTree, Document } from '@/api/client'

interface MoveDocDialogProps {
  open: boolean
  docId: string
  currentSpaceId: string
  onClose: () => void
}

export function MoveDocDialog({ open, docId, currentSpaceId, onClose }: MoveDocDialogProps) {
  const queryClient = useQueryClient()
  const [selectedSpaceId, setSelectedSpaceId] = useState<string | null>(null)
  const [selectedCollectionId, setSelectedCollectionId] = useState<string | null>(null)

  const { data: spaces } = useQuery({
    queryKey: ['spaces'],
    queryFn: () => api.get<Space[]>('/spaces'),
    enabled: open,
  })

  const { data: tree } = useQuery({
    queryKey: ['space-tree', selectedSpaceId],
    queryFn: () => api.get<SpaceTree>(`/spaces/${selectedSpaceId}/tree`),
    enabled: !!selectedSpaceId,
  })

  const moveMutation = useMutation({
    mutationFn: () =>
      api.patch<Document>(`/documents/${docId}/move`, {
        space_id: selectedSpaceId,
        collection_id: selectedCollectionId,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['space-tree'] })
      queryClient.invalidateQueries({ queryKey: ['documents'] })
      handleClose()
    },
  })

  const handleClose = () => {
    setSelectedSpaceId(null)
    setSelectedCollectionId(null)
    onClose()
  }

  if (!open) return null

  const otherSpaces = (spaces ?? []).filter((s) => s.id !== currentSpaceId)
  const canMove = selectedSpaceId && !moveMutation.isPending

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/60" onClick={handleClose} />
      <div className="relative w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">Move to Space</h2>
          <button
            onClick={handleClose}
            className="rounded p-1 text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Select a destination space and optionally a collection.
          </p>

          {/* Space list */}
          <div className="max-h-64 space-y-1 overflow-y-auto rounded-md border border-border p-1">
            {otherSpaces.length === 0 ? (
              <p className="py-4 text-center text-sm text-muted-foreground">
                No other spaces available.
              </p>
            ) : (
              otherSpaces.map((space) => (
                <div key={space.id}>
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedSpaceId(space.id)
                      setSelectedCollectionId(null)
                    }}
                    className={`flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors ${
                      selectedSpaceId === space.id
                        ? 'bg-accent/10 text-accent ring-1 ring-accent/40'
                        : 'hover:bg-muted'
                    }`}
                  >
                    <ChevronRight
                      className={`h-3.5 w-3.5 shrink-0 transition-transform ${
                        selectedSpaceId === space.id ? 'rotate-90' : ''
                      }`}
                    />
                    <span className="font-medium">{space.name}</span>
                  </button>

                  {/* Collections for selected space */}
                  {selectedSpaceId === space.id && tree?.collections && tree.collections.length > 0 && (
                    <div className="ml-6 mt-1 space-y-0.5">
                      <button
                        type="button"
                        onClick={() => setSelectedCollectionId(null)}
                        className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-xs transition-colors ${
                          selectedCollectionId === null
                            ? 'bg-accent/10 text-accent'
                            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                        }`}
                      >
                        (No collection)
                      </button>
                      {tree.collections.map((col) => (
                        <button
                          key={col.collection_id}
                          type="button"
                          onClick={() => setSelectedCollectionId(col.collection_id)}
                          className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-xs transition-colors ${
                            selectedCollectionId === col.collection_id
                              ? 'bg-accent/10 text-accent'
                              : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                          }`}
                        >
                          <FolderOpen className="h-3.5 w-3.5 shrink-0" />
                          {col.collection_name}
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              ))
            )}
          </div>

          {moveMutation.isError && (
            <p className="text-sm text-destructive">
              Failed to move document. Please try again.
            </p>
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
              type="button"
              disabled={!canMove}
              onClick={() => moveMutation.mutate()}
              className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-foreground hover:bg-accent/90 disabled:opacity-50"
            >
              {moveMutation.isPending ? 'Moving...' : 'Move'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
