import type { ReactNode } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { BookOpen, Search, FileText, Clock } from 'lucide-react'

export const Route = createFileRoute('/')({
  component: HomePage,
})

function HomePage() {
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
        />
        <QuickAction
          icon={<Search className="h-5 w-5" />}
          title="Search Docs"
          description="Find documents across all spaces"
        />
        <QuickAction
          icon={<FileText className="h-5 w-5" />}
          title="New Document"
          description="Create a new document or runbook"
        />
        <QuickAction
          icon={<Clock className="h-5 w-5" />}
          title="Recent Changes"
          description="View recently updated documents"
        />
      </div>

      <footer className="mt-16 text-center text-sm text-muted-foreground">
        BookOwl v0.1.0 — A Wisbric product · Part of the NightOwl platform
      </footer>
    </div>
  )
}

function QuickAction({
  icon,
  title,
  description,
}: {
  icon: ReactNode
  title: string
  description: string
}) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-card p-4 transition-colors hover:border-accent/50 hover:bg-card/80">
      <div className="mt-0.5 text-accent">{icon}</div>
      <div>
        <h3 className="font-medium">{title}</h3>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </div>
  )
}
