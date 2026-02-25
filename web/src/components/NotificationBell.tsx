import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect, useRef } from 'react'
import { Bell, MessageSquare } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { api } from '@/api/client'
import type { Notification, NotificationCount } from '@/api/client'

interface NotificationBellProps {
  collapsed?: boolean
}

export function NotificationBell({ collapsed }: NotificationBellProps) {
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const { data: countData } = useQuery({
    queryKey: ['notification-count'],
    queryFn: () => api.get<NotificationCount>('/notifications/unread-count'),
    refetchInterval: 60000,
  })

  const { data: notifications } = useQuery({
    queryKey: ['notifications'],
    queryFn: () => api.get<Notification[]>('/notifications?limit=20'),
    enabled: open,
  })

  const markAllReadMutation = useMutation({
    mutationFn: () =>
      api.post('/notifications/mark-read', { all: true }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notification-count'] })
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })

  const markReadMutation = useMutation({
    mutationFn: (ids: string[]) =>
      api.post('/notifications/mark-read', { ids }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notification-count'] })
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })

  // Close on outside click.
  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  const unreadCount = countData?.count ?? 0

  const handleNotificationClick = (n: Notification) => {
    if (!n.is_read) {
      markReadMutation.mutate([n.id])
    }
    if (n.document_id) {
      // Navigate to document — we need spaceId but don't have it here,
      // so we'll use search to resolve.
      setOpen(false)
    }
  }

  const getTypeLabel = (type: string, actor: Notification['actor']) => {
    const name = actor?.display_name ?? 'Someone'
    switch (type) {
      case 'comment_mention':
        return `${name} mentioned you`
      case 'comment_reply':
        return `${name} replied to your comment`
      case 'document_comment':
        return `${name} commented on your document`
      default:
        return `${name} sent a notification`
    }
  }

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(!open)}
        className="relative rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        title="Notifications"
      >
        <Bell className="h-4 w-4" />
        {unreadCount > 0 && (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className={`absolute z-50 w-80 rounded-lg border border-border bg-card shadow-lg ${
          collapsed
            ? 'bottom-0 left-full ml-2'
            : 'left-0 top-full mt-2'
        }`}>
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <span className="text-sm font-semibold text-foreground">Notifications</span>
            {unreadCount > 0 && (
              <button
                onClick={() => markAllReadMutation.mutate()}
                className="text-xs text-accent hover:underline"
              >
                Mark all as read
              </button>
            )}
          </div>

          <div className="max-h-80 overflow-y-auto">
            {notifications?.length === 0 && (
              <div className="px-4 py-6 text-center text-sm text-muted-foreground">
                No notifications yet.
              </div>
            )}

            {notifications?.map((n) => (
              <button
                key={n.id}
                onClick={() => handleNotificationClick(n)}
                className={`flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/50 ${
                  !n.is_read ? 'bg-accent/5' : ''
                }`}
              >
                <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-accent/20 text-accent">
                  <MessageSquare className="h-3 w-3" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm text-foreground">
                    {getTypeLabel(n.type, n.actor)}
                  </p>
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    {formatDistanceToNow(new Date(n.created_at))} ago
                  </p>
                </div>
                {!n.is_read && (
                  <div className="mt-2 h-2 w-2 shrink-0 rounded-full bg-accent" />
                )}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
