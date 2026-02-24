import { Link } from '@tanstack/react-router'
import { ChevronRight, FileText, FolderOpen, Plus } from 'lucide-react'
import { useState } from 'react'
import type { SpaceTree, TreeCollection, TreeDoc } from '@/api/client'
import { CreateDocumentDialog } from '@/components/CreateDocumentDialog'

interface SidebarTreeProps {
  spaceId: string
  tree: SpaceTree
}

export function SidebarTree({ spaceId, tree }: SidebarTreeProps) {
  const [showCreateDoc, setShowCreateDoc] = useState(false)

  return (
    <>
      <nav className="space-y-0.5">
        {/* Uncollected documents (top-level, no collection) */}
        {tree.documents?.map((doc) => (
          <DocumentLink key={doc.id} spaceId={spaceId} doc={doc} />
        ))}

        {/* Collections with their documents */}
        {tree.collections?.map((node) => (
          <CollectionNode key={node.collection_id} spaceId={spaceId} node={node} />
        ))}
      </nav>

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

function CollectionNode({ spaceId, node }: { spaceId: string; node: TreeCollection }) {
  const [expanded, setExpanded] = useState(true)
  const [showCreateDoc, setShowCreateDoc] = useState(false)

  return (
    <>
      <div>
        <div className="group flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-sm text-sidebar-foreground transition-colors hover:bg-muted">
          <button
            onClick={() => setExpanded(!expanded)}
            className="flex flex-1 items-center gap-1.5"
          >
            <ChevronRight
              className={`h-3.5 w-3.5 shrink-0 transition-transform ${expanded ? 'rotate-90' : ''}`}
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

        {expanded && node.documents.length > 0 && (
          <div className="ml-4 space-y-0.5 py-0.5">
            {node.documents.map((doc) => (
              <DocumentLink key={doc.id} spaceId={spaceId} doc={doc} />
            ))}
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

function DocumentLink({ spaceId, doc }: { spaceId: string; doc: TreeDoc }) {
  const typeColor = getDocTypeColor(doc.doc_type)

  return (
    <Link
      to="/spaces/$spaceId/docs/$docId"
      params={{ spaceId, docId: doc.id }}
      className="flex items-center gap-1.5 rounded-md px-2 py-1 text-sm text-sidebar-foreground transition-colors hover:bg-muted [&.active]:bg-muted [&.active]:text-accent"
    >
      <span
        className="inline-block h-1.5 w-1.5 shrink-0 rounded-full"
        style={{ backgroundColor: typeColor }}
      />
      <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      <span className="truncate">{doc.title}</span>
    </Link>
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
