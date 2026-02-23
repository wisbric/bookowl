import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { Settings } from 'lucide-react'

interface ServiceStatusBlockProps {
  serviceName: string | null
  label: string | null
  editable: boolean
}

interface LiveContextEnvelope {
  data: {
    items?: Array<{ severity: string; title: string; created_at: string }>
    total?: number
  }
  cached_at: string | null
  source: string
}

export function ServiceStatusBlock({ serviceName, label, editable }: ServiceStatusBlockProps) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['live-context', 'service', serviceName],
    queryFn: () =>
      api.get<LiveContextEnvelope>(`/live-context/service/${serviceName}`),
    enabled: !!serviceName,
    refetchInterval: 30_000,
  })

  if (!serviceName) {
    return (
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">Service Status — not configured</span>
        {editable && <Settings className="h-4 w-4 text-muted-foreground" />}
      </div>
    )
  }

  if (isLoading) {
    return <div className="text-sm text-muted-foreground">Loading service status...</div>
  }

  if (isError || data?.source === 'unavailable') {
    return <div className="text-sm text-muted-foreground">Service status unavailable</div>
  }

  const alerts = data?.data?.items || []
  const total = data?.data?.total || alerts.length
  const criticalCount = alerts.filter((a) => a.severity === 'critical').length
  const warningCount = alerts.filter((a) => a.severity === 'warning' || a.severity === 'major').length

  const statusIcon = total > 0 ? '🔴' : '🟢'

  return (
    <div>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm font-medium">
          <span>{statusIcon}</span>
          <span>{label || serviceName}</span>
        </div>
        {editable && (
          <Settings className="h-4 w-4 cursor-pointer text-muted-foreground hover:text-foreground" />
        )}
      </div>
      <div className="mt-1 text-sm text-muted-foreground">
        {total > 0 ? (
          <>
            {total} active alert{total !== 1 ? 's' : ''}
            {(criticalCount > 0 || warningCount > 0) && (
              <> · {criticalCount > 0 && `${criticalCount} critical`}{criticalCount > 0 && warningCount > 0 && ', '}{warningCount > 0 && `${warningCount} warning`}</>
            )}
          </>
        ) : (
          'No active alerts'
        )}
      </div>
    </div>
  )
}
