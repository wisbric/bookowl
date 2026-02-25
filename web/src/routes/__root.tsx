import { createRootRoute, Outlet, useNavigate, useRouterState } from '@tanstack/react-router'
import { useState, useEffect, useCallback } from 'react'
import { Sidebar } from '@/components/layout/Sidebar'
import { CommandPalette } from '@/components/CommandPalette'
import { useAuth } from '@/auth/auth-provider'

const PUBLIC_ROUTES = ['/login', '/callback', '/change-password']

export const Route = createRootRoute({
  component: RootLayout,
})

function RootLayout() {
  const [paletteOpen, setPaletteOpen] = useState(false)
  const [manualCollapse, setManualCollapse] = useState<boolean | null>(null)
  const { isLoading, isAuthenticated, devMode } = useAuth()
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  const isPublicRoute = PUBLIC_ROUTES.some((r) => pathname.startsWith(r))
  const isInsideSpace = pathname.startsWith('/spaces/')

  // Reset manual override when route changes between space/non-space.
  useEffect(() => {
    setManualCollapse(null)
  }, [isInsideSpace])

  const collapsed = manualCollapse ?? isInsideSpace

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault()
      setPaletteOpen((prev) => !prev)
    }
  }, [])

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  useEffect(() => {
    if (!isLoading && !isAuthenticated && !devMode && !isPublicRoute) {
      navigate({ to: '/login', search: { return: pathname } as Record<string, string> })
    }
  }, [isLoading, isAuthenticated, devMode, isPublicRoute, navigate, pathname])

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <p className="text-muted-foreground">Loading...</p>
      </div>
    )
  }

  // Public routes render without sidebar.
  if (isPublicRoute) {
    return <Outlet />
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background text-foreground">
      <Sidebar onOpenPalette={() => setPaletteOpen(true)} collapsed={collapsed} onToggle={() => setManualCollapse(!collapsed)} />
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
    </div>
  )
}
