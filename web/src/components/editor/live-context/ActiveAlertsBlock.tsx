import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { Settings } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'

interface ActiveAlertsBlockProps {
  serviceName: string | null
  label: string | null
  editable: boolean
}

interface LiveContextEnvelope {
  data: {
    items?: Array<{
      id: string
      title: string
      severity: string
      created_at: string
    }>
  }
  cached_at: string | null
  source: string
}

export function ActiveAlertsBlock({ serviceName, label, editable }: ActiveAlertsBlockProps) {
  const queryParams = serviceName
    ? `?service=${encodeURIComponent(serviceName)}`
    : '?severity=critical,major&limit=5'

  const { data, isLoading, isError } = useQuery({
    queryKey: ['live-context', 'alerts', serviceName],
    queryFn: () =>
      api.get<LiveContextEnvelope>(`/live-context/alerts${queryParams}`),
    refetchInterval: 30_000,
  })

  if (isLoading) {
    return <div className="text-sm text-muted-foreground">Loading alerts...</div>
  }

  if (isError || data?.source === 'unavailable') {
    return <div className="text-sm text-muted-foreground">Alerts unavailable</div>
  }

  const alerts = data?.data?.items || []

  return (
    <div>
      <div className="flex items-center justify-between">
        <div className="text-sm font-medium">
          {label || 'Active Alerts'}
        </div>
        {editable && (
          <Settings className="h-4 w-4 cursor-pointer text-muted-foreground hover:text-foreground" />
        )}
      </div>
      {alerts.length === 0 ? (
        <div className="mt-1 text-sm text-muted-foreground">No active alerts</div>
      ) : (
        <div className="mt-2 space-y-1">
          {alerts.map((alert) => (
            <div
              key={alert.id}
              className="flex items-center gap-2 text-sm"
            >
              <SeverityDot severity={alert.severity} />
              <span className="truncate">{alert.title}</span>
              <span className="shrink-0 text-xs text-muted-foreground">
                {formatDistanceToNow(new Date(alert.created_at))} ago
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function SeverityDot({ severity }: { severity: string }) {
  const color = {
    critical: '#DC2626',
    major: '#F59E0B',
    minor: '#3B82F6',
    info: '#6B7280',
  }[severity] || '#6B7280'

  return (
    <span
      className="inline-block h-2 w-2 shrink-0 rounded-full"
      style={{ backgroundColor: color }}
    />
  )
}
