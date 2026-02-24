export type DragItem =
  | { kind: 'document'; id: string; title: string; icon?: string; docType: string; collectionId: string | null; spaceId: string }
  | { kind: 'collection'; id: string; name: string; icon?: string; parentId: string | null; spaceId: string }

export type DropTarget =
  | { kind: 'collection'; id: string; spaceId: string }
  | { kind: 'space-root'; spaceId: string }
