import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Clock } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { api } from '@/api/client'
import type { Document, Space, ListResponse } from '@/api/client'

export const Route = createFileRoute('/recent')({
  component: RecentChangesPage,
})

function RecentChangesPage() {
  const { data: docsData, isLoading: docsLoading } = useQuery({
    queryKey: ['documents', 'recent'],
    queryFn: () => api.get<ListResponse<Document>>('/documents?limit=20'),
  })

  const { data: spacesData } = useQuery({
    queryKey: ['spaces'],
    queryFn: () => api.get<ListResponse<Space>>('/spaces?limit=100'),
  })

  const spacesById = new Map(
    spacesData?.items?.map((s) => [s.id, s]) ?? [],
  )

  return (
    <div className="max-w-[860px] px-8 py-8">
      <div className="mb-6 flex items-center gap-2">
        <Clock className="h-5 w-5 text-accent" />
        <h1 className="text-2xl font-bold tracking-tight">Recent Changes</h1>
      </div>

      {docsLoading && (
        <p className="text-muted-foreground">Loading...</p>
      )}

      {docsData?.items && docsData.items.length === 0 && (
        <p className="text-muted-foreground">No documents yet.</p>
      )}

      {docsData?.items && docsData.items.length > 0 && (
        <div className="space-y-2">
          {docsData.items.map((doc) => {
            const space = spacesById.get(doc.space_id)
            return (
              <Link
                key={doc.id}
                to="/spaces/$spaceId/docs/$docId"
                params={{ spaceId: doc.space_id, docId: doc.id }}
                className="flex items-center justify-between rounded-lg border border-border bg-card p-4 transition-colors hover:border-accent/50 hover:bg-card/80"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <h3 className="truncate font-medium">
                      {doc.icon && <span className="mr-1">{doc.icon}</span>}
                      {doc.title}
                    </h3>
                    <DocTypeBadge type={doc.doc_type} />
                  </div>
                  <p className="mt-0.5 text-sm text-muted-foreground">
                    {space?.name ?? 'Unknown space'}
                  </p>
                </div>
                <span className="ml-4 shrink-0 text-xs text-muted-foreground">
                  {formatDistanceToNow(new Date(doc.updated_at), { addSuffix: true })}
                </span>
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}

function DocTypeBadge({ type }: { type: string }) {
  const colors: Record<string, string> = {
    'runbook': 'bg-violet-500/20 text-violet-400',
    'post-mortem': 'bg-red-500/20 text-red-400',
    'sop': 'bg-blue-500/20 text-blue-400',
    'adr': 'bg-amber-500/20 text-amber-400',
  }

  return (
    <span
      className={`rounded-full px-2 py-0.5 text-xs font-medium ${colors[type] ?? 'bg-muted text-muted-foreground'}`}
    >
      {type}
    </span>
  )
}
