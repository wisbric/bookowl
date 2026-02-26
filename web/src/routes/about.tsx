import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'

export const Route = createFileRoute('/about')({
  component: AboutPage,
})

interface StatusInfo {
  version: string
  commit_sha: string
  uptime: string
}

function AboutPage() {
  const { data: status, isLoading } = useQuery({
    queryKey: ['public-status'],
    queryFn: async () => {
      const res = await fetch('/status')
      if (!res.ok) throw new Error('Failed to fetch status')
      return res.json() as Promise<StatusInfo>
    },
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <p className="text-sm text-muted-foreground">Loading...</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-6 py-8">
      <div className="flex flex-col items-center gap-4 text-center">
        <img src="/owl.png" alt="BookOwl" className="h-20 w-auto" />
        <h1 className="text-3xl font-bold">BookOwl</h1>
        <p className="text-muted-foreground">
          Knowledge management platform for teams — runbooks, post-mortems, and
          structured documentation.
        </p>
      </div>

      <div className="rounded-lg border border-border bg-card text-card-foreground shadow-sm">
        <div className="flex flex-col space-y-1.5 p-4">
          <h3 className="font-semibold leading-none tracking-tight">Build Info</h3>
        </div>
        <div className="space-y-3 p-4 pt-0 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Version</span>
            <span className="font-mono">{status?.version ?? '—'}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Commit</span>
            <span className="font-mono">{status?.commit_sha ?? '—'}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Uptime</span>
            <span className="font-mono">{status?.uptime ?? '—'}</span>
          </div>
        </div>
      </div>

      <p className="text-center text-xs text-muted-foreground">
        A Wisbric product
      </p>
    </div>
  )
}
