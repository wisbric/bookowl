import { createFileRoute, Outlet } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { SidebarTree } from '@/components/layout/SidebarTree'
import type { Space, TreeNode } from '@/api/client'

export const Route = createFileRoute('/spaces/$spaceId')({
  component: SpaceLayout,
})

function SpaceLayout() {
  const { spaceId } = Route.useParams()

  const { data: space } = useQuery({
    queryKey: ['space', spaceId],
    queryFn: () => api.get<Space>(`/spaces/${spaceId}`),
  })

  const { data: tree } = useQuery({
    queryKey: ['space-tree', spaceId],
    queryFn: () => api.get<TreeNode[]>(`/spaces/${spaceId}/tree`),
  })

  return (
    <div className="flex h-full">
      <aside className="hidden w-64 shrink-0 overflow-y-auto border-r border-border bg-sidebar p-4 lg:block">
        {space && (
          <div className="mb-4">
            <h2 className="truncate font-semibold text-sidebar-foreground">
              {space.icon && <span className="mr-1">{space.icon}</span>}
              {space.name}
            </h2>
            {space.description && (
              <p className="mt-1 truncate text-xs text-muted-foreground">
                {space.description}
              </p>
            )}
          </div>
        )}
        {tree && <SidebarTree spaceId={spaceId} nodes={tree} />}
      </aside>
      <div className="flex-1 overflow-y-auto">
        <Outlet />
      </div>
    </div>
  )
}
