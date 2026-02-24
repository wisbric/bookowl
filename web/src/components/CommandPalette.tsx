import React, { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Search, FileText, X } from 'lucide-react'
import { api } from '@/api/client'
import type { SearchResult } from '@/api/client'

interface CommandPaletteProps {
  open: boolean
  onClose: () => void
}

export function CommandPalette({ open, onClose }: CommandPaletteProps) {
  const navigate = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null)
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [debouncedQuery, setDebouncedQuery] = useState('')

  // Debounce the query by 300ms.
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 300)
    return () => clearTimeout(timer)
  }, [query])

  const { data, isLoading } = useQuery({
    queryKey: ['palette-search', debouncedQuery],
    queryFn: () =>
      api.get<SearchResult[]>(
        `/search?q=${encodeURIComponent(debouncedQuery)}&limit=8`,
      ),
    enabled: debouncedQuery.length > 1,
  })

  const results = data || []

  // Focus input when opened.
  useEffect(() => {
    if (open) {
      setQuery('')
      setDebouncedQuery('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

  // Reset selection when results change.
  useEffect(() => {
    setSelectedIndex(0)
  }, [results.length])

  const navigateToResult = useCallback(
    (result: SearchResult) => {
      onClose()
      navigate({
        to: '/spaces/$spaceId/docs/$docId',
        params: { spaceId: result.space_id, docId: result.id },
      })
    },
    [navigate, onClose],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setSelectedIndex((i) => Math.min(i + 1, results.length - 1))
          break
        case 'ArrowUp':
          e.preventDefault()
          setSelectedIndex((i) => Math.max(i - 1, 0))
          break
        case 'Enter':
          e.preventDefault()
          if (results[selectedIndex]) {
            navigateToResult(results[selectedIndex])
          } else if (query) {
            onClose()
            navigate({ to: '/search', search: { q: query } })
          }
          break
        case 'Escape':
          onClose()
          break
      }
    },
    [results, selectedIndex, navigateToResult, query, navigate, onClose],
  )

  if (!open) return null

  const docTypeColor = (dt: string) => {
    switch (dt) {
      case 'runbook': return '#8B5CF6'
      case 'post-mortem': return '#DC2626'
      case 'sop': return '#3B82F6'
      case 'adr': return '#F59E0B'
      default: return '#6B7280'
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/60" onClick={onClose} />

      {/* Modal */}
      <div className="relative w-full max-w-xl rounded-xl border border-border bg-card shadow-2xl">
        {/* Search input */}
        <div className="flex items-center border-b border-border px-4">
          <Search className="h-5 w-5 shrink-0 text-muted-foreground" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search documents..."
            className="flex-1 bg-transparent px-3 py-4 text-foreground placeholder:text-muted-foreground focus:outline-none"
          />
          <button
            onClick={onClose}
            className="rounded p-1 text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Results */}
        <div className="max-h-80 overflow-y-auto p-2">
          {isLoading && debouncedQuery && (
            <div className="px-3 py-6 text-center text-sm text-muted-foreground">
              Searching...
            </div>
          )}

          {!isLoading && results.length === 0 && debouncedQuery.length > 1 && (
            <div className="px-3 py-6 text-center text-sm text-muted-foreground">
              No results for &ldquo;{debouncedQuery}&rdquo;
            </div>
          )}

          {!debouncedQuery && (
            <div className="px-3 py-6 text-center text-sm text-muted-foreground">
              Start typing to search across all documents
            </div>
          )}

          {results.map((result, index) => (
            <button
              key={result.id}
              onClick={() => navigateToResult(result)}
              onMouseEnter={() => setSelectedIndex(index)}
              className={`flex w-full items-start gap-3 rounded-lg px-3 py-2.5 text-left transition-colors ${
                index === selectedIndex ? 'bg-muted' : 'hover:bg-muted/50'
              }`}
            >
              <FileText
                className="mt-0.5 h-4 w-4 shrink-0"
                style={{ color: docTypeColor(result.doc_type) }}
              />
              <div className="min-w-0 flex-1">
                <div
                  className="truncate text-sm font-medium text-foreground"
                  dangerouslySetInnerHTML={{
                    __html: result.title_highlight || result.title,
                  }}
                />
                <div className="mt-0.5 truncate text-xs text-muted-foreground">
                  {result.space_name} · {result.doc_type}
                </div>
              </div>
            </button>
          ))}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-border px-4 py-2 text-xs text-muted-foreground">
          <div className="flex items-center gap-3">
            <span><kbd className="rounded bg-muted px-1 py-0.5">↑↓</kbd> Navigate</span>
            <span><kbd className="rounded bg-muted px-1 py-0.5">↵</kbd> Open</span>
            <span><kbd className="rounded bg-muted px-1 py-0.5">Esc</kbd> Close</span>
          </div>
        </div>
      </div>
    </div>
  )
}
