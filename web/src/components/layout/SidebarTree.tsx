import { Link } from '@tanstack/react-router'
import { DndContext, PointerSensor, KeyboardSensor, useSensor, useSensors } from '@dnd-kit/core'
import { useDraggable, useDroppable } from '@dnd-kit/core'
import { ChevronRight, FileText, FolderOpen, GripVertical, Plus } from 'lucide-react'
import { useState } from 'react'
import type { SpaceTree, TreeCollection, TreeDoc } from '@/api/client'
import { CreateDocumentDialog } from '@/components/CreateDocumentDialog'
import { SidebarDragOverlay } from '@/components/sidebar/dnd/DragOverlay'
import { useSidebarDnd } from '@/components/sidebar/dnd/useSidebarDnd'
import type { DragItem, DropTarget } from '@/components/sidebar/dnd/types'

interface SidebarTreeProps {
  spaceId: string
  tree: SpaceTree
}

export function SidebarTree({ spaceId, tree }: SidebarTreeProps) {
  const [showCreateDoc, setShowCreateDoc] = useState(false)
  const dnd = useSidebarDnd(spaceId)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor),
  )

  return (
    <>
      <DndContext
        sensors={sensors}
        onDragStart={dnd.handleDragStart}
        onDragOver={dnd.handleDragOver}
        onDragEnd={dnd.handleDragEnd}
        onDragCancel={dnd.handleDragCancel}
      >
        <nav className="space-y-0.5">
          {/* Uncollected documents (top-level, no collection) */}
          {tree.documents?.map((doc) => (
            <DraggableDocument
              key={doc.id}
              spaceId={spaceId}
              doc={doc}
              collectionId={null}
              isDragging={dnd.activeItem?.kind === 'document' && dnd.activeItem.id === doc.id}
            />
          ))}

          {/* Collections with their documents */}
          {tree.collections?.map((node) => (
            <DroppableCollection
              key={node.collection_id}
              spaceId={spaceId}
              node={node}
              isOverTarget={dnd.overId === `collection:${node.collection_id}`}
              isValidTarget={dnd.activeItem !== null && dnd.isValidDrop({ kind: 'collection', id: node.collection_id, spaceId })}
              isDragging={dnd.activeItem?.kind === 'collection' && dnd.activeItem.id === node.collection_id}
              activeDragItem={dnd.activeItem}
              autoExpanded={dnd.autoExpandedIds.has(node.collection_id)}
            />
          ))}
        </nav>

        <SidebarDragOverlay activeItem={dnd.activeItem} />
      </DndContext>

      {/* Create doc at space level (no collection) */}
      <button
        onClick={() => setShowCreateDoc(true)}
        className="mt-2 flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        <Plus className="h-3 w-3" />
        New document
      </button>

      <CreateDocumentDialog
        open={showCreateDoc}
        spaceId={spaceId}
        onClose={() => setShowCreateDoc(false)}
      />
    </>
  )
}

// --- Draggable Document ---

