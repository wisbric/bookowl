import { useState, type ReactNode } from 'react'
import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { BookOpen, Search, FileText, Clock } from 'lucide-react'
import { api } from '@/api/client'
import type { Space } from '@/api/client'
import { CreateDocumentDialog } from '@/components/CreateDocumentDialog'

export const Route = createFileRoute('/')({
  component: HomePage,
})

function HomePage() {
  const [showCreateDoc, setShowCreateDoc] = useState(false)

  const { data: spaces } = useQuery({
    queryKey: ['spaces'],
    queryFn: () => api.get<Space[]>('/spaces'),
  })

  const firstSpaceId = spaces?.[0]?.id
  const hasSpaces = !!firstSpaceId

  return (
    <div className="mx-auto max-w-3xl px-8 py-12">
      <div className="mb-8">
        <h1 className="text-3xl font-bold tracking-tight">Welcome to BookOwl</h1>
        <p className="mt-2 text-muted-foreground">
          Where your operational knowledge lives.
        </p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <QuickAction
          icon={<BookOpen className="h-5 w-5" />}
          title="Browse Spaces"
          description="Explore your documentation spaces"
          to={hasSpaces ? `/spaces/${firstSpaceId}` : undefined}
          disabled={!hasSpaces}
          disabledReason="No spaces yet — create one first"
        />
        <QuickAction
          icon={<Search className="h-5 w-5" />}
          title="Search Docs"
          description="Find documents across all spaces"
          to="/search"
        />
        <QuickAction
          icon={<FileText className="h-5 w-5" />}
          title="New Document"
          description="Create a new document or runbook"
          onClick={hasSpaces ? () => setShowCreateDoc(true) : undefined}
          disabled={!hasSpaces}
          disabledReason="No spaces yet — create one first"
        />
        <QuickAction
          icon={<Clock className="h-5 w-5" />}
          title="Recent Changes"
          description="View recently updated documents"
          to="/recent"
        />
      </div>

      <footer className="mt-16 text-center text-sm text-muted-foreground">
        BookOwl v0.1.0 — A Wisbric product · Part of the NightOwl platform
      </footer>

      {firstSpaceId && (
        <CreateDocumentDialog
          open={showCreateDoc}
          spaceId={firstSpaceId}
          onClose={() => setShowCreateDoc(false)}
        />
      )}
    </div>
  )
}

function QuickAction({
  icon,
  title,
  description,
  to,
  onClick,
  disabled,
  disabledReason,
}: {
  icon: ReactNode
  title: string
  description: string
  to?: string
  onClick?: () => void
  disabled?: boolean
  disabledReason?: string
}) {
  const className =
    'flex items-start gap-3 rounded-lg border border-border bg-card p-4 transition-colors hover:border-accent/50 hover:bg-card/80'

  const content = (
    <>
      <div className="mt-0.5 text-accent">{icon}</div>
      <div>
        <h3 className="font-medium">{title}</h3>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </>
  )

  if (disabled || (!to && !onClick)) {
    return (
      <div className={`${className} cursor-not-allowed opacity-50`} title={disabledReason}>
        {content}
      </div>
    )
  }

  if (onClick) {
    return (
      <button type="button" onClick={onClick} className={`${className} w-full cursor-pointer text-left`}>
        {content}
      </button>
    )
  }

  return (
    <Link to={to!} className={className}>
      {content}
    </Link>
  )
}
