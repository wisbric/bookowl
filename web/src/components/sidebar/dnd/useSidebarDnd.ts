import { useState, useCallback, useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { DragStartEvent, DragEndEvent, DragOverEvent } from '@dnd-kit/core'
import { api } from '@/api/client'
import type { Document, Collection, SpaceTree, TreeDoc } from '@/api/client'
import type { DragItem, DropTarget } from './types'

export function useSidebarDnd(spaceId: string) {
  const [activeItem, setActiveItem] = useState<DragItem | null>(null)
  const [overId, setOverId] = useState<string | null>(null)
  const queryClient = useQueryClient()
  const autoExpandTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [autoExpandedIds, setAutoExpandedIds] = useState<Set<string>>(new Set())

  // Move document to a different collection.
  const moveDocMutation = useMutation({
    mutationFn: async ({ docId, collectionId }: { docId: string; collectionId: string }) => {
      const doc = await api.get<Document>(`/documents/${docId}`)
      return api.put<Document>(`/documents/${docId}`, {
        title: doc.title,
        slug: doc.slug,
        content: doc.content,
        collection_id: collectionId,
        doc_type: doc.doc_type,
        status: doc.status,
        tags: doc.tags,
        icon: doc.icon,
        position: doc.position,
      })
    },
    onMutate: async ({ docId, collectionId }) => {
      await queryClient.cancelQueries({ queryKey: ['space-tree', spaceId] })
      const previous = queryClient.getQueryData<SpaceTree>(['space-tree', spaceId])
      queryClient.setQueryData<SpaceTree>(['space-tree', spaceId], (old) => {
        if (!old) return old
        return moveDocInTree(old, docId, collectionId)
      })
      return { previous }
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['space-tree', spaceId], context.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['space-tree', spaceId] })
    },
  })

  // Reorder collection (change position or parent).
  const moveColMutation = useMutation({
    mutationFn: async ({ colId, position }: { colId: string; position: number }) => {
      const col = await api.get<Collection>(`/spaces/${spaceId}/collections/${colId}`)
      return api.put<Collection>(`/spaces/${spaceId}/collections/${colId}`, {
        name: col.name,
        slug: col.slug,
        icon: col.icon,
        position,
        parent_id: col.parent_id,
      })
    },
    onMutate: async ({ colId, position }) => {
      await queryClient.cancelQueries({ queryKey: ['space-tree', spaceId] })
      const previous = queryClient.getQueryData<SpaceTree>(['space-tree', spaceId])
      queryClient.setQueryData<SpaceTree>(['space-tree', spaceId], (old) => {
        if (!old) return old
        return reorderCollectionInTree(old, colId, position)
      })
      return { previous }
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['space-tree', spaceId], context.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['space-tree', spaceId] })
    },
  })

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveItem(event.active.data.current as DragItem)
  }, [])

  const handleDragOver = useCallback((event: DragOverEvent) => {
    const overId = event.over?.id?.toString() ?? null
    setOverId(overId)

    // Auto-expand collapsed collection after 600ms hover.
    if (autoExpandTimer.current) {
      clearTimeout(autoExpandTimer.current)
      autoExpandTimer.current = null
    }

    if (overId && event.over?.data.current) {
      const target = event.over.data.current as DropTarget
      if (target.kind === 'collection') {
        autoExpandTimer.current = setTimeout(() => {
          setAutoExpandedIds((prev) => new Set(prev).add(target.id))
        }, 600)
      }
    }
  }, [])

  const handleDragEnd = useCallback((event: DragEndEvent) => {
    const { active, over } = event
    setActiveItem(null)
    setOverId(null)
    setAutoExpandedIds(new Set())

    if (autoExpandTimer.current) {
      clearTimeout(autoExpandTimer.current)
      autoExpandTimer.current = null
    }

    if (!over) return

    const dragged = active.data.current as DragItem
    const target = over.data.current as DropTarget

    if (!dragged || !target) return

    // Block cross-space moves.
    if (dragged.spaceId !== target.spaceId) return

    if (dragged.kind === 'document' && target.kind === 'collection') {
      // Don't move if already in the same collection.
      if (dragged.collectionId === target.id) return
      moveDocMutation.mutate({ docId: dragged.id, collectionId: target.id })
    } else if (dragged.kind === 'collection' && target.kind === 'collection') {
      // Reorder collection — place it after the target.
      if (dragged.id === target.id) return
      const tree = queryClient.getQueryData<SpaceTree>(['space-tree', spaceId])
      const targetIdx = tree?.collections?.findIndex((c) => c.collection_id === target.id) ?? -1
      if (targetIdx >= 0) {
        moveColMutation.mutate({ colId: dragged.id, position: targetIdx + 1 })
      }
    } else if (dragged.kind === 'collection' && target.kind === 'space-root') {
      // Move collection to position 0 (top of root).
      moveColMutation.mutate({ colId: dragged.id, position: 0 })
    }
  }, [spaceId, moveDocMutation, moveColMutation, queryClient])

  const handleDragCancel = useCallback(() => {
    setActiveItem(null)
    setOverId(null)
    setAutoExpandedIds(new Set())
    if (autoExpandTimer.current) {
      clearTimeout(autoExpandTimer.current)
      autoExpandTimer.current = null
    }
  }, [])

  // Check if a given drop target is valid for the current drag item.
  function isValidDrop(target: DropTarget): boolean {
    if (!activeItem) return false
    if (activeItem.spaceId !== target.spaceId) return false

    if (activeItem.kind === 'document') {
      // Documents can only drop onto collections.
      return target.kind === 'collection' && target.id !== activeItem.collectionId
    }

    if (activeItem.kind === 'collection') {
      // Collections can drop onto space-root or other collections (for reorder).
      if (target.kind === 'space-root') return true
      if (target.kind === 'collection') return target.id !== activeItem.id
    }

    return false
  }

  return {
    activeItem,
    overId,
    autoExpandedIds,
    handleDragStart,
    handleDragOver,
    handleDragEnd,
    handleDragCancel,
    isValidDrop,
  }
}

// --- Tree manipulation helpers ---

function moveDocInTree(tree: SpaceTree, docId: string, targetCollectionId: string): SpaceTree {
  let movedDoc: TreeDoc | null = null

  // Remove from uncollected documents.
  const documents = (tree.documents ?? []).filter((d) => {
    if (d.id === docId) { movedDoc = d; return false }
    return true
  })

  // Remove from collections.
  const collections = (tree.collections ?? []).map((col) => {
    const filtered = col.documents.filter((d) => {
      if (d.id === docId) { movedDoc = d; return false }
      return true
    })
    return { ...col, documents: filtered }
  })

  if (!movedDoc) return tree

  // Add to target collection.
  return {
    documents,
    collections: collections.map((col) => {
      if (col.collection_id === targetCollectionId) {
        return { ...col, documents: [...col.documents, movedDoc!] }
      }
      return col
    }),
  }
}

function reorderCollectionInTree(tree: SpaceTree, colId: string, newPosition: number): SpaceTree {
  const collections = [...(tree.collections ?? [])]
  const currentIdx = collections.findIndex((c) => c.collection_id === colId)
  if (currentIdx < 0) return tree

  const [moved] = collections.splice(currentIdx, 1)
  const insertIdx = Math.min(newPosition, collections.length)
  collections.splice(insertIdx, 0, { ...moved, position: newPosition })

  return { ...tree, collections }
}
