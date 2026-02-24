import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Search, Plus, Settings, ExternalLink, Sun, Moon } from 'lucide-react'
import { useState, useEffect } from 'react'
import { api } from '@/api/client'
import type { Space } from '@/api/client'
import { CreateSpaceDialog } from '@/components/CreateSpaceDialog'

interface SidebarProps {
  onOpenPalette: () => void
}

export function Sidebar({ onOpenPalette }: SidebarProps) {
  const [darkMode, setDarkMode] = useState(true)
  const [showCreateSpace, setShowCreateSpace] = useState(false)

  const { data: spaces } = useQuery({
    queryKey: ['spaces'],
    queryFn: () => api.get<Space[]>('/spaces'),
  })

  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode)
    document.documentElement.classList.toggle('light', !darkMode)
  }, [darkMode])

  return (
    <>
      <aside className="flex w-80 shrink-0 flex-col border-r border-border bg-sidebar">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <Link to="/" className="flex items-center gap-2 font-semibold text-sidebar-foreground">
            <span className="text-xl">🦉</span>
            <span>BookOwl</span>
          </Link>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setDarkMode(!darkMode)}
              className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              title="Toggle theme"
            >
              {darkMode ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            </button>
          </div>
        </div>

        {/* Search trigger */}
        <div className="px-3 py-3">
          <button
            onClick={onOpenPalette}
            className="flex w-full items-center gap-2 rounded-md border border-border bg-card px-3 py-2 text-sm text-muted-foreground transition-colors hover:border-accent/50"
          >
            <Search className="h-4 w-4" />
            <span>Search docs...</span>
            <kbd className="ml-auto rounded bg-muted px-1.5 py-0.5 text-xs">⌘K</kbd>
          </button>
        </div>

        {/* Spaces list */}
        <div className="flex-1 overflow-y-auto px-3">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Spaces
            </span>
            <button
              onClick={() => setShowCreateSpace(true)}
              className="rounded p-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              title="New space"
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
          </div>

          <nav className="space-y-0.5">
            {spaces?.map((space) => (
              <Link
                key={space.id}
                to="/spaces/$spaceId"
                params={{ spaceId: space.id }}
                className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted [&.active]:bg-muted [&.active]:text-accent"
              >
                <span>{space.icon || '📁'}</span>
                <span className="truncate">{space.name}</span>
              </Link>
            ))}

            {!spaces?.length && (
              <div className="px-2 py-4 text-center text-xs text-muted-foreground">
                No spaces yet. Create one to get started.
              </div>
            )}
          </nav>
        </div>

        {/* Footer */}
        <div className="border-t border-border px-3 py-3">
          <div className="flex items-center gap-2">
            <Link
              to="/admin"
              className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              <Settings className="h-4 w-4" />
              <span>Admin</span>
            </Link>
            <a
              href="#"
              className="flex items-center gap-1 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              <span>NightOwl</span>
              <ExternalLink className="h-3 w-3" />
            </a>
          </div>
        </div>
      </aside>

      <CreateSpaceDialog
        open={showCreateSpace}
        onClose={() => setShowCreateSpace(false)}
      />
    </>
  )
}
