import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { Space } from '@/api/client'

export const Route = createFileRoute('/spaces/$spaceId/')({
  component: SpaceIndex,
})

function SpaceIndex() {
  const { spaceId } = Route.useParams()

  const { data: space } = useQuery({
    queryKey: ['space', spaceId],
    queryFn: () => api.get<Space>(`/spaces/${spaceId}`),
  })

  return (
    <div className="px-8 py-8">
      {space && (
        <>
          <h1 className="text-2xl font-bold tracking-tight">
            {space.icon && <span className="mr-2">{space.icon}</span>}
            {space.name}
          </h1>
          {space.description && (
            <p className="mt-2 text-muted-foreground">{space.description}</p>
          )}
        </>
      )}

      <div className="mt-8 text-sm text-muted-foreground">
        Select a document from the sidebar to get started.
      </div>
    </div>
  )
}
