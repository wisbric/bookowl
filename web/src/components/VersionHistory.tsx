import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { X, RotateCcw, Eye } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { api } from '@/api/client'
import type { DocumentVersion, Document } from '@/api/client'
import { BookOwlEditor } from './editor/BookOwlEditor'

interface VersionHistoryProps {
  documentId: string
  currentVersion: number
  onClose: () => void
}

export function VersionHistory({ documentId, currentVersion, onClose }: VersionHistoryProps) {
  const queryClient = useQueryClient()
  const [previewVersion, setPreviewVersion] = useState<DocumentVersion | null>(null)

  const { data: versions, isLoading } = useQuery({
    queryKey: ['document-versions', documentId],
    queryFn: () =>
      api.get<{ items: DocumentVersion[] }>(`/documents/${documentId}/versions`),
  })

  const restoreMutation = useMutation({
    mutationFn: (versionId: string) =>
      api.post<Document>(`/documents/${documentId}/versions/${versionId}/restore`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['document', documentId] })
      queryClient.invalidateQueries({ queryKey: ['document-versions', documentId] })
      onClose()
    },
  })

  return (
    <div className="flex h-full flex-col border-l border-border bg-sidebar">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3 className="text-sm font-semibold text-sidebar-foreground">Version History</h3>
        <button
          onClick={onClose}
          className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Version list or preview */}
      {previewVersion ? (
        <div className="flex flex-1 flex-col overflow-hidden">
          <div className="flex items-center justify-between border-b border-border px-4 py-2">
            <span className="text-sm text-muted-foreground">
              v{previewVersion.version} preview
            </span>
            <div className="flex items-center gap-2">
              <button
                onClick={() =>
                  restoreMutation.mutate(previewVersion.id)
                }
                disabled={restoreMutation.isPending || previewVersion.version === currentVersion}
                className="inline-flex items-center gap-1 rounded-md bg-accent px-2 py-1 text-xs font-medium text-accent-foreground disabled:opacity-40"
              >
                <RotateCcw className="h-3 w-3" />
                Restore
              </button>
              <button
                onClick={() => setPreviewVersion(null)}
                className="rounded p-1 text-muted-foreground hover:text-foreground"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto p-4">
            <BookOwlEditor
              content={previewVersion.content}
              editable={false}
            />
          </div>
        </div>
      ) : (
        <div className="flex-1 overflow-y-auto">
          {isLoading && (
            <div className="px-4 py-6 text-center text-sm text-muted-foreground">
              Loading versions...
            </div>
          )}

          {versions?.items?.map((version) => (
            <div
              key={version.id}
              className="border-b border-border px-4 py-3 transition-colors hover:bg-muted/50"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">
                  v{version.version}
                  {version.version === currentVersion && (
                    <span className="ml-2 text-xs text-accent">current</span>
                  )}
                </span>
                <button
                  onClick={() => setPreviewVersion(version)}
                  className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
                  title="Preview version"
                >
                  <Eye className="h-3.5 w-3.5" />
                </button>
              </div>
              {version.change_summary && (
                <p className="mt-0.5 text-xs text-muted-foreground">
                  {version.change_summary}
                </p>
              )}
              <div className="mt-1 text-xs text-muted-foreground">
                {version.created_by && <>{version.created_by} · </>}
                {formatDistanceToNow(new Date(version.created_at))} ago
              </div>
            </div>
          ))}

          {versions?.items?.length === 0 && (
            <div className="px-4 py-6 text-center text-sm text-muted-foreground">
              No version history yet.
            </div>
          )}
        </div>
      )}
    </div>
  )
}
