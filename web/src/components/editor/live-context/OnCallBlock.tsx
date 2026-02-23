import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { Settings } from 'lucide-react'

interface OnCallBlockProps {
  rosterId: string | null
  label: string | null
  editable: boolean
}

interface LiveContextEnvelope {
  data: {
    primary?: { name: string; timezone?: string }
    secondary?: { name: string; timezone?: string }
    shift_start?: string
    shift_end?: string
  }
  cached_at: string | null
  source: string
}

export function OnCallBlock({ rosterId, label, editable }: OnCallBlockProps) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['live-context', 'oncall', rosterId],
    queryFn: () =>
      api.get<LiveContextEnvelope>(`/live-context/oncall/${rosterId}`),
    enabled: !!rosterId,
    refetchInterval: 30_000,
  })

  if (!rosterId) {
    return (
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">On-Call block — not configured</span>
        {editable && <Settings className="h-4 w-4 text-muted-foreground" />}
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="text-sm text-muted-foreground">
        Loading on-call data...
      </div>
    )
  }

  if (isError || data?.source === 'unavailable') {
    return (
      <div className="text-sm text-muted-foreground">
        On-call data unavailable
      </div>
    )
  }

  const oncall = data?.data

  return (
    <div>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm font-medium">
          <span>🦉</span>
          <span>On-Call{label ? ` — ${label}` : ''}</span>
        </div>
        {editable && (
          <Settings className="h-4 w-4 cursor-pointer text-muted-foreground hover:text-foreground" />
        )}
      </div>
      <div className="mt-2 space-y-1 text-sm">
        {oncall?.primary && (
          <div>
            <span className="text-muted-foreground">Primary: </span>
            <span>{oncall.primary.name}</span>
            {oncall.primary.timezone && (
              <span className="text-muted-foreground"> ({oncall.primary.timezone})</span>
            )}
          </div>
        )}
        {oncall?.secondary && (
          <div>
            <span className="text-muted-foreground">Secondary: </span>
            <span>{oncall.secondary.name}</span>
            {oncall.secondary.timezone && (
              <span className="text-muted-foreground"> ({oncall.secondary.timezone})</span>
            )}
          </div>
        )}
      </div>
      {data?.cached_at && (
        <div className="mt-2 text-xs text-muted-foreground">
          {data.source === 'cache' ? 'cached' : 'live'}
        </div>
      )}
    </div>
  )
}
