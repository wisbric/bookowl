import { DragOverlay as DndDragOverlay } from '@dnd-kit/core'
import { FileText, FolderOpen } from 'lucide-react'
import type { DragItem } from './types'

interface Props {
  activeItem: DragItem | null
}

export function SidebarDragOverlay({ activeItem }: Props) {
  if (!activeItem) return null

  return (
    <DndDragOverlay dropAnimation={null}>
      <div className="flex max-w-[280px] items-center gap-1.5 rounded bg-card/90 px-2 py-1.5 text-sm shadow-lg ring-1 ring-border">
        {activeItem.kind === 'document' ? (
          <>
            <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            <span className="truncate">{activeItem.title}</span>
          </>
        ) : (
          <>
            <FolderOpen className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            <span className="truncate">{activeItem.name}</span>
          </>
        )}
      </div>
    </DndDragOverlay>
  )
}
