import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Pencil } from 'lucide-react'
import { api } from '@/api/client'
import type { Space } from '@/api/client'
import { CreateSpaceDialog } from '@/components/CreateSpaceDialog'

export const Route = createFileRoute('/spaces/$spaceId/')({
  component: SpaceIndex,
})

function SpaceIndex() {
  const { spaceId } = Route.useParams()
  const [editOpen, setEditOpen] = useState(false)

  const { data: space } = useQuery({
    queryKey: ['space', spaceId],
    queryFn: () => api.get<Space>(`/spaces/${spaceId}`),
  })

  return (
    <div className="px-8 py-8">
      {space && (
        <>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold tracking-tight">
              {space.icon && <span className="mr-2">{space.icon}</span>}
              {space.name}
            </h1>
            <button
              onClick={() => setEditOpen(true)}
              className="rounded p-1 text-muted-foreground hover:text-foreground"
              title="Edit space"
            >
              <Pencil className="h-4 w-4" />
            </button>
          </div>
          {space.description && (
            <p className="mt-2 text-muted-foreground">{space.description}</p>
          )}
          <CreateSpaceDialog
            open={editOpen}
            onClose={() => setEditOpen(false)}
            space={space}
          />
        </>
      )}

      <div className="mt-8 text-sm text-muted-foreground">
        Select a document from the sidebar to get started.
      </div>
    </div>
  )
}
