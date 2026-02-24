# BookOwl — Sidebar Drag & Drop

## 1. Overview

Users can drag documents and collections in the sidebar tree to reorganise them. Moves are persisted immediately via the existing PUT endpoints.

**Library:** `@dnd-kit/core` + `@dnd-kit/sortable` + `@dnd-kit/utilities`

```bash
npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities
```

---

## 2. What Can Be Dragged Where

| Draggable | Valid drop targets | Invalid |
|-----------|-------------------|---------|
| Document | Any collection in the same space | Different space, root of space (must be in a collection) |
| Collection | Root of space, other collections at same level | Into itself, into its own descendants, different space |

Cross-space moves are not supported in v1 — documents are space-scoped and moving them would require reassigning space membership. Show a "Can't move between spaces" tooltip if user tries.

---

## 3. API Calls on Drop

**Move document to different collection:**
```
PUT /api/v1/documents/:id
Body: { "collection_id": "<target-collection-id>" }
```

**Reorder document within same collection:**
```
PUT /api/v1/documents/:id
Body: { "position": <new-position> }
```

**Move collection to different parent (or root):**
```
PUT /api/v1/spaces/:spaceId/collections/:id
Body: { "parent_id": "<target-collection-id>" }  // null = root level
```

**Reorder collection within same parent:**
```
PUT /api/v1/spaces/:spaceId/collections/:id
Body: { "position": <new-position> }
```

All existing endpoints — no backend changes required.

---

## 4. Visual Design

### 4.1 Drag Handle

Each sidebar row shows a `⠿` drag handle icon on hover (left of the row, before the icon/title). Hidden at rest to keep the sidebar clean.

```
                          ← at rest
⠿ 📄 Pod CrashLoopBackOff  ← on hover, handle appears
```

The entire row is also draggable (not just the handle) — the handle is a visual affordance, not the only drag initiator.

### 4.2 Drag Ghost

While dragging, a ghost element follows the cursor showing:
- The item's icon + title
- Muted background (`--card` with 90% opacity)
- Slight shadow and 4px border-radius
- Constrained width: 280px max

### 4.3 Drop Target Highlight

When dragging over a valid drop target:
- Collection row: accent-coloured left border + subtle background tint (`--accent/10`)
- "Drop here" indicator: a 2px accent-coloured horizontal line appears between items when reordering within a collection

When dragging over an invalid target (different space, collection's own descendant):
- Red tint background, `🚫` cursor

### 4.4 Collapse Behaviour

When dragging a document over a collapsed collection for 600ms, the collection auto-expands (standard tree drag-and-drop convention).

---

## 5. Component Structure

```
web/src/components/sidebar/
├── SidebarTree.tsx          ← wrap with DndContext here
├── SidebarSpaceSection.tsx  ← DroppableSpace (root level)
├── SidebarCollection.tsx    ← DraggableCollection + DroppableCollection
├── SidebarDocument.tsx      ← DraggableDocument
└── dnd/
    ├── useSidebarDnd.ts     ← DnD state, drag handlers, drop logic
    ├── DragOverlay.tsx      ← ghost element rendered in portal
    └── types.ts             ← DragItem type union
```

### 5.1 DragItem Type

```typescript
// web/src/components/sidebar/dnd/types.ts

export type DragItem =
  | { kind: 'document'; id: string; title: string; collectionId: string; spaceId: string }
  | { kind: 'collection'; id: string; name: string; parentId: string | null; spaceId: string }
```

### 5.2 useSidebarDnd Hook

```typescript
// web/src/components/sidebar/dnd/useSidebarDnd.ts

export function useSidebarDnd() {
  const [activeItem, setActiveItem] = useState<DragItem | null>(null)

  function handleDragStart(event: DragStartEvent) {
    setActiveItem(event.active.data.current as DragItem)
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    setActiveItem(null)

    if (!over || active.id === over.id) return

    const dragged = active.data.current as DragItem
    const target = over.data.current as DropTarget

    // Validate: no cross-space moves
    if (dragged.spaceId !== target.spaceId) return

    // Validate: no dropping collection into itself or descendant
    if (dragged.kind === 'collection' && isDescendant(dragged.id, target.id)) return

    if (dragged.kind === 'document') {
      moveDocument(dragged.id, target.collectionId, target.position)
    } else {
      moveCollection(dragged.id, target.parentId, target.position)
    }
  }

  return { activeItem, handleDragStart, handleDragEnd }
}
```

### 5.3 Optimistic Updates

Use TanStack Query's `useMutation` with `onMutate` / `onError` for optimistic UI:

```typescript
const moveDocumentMutation = useMutation({
  mutationFn: ({ id, collectionId }: { id: string; collectionId: string }) =>
    api.put(`/documents/${id}`, { collection_id: collectionId }),

  onMutate: async ({ id, collectionId }) => {
    // Cancel any in-flight refetches
    await queryClient.cancelQueries({ queryKey: ['space-tree', spaceId] })
    // Snapshot current tree
    const previous = queryClient.getQueryData(['space-tree', spaceId])
    // Optimistically update tree
    queryClient.setQueryData(['space-tree', spaceId], (old) =>
      moveDocumentInTree(old, id, collectionId)
    )
    return { previous }
  },

  onError: (_, __, context) => {
    // Revert on failure
    queryClient.setQueryData(['space-tree', spaceId], context?.previous)
    toast.error('Failed to move document')
  },
})
```

---

## 6. Keyboard Accessibility

`@dnd-kit` has built-in keyboard support:
- Space/Enter to pick up an item
- Arrow keys to move through valid drop targets
- Space/Enter to drop, Escape to cancel

No extra work required — it comes from `@dnd-kit/core` out of the box.

---

## 7. Tasks

Add to `docs/08-tasks.md` as Phase 16:

### Phase 16: Sidebar Drag & Drop

- [ ] `npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities`
- [ ] `web/src/components/sidebar/dnd/types.ts` — DragItem + DropTarget types
- [ ] `web/src/components/sidebar/dnd/useSidebarDnd.ts` — drag start/end handlers, validation, mutation calls
- [ ] `web/src/components/sidebar/dnd/DragOverlay.tsx` — ghost element (icon + title, 280px, shadow)
- [ ] Wrap `SidebarTree.tsx` with `DndContext` + `DragOverlay`
- [ ] `SidebarDocument.tsx` — add `useDraggable`, show drag handle on hover
- [ ] `SidebarCollection.tsx` — add `useDraggable` + `useDroppable`, drop highlight styles
- [ ] `SidebarSpaceSection.tsx` — add `useDroppable` for root-level collection drops
- [ ] Optimistic tree update on drop using TanStack Query `onMutate`/`onError`
- [ ] Revert + toast on API failure
- [ ] Auto-expand collapsed collection after 600ms hover while dragging
- [ ] Block cross-space drops (red tint + no-drop cursor)
- [ ] Block collection-into-descendant drops
- [ ] Manual test: drag document between collections, reload — persisted correctly
- [ ] Manual test: drag document to wrong space — blocked with visual feedback
- [ ] Manual test: keyboard drag (Space to pick up, arrows, Space to drop)
