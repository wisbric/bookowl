import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Search, Plus, Settings, Info, ExternalLink, Sun, Moon, LogOut, User, Key, LayoutTemplate, PanelLeftClose, PanelLeftOpen } from 'lucide-react'
import { useState, useEffect, useRef } from 'react'
import { api } from '@/api/client'
import type { Space } from '@/api/client'
import { CreateSpaceDialog } from '@/components/CreateSpaceDialog'
import { UserAvatar } from '@/components/UserAvatar'
import { NotificationBell } from '@/components/NotificationBell'
import { useAuth } from '@/auth/auth-provider'

interface SidebarProps {
  onOpenPalette: () => void
  collapsed?: boolean
  onToggle?: () => void
}

export function Sidebar({ onOpenPalette, collapsed, onToggle }: SidebarProps) {
  const [darkMode, setDarkMode] = useState(true)
  const [showCreateSpace, setShowCreateSpace] = useState(false)
  const [showUserMenu, setShowUserMenu] = useState(false)
  const userMenuRef = useRef<HTMLDivElement>(null)
  const { profile, logout } = useAuth()

  const { data: spaces } = useQuery({
    queryKey: ['spaces'],
    queryFn: () => api.get<Space[]>('/spaces'),
  })

  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode)
    document.documentElement.classList.toggle('light', !darkMode)
  }, [darkMode])

  // Close user menu on outside click.
  useEffect(() => {
    if (!showUserMenu) return
    const handler = (e: MouseEvent) => {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setShowUserMenu(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [showUserMenu])

  // --- Collapsed icon strip ---
  if (collapsed) {
    return (
      <>
        <aside className="flex w-14 shrink-0 flex-col items-center border-r border-border bg-sidebar py-3">
          <Link to="/" className="mb-2" title="Home">
            <img src="/owl.png" alt="BookOwl" className="h-7 brightness-0 dark:brightness-100" />
          </Link>

          <button
            onClick={onOpenPalette}
            className="mb-2 rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            title="Search (⌘K)"
          >
            <Search className="h-4 w-4" />
          </button>

          <Link
            to="/templates"
            className="mb-2 rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground [&.active]:text-accent"
            title="Templates"
          >
            <LayoutTemplate className="h-4 w-4" />
          </Link>

          <div className="my-2 h-px w-6 bg-border" />

          <nav className="flex flex-1 flex-col items-center gap-1 overflow-y-auto">
            {spaces?.map((space) => (
              <Link
                key={space.id}
                to="/spaces/$spaceId"
                params={{ spaceId: space.id }}
                className="rounded-md p-2 text-sm transition-colors hover:bg-muted [&.active]:bg-muted [&.active]:text-accent"
                title={space.name}
              >
                <span>{space.icon || '📁'}</span>
              </Link>
            ))}
          </nav>

          <div className="mt-auto flex flex-col items-center gap-1 pt-2">
            <NotificationBell collapsed />
            {onToggle && (
              <button
                onClick={onToggle}
                className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                title="Expand sidebar"
              >
                <PanelLeftOpen className="h-4 w-4" />
              </button>
            )}
            <button
              onClick={() => setDarkMode(!darkMode)}
              className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              title={darkMode ? 'Light mode' : 'Dark mode'}
            >
              {darkMode ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            </button>
            {profile && (
              <div className="relative" ref={userMenuRef}>
                <button
                  onClick={() => setShowUserMenu(!showUserMenu)}
                  className="rounded-md p-1 transition-colors hover:bg-muted"
                  title={profile.display_name || profile.email}
                >
                  <UserAvatar profile={profile} size="sm" />
                </button>
                {showUserMenu && (
                  <div className="absolute bottom-full left-full z-50 mb-1 ml-1 w-48 rounded-lg border border-border bg-card shadow-lg">
                    <div className="border-b border-border px-3 py-2.5">
                      <div className="truncate text-sm font-medium text-foreground">
                        {profile.display_name || 'User'}
                      </div>
                      <div className="truncate text-xs text-muted-foreground">{profile.email}</div>
                    </div>
                    <div className="py-1">
                      <Link to="/profile" onClick={() => setShowUserMenu(false)}
                        className="flex items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted">
                        <User className="h-4 w-4" /> Profile
                      </Link>
                      <Link to="/profile/tokens" onClick={() => setShowUserMenu(false)}
                        className="flex items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted">
                        <Key className="h-4 w-4" /> Personal tokens
                      </Link>
                    </div>
                    <div className="border-t border-border py-1">
                      <button onClick={() => { setShowUserMenu(false); logout() }}
                        className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-red-400 transition-colors hover:bg-muted">
                        <LogOut className="h-4 w-4" /> Sign out
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}
          {profile?.role === 'admin' && (
            <Link
              to="/admin"
              className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground [&.active]:bg-muted [&.active]:text-accent"
              title="Admin"
            >
              <Settings className="h-4 w-4" />
            </Link>
          )}
          <Link
            to="/about"
            className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground [&.active]:bg-muted [&.active]:text-accent"
            title="About"
          >
            <Info className="h-4 w-4" />
          </Link>
          </div>
        </aside>

        <CreateSpaceDialog open={showCreateSpace} onClose={() => setShowCreateSpace(false)} />
      </>
    )
  }

  // --- Full sidebar ---
  return (
    <>
      <aside className="flex w-80 shrink-0 flex-col border-r border-border bg-sidebar">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <Link to="/" className="flex items-center gap-2 font-semibold text-sidebar-foreground">
            <img src="/owl.png" alt="BookOwl" className="h-8 brightness-0 dark:brightness-100" />
            <span>BookOwl</span>
          </Link>
          <NotificationBell />
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

        {/* Quick links */}
        <div className="px-3 pb-2">
          <Link
            to="/templates"
            className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted [&.active]:bg-muted [&.active]:text-accent"
          >
            <LayoutTemplate className="h-4 w-4" />
            Templates
          </Link>
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

        {/* Collapse toggle */}
        <div className="px-3 pb-1 space-y-0.5">
          {onToggle && (
            <button
              onClick={onToggle}
              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              title="Collapse sidebar"
            >
              <PanelLeftClose className="h-4 w-4" />
              <span>Collapse</span>
            </button>
          )}
        </div>

        {/* Footer with user menu */}
        <div className="border-t border-border px-3 py-3">
          {profile ? (
            <div className="relative" ref={userMenuRef}>
              <button
                onClick={() => setShowUserMenu(!showUserMenu)}
                className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-muted"
              >
                <UserAvatar profile={profile} size="sm" />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium text-sidebar-foreground">
                    {profile.display_name || profile.email}
                  </div>
                  <div className="truncate text-xs text-muted-foreground">
                    {profile.role}
                  </div>
                </div>
              </button>

              {/* User dropdown menu */}
              {showUserMenu && (
                <div className="absolute bottom-full left-0 right-0 z-50 mb-1 rounded-lg border border-border bg-card shadow-lg">
                  <div className="border-b border-border px-3 py-2.5">
                    <div className="truncate text-sm font-medium text-foreground">
                      {profile.display_name || 'User'}
                    </div>
                    <div className="truncate text-xs text-muted-foreground">{profile.email}</div>
                  </div>

                  <div className="py-1">
                    <Link
                      to="/profile"
                      onClick={() => setShowUserMenu(false)}
                      className="flex items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                    >
                      <User className="h-4 w-4" />
                      Profile
                    </Link>
                    <Link
                      to="/profile/tokens"
                      onClick={() => setShowUserMenu(false)}
                      className="flex items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                    >
                      <Key className="h-4 w-4" />
                      Personal tokens
                    </Link>
                    <button
                      onClick={() => {
                        setDarkMode(!darkMode)
                      }}
                      className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                    >
                      {darkMode ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
                      {darkMode ? 'Light mode' : 'Dark mode'}
                    </button>
                  </div>

                  <div className="border-t border-border py-1">
                    <a
                      href="#"
                      className="flex items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                    >
                      <ExternalLink className="h-4 w-4" />
                      NightOwl
                    </a>
                  </div>

                  <div className="border-t border-border py-1">
                    <button
                      onClick={() => {
                        setShowUserMenu(false)
                        logout()
                      }}
                      className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-red-400 transition-colors hover:bg-muted"
                    >
                      <LogOut className="h-4 w-4" />
                      Sign out
                    </button>
                  </div>
                </div>
              )}
            </div>
          ) : null}
        </div>

        <div className="px-3 pb-2 space-y-0.5">
          {profile?.role === 'admin' && (
            <Link
              to="/admin"
              className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground [&.active]:bg-muted [&.active]:text-accent"
            >
              <Settings className="h-4 w-4" />
              Admin
            </Link>
          )}
          <Link
            to="/about"
            className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground [&.active]:bg-muted [&.active]:text-accent"
          >
            <Info className="h-4 w-4" />
            About
          </Link>
        </div>
      </aside>

      <CreateSpaceDialog
        open={showCreateSpace}
        onClose={() => setShowCreateSpace(false)}
      />
    </>
  )
}
