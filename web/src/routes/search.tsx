import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Search as SearchIcon } from 'lucide-react'
import React, { useState } from 'react'
import { api } from '@/api/client'
import type { SearchResult } from '@/api/client'

export const Route = createFileRoute('/search')({
  component: SearchPage,
  validateSearch: (search: Record<string, unknown>) => ({
    q: (search.q as string) || '',
  }),
})

function SearchPage() {
  const { q } = Route.useSearch()
  const navigate = useNavigate()
  const [query, setQuery] = useState(q)

  const { data, isLoading } = useQuery({
    queryKey: ['search', q],
    queryFn: () => api.get<{ items: SearchResult[] }>(`/search?q=${encodeURIComponent(q)}`),
    enabled: q.length > 0,
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    navigate({ to: '/search', search: { q: query } })
  }

  return (
    <div className="mx-auto max-w-3xl px-8 py-8">
      <form onSubmit={handleSubmit} className="mb-8">
        <div className="relative">
          <SearchIcon className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search documents..."
            className="w-full rounded-lg border border-border bg-card py-3 pl-10 pr-4 text-foreground placeholder:text-muted-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>
      </form>

      {isLoading && (
        <p className="text-muted-foreground">Searching...</p>
      )}

      {data?.items && data.items.length > 0 && (
        <div className="space-y-4">
          {data.items.map((result) => (
            <SearchResultCard key={result.id} result={result} />
          ))}
        </div>
      )}

      {data?.items && data.items.length === 0 && q && (
        <p className="text-muted-foreground">No results found for "{q}".</p>
      )}
    </div>
  )
}

function SearchResultCard({ result }: { result: SearchResult }) {
  const docTypeColor = getDocTypeColor(result.doc_type)

  return (
    <div
      className="rounded-lg border border-border bg-card p-4"
      style={{ borderLeftColor: docTypeColor, borderLeftWidth: '3px' }}
    >
      <div className="mb-1 text-xs text-muted-foreground">
        {result.space_name} · {result.doc_type}
      </div>
      <h3
        className="font-medium"
        dangerouslySetInnerHTML={{ __html: result.title_highlight || result.title }}
      />
      {result.content_highlight && (
        <p
          className="mt-1 text-sm text-muted-foreground"
          dangerouslySetInnerHTML={{ __html: result.content_highlight }}
        />
      )}
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
