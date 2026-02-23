import { createRootRoute, Outlet } from '@tanstack/react-router'
import { Sidebar } from '@/components/layout/Sidebar'

export const Route = createRootRoute({
  component: RootLayout,
})

function RootLayout() {
  return (
    <div className="flex h-screen overflow-hidden bg-background text-foreground">
      <Sidebar />
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  )
}
