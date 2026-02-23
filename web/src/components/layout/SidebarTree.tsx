import { Link } from '@tanstack/react-router'
import { ChevronRight, FileText, FolderOpen } from 'lucide-react'
import { useState } from 'react'
import type { TreeNode, TreeDoc } from '@/api/client'

interface SidebarTreeProps {
  spaceId: string
  nodes: TreeNode[]
}

export function SidebarTree({ spaceId, nodes }: SidebarTreeProps) {
  return (
    <nav className="space-y-0.5">
      {nodes.map((node) => (
        <CollectionNode key={node.collection_id} spaceId={spaceId} node={node} />
      ))}
    </nav>
  )
}

function CollectionNode({ spaceId, node }: { spaceId: string; node: TreeNode }) {
  const [expanded, setExpanded] = useState(true)

  return (
    <div>
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
      >
        <ChevronRight
          className={`h-3.5 w-3.5 shrink-0 transition-transform ${expanded ? 'rotate-90' : ''}`}
        />
        <FolderOpen className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate">{node.collection_name}</span>
      </button>

      {expanded && node.documents.length > 0 && (
        <div className="ml-4 space-y-0.5 py-0.5">
          {node.documents.map((doc) => (
            <DocumentLink key={doc.id} spaceId={spaceId} doc={doc} />
          ))}
        </div>
      )}
    </div>
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
