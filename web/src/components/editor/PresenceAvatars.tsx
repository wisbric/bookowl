import { useEffect, useState } from 'react'
import type { HocuspocusProvider } from '@hocuspocus/provider'
import { getInitials } from '@/lib/utils'

interface ActiveUser {
  id: string
  name: string
  color: string
}

export function PresenceAvatars({ provider }: { provider: HocuspocusProvider | null }) {
  const [users, setUsers] = useState<ActiveUser[]>([])

  useEffect(() => {
    if (!provider?.awareness) return

    const awareness = provider.awareness

    const update = () => {
      const states = Array.from(awareness.getStates().values())
      const active: ActiveUser[] = []
      for (const s of states) {
        const u = (s as Record<string, unknown>).user as ActiveUser | undefined
        if (u?.id) active.push(u)
      }
      setUsers(active)
    }

    awareness.on('change', update)
    update()
    return () => {
      awareness.off('change', update)
    }
  }, [provider])

  if (users.length === 0) return null

  const visible = users.slice(0, 3)
  const overflow = users.length - 3

  return (
    <div className="flex items-center gap-1">
      {visible.map((u) => (
        <div
          key={u.id}
          className="flex h-7 w-7 items-center justify-center rounded-full text-xs font-medium text-white"
          style={{ backgroundColor: u.color }}
          title={`${u.name} — editing now`}
        >
          {getInitials(u.name)}
        </div>
      ))}
      {overflow > 0 && (
        <div className="flex h-7 w-7 items-center justify-center rounded-full bg-muted text-xs">
          +{overflow}
        </div>
      )}
    </div>
  )
}