function DraggableDocument({
  spaceId,
  doc,
  collectionId,
  isDragging,
}: {
  spaceId: string
  doc: TreeDoc
  collectionId: string | null
  isDragging: boolean
}) {
  const dragData: DragItem = {
    kind: 'document',
    id: doc.id,
    title: doc.title,
    icon: doc.icon,
    docType: doc.doc_type,
    collectionId,
    spaceId,
  }

  const { attributes, listeners, setNodeRef, transform } = useDraggable({
    id: `doc:${doc.id}`,
    data: dragData,
  })

  const typeColor = getDocTypeColor(doc.doc_type)

  const style = transform
    ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` }
    : undefined

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`group flex items-center rounded-md text-sm text-sidebar-foreground transition-colors hover:bg-muted ${
        isDragging ? 'opacity-30' : ''
      }`}
    >
      <span
        {...listeners}
        {...attributes}
        className="flex shrink-0 cursor-grab items-center px-0.5 text-muted-foreground opacity-0 group-hover:opacity-60"
        aria-label="Drag to reorder"
      >
        <GripVertical className="h-3 w-3" />
      </span>
      <Link
        to="/spaces/$spaceId/docs/$docId"
        params={{ spaceId, docId: doc.id }}
        className="flex flex-1 items-center gap-1.5 rounded-md px-1 py-1 transition-colors [&.active]:bg-muted [&.active]:text-accent"
      >
        <span
          className="inline-block h-1.5 w-1.5 shrink-0 rounded-full"
          style={{ backgroundColor: typeColor }}
        />
        <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate">{doc.title}</span>
      </Link>
    </div>
  )
}

// --- Droppable Collection ---

function DroppableCollection({
  spaceId,
  node,
  isOverTarget,
  isValidTarget,
  isDragging,
  activeDragItem,
  autoExpanded,
}: {
  spaceId: string
  node: TreeCollection
  isOverTarget: boolean
  isValidTarget: boolean
  isDragging: boolean
  activeDragItem: DragItem | null
  autoExpanded: boolean
}) {
  const [expanded, setExpanded] = useState(true)
  const [showCreateDoc, setShowCreateDoc] = useState(false)

  const isExpanded = expanded || autoExpanded

  const dropData: DropTarget = { kind: 'collection', id: node.collection_id, spaceId }

  const { setNodeRef: setDropRef, isOver } = useDroppable({
    id: `collection:${node.collection_id}`,
    data: dropData,
  })

  const dragData: DragItem = {
    kind: 'collection',
    id: node.collection_id,
    name: node.collection_name,
    icon: node.icon,
    parentId: node.parent_id ?? null,
    spaceId,
  }

  const { attributes, listeners, setNodeRef: setDragRef, transform } = useDraggable({
    id: `collection:${node.collection_id}`,
    data: dragData,
  })

  const isActiveOver = isOver || isOverTarget
  const showInvalid = isActiveOver && !isValidTarget && activeDragItem !== null
  const showValid = isActiveOver && isValidTarget

  const style = transform
    ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` }
    : undefined

  // Combine drag + drop refs.
  function setRefs(el: HTMLElement | null) {
    setDropRef(el)
    setDragRef(el)
  }

  return (
    <>
      <div
        ref={setRefs}
        style={style}
        className={`rounded-md transition-colors ${isDragging ? 'opacity-30' : ''} ${
          showValid ? 'bg-accent/10 ring-1 ring-accent/40' : ''
        } ${showInvalid ? 'bg-destructive/10 ring-1 ring-destructive/40' : ''}`}
      >
        <div className="group flex w-full items-center rounded-md px-1 py-1 text-sm text-sidebar-foreground transition-colors hover:bg-muted">
          <span
            {...listeners}
            {...attributes}
            className="flex shrink-0 cursor-grab items-center px-0.5 text-muted-foreground opacity-0 group-hover:opacity-60"
            aria-label="Drag to reorder"
          >
            <GripVertical className="h-3 w-3" />
          </span>
          <button
            onClick={() => setExpanded(!expanded)}
            className="flex flex-1 items-center gap-1.5"
          >
            <ChevronRight
              className={`h-3.5 w-3.5 shrink-0 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
            />
            <FolderOpen className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            <span className="truncate">{node.collection_name}</span>
          </button>
          <button
            onClick={() => setShowCreateDoc(true)}
            className="rounded p-0.5 text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100"
            title="New document"
          >
            <Plus className="h-3 w-3" />
          </button>
        </div>

        {isExpanded && node.documents.length > 0 && (
          <div className="ml-4 space-y-0.5 py-0.5">
            {node.documents.map((doc) => (
              <DraggableDocument
                key={doc.id}
                spaceId={spaceId}
                doc={doc}
                collectionId={node.collection_id}
                isDragging={activeDragItem?.kind === 'document' && activeDragItem.id === doc.id}
              />
            ))}
          </div>
        )}

        {/* Drop hint when dragging over empty or expanded collection */}
        {showValid && isExpanded && node.documents.length === 0 && (
          <div className="mx-4 my-1 rounded border border-dashed border-accent/50 px-2 py-1 text-xs text-muted-foreground">
            Drop here
          </div>
        )}
      </div>

      <CreateDocumentDialog
        open={showCreateDoc}
        spaceId={spaceId}
        collectionId={node.collection_id}
        onClose={() => setShowCreateDoc(false)}
      />
    </>
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
