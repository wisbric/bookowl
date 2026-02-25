import { HocuspocusProvider } from '@hocuspocus/provider'
import { useEffect, useMemo, useRef, useState } from 'react'
import * as Y from 'yjs'

interface CollabProviderOptions {
  documentId: string
  tenantSlug: string
  token: string | null
  wsUrl: string
  enabled: boolean
}

export function useCollabProvider({ documentId, tenantSlug, token, wsUrl, enabled }: CollabProviderOptions) {
  const ydoc = useMemo(() => new Y.Doc(), [documentId, tenantSlug])
  const [provider, setProvider] = useState<HocuspocusProvider | null>(null)
  const [synced, setSynced] = useState(false)
  const providerRef = useRef<HocuspocusProvider | null>(null)

  useEffect(() => {
    if (!enabled || !token || !wsUrl) {
      setProvider(null)
      setSynced(false)
      return
    }

    const hp = new HocuspocusProvider({
      url: wsUrl,
      name: `${tenantSlug}/${documentId}`,
      document: ydoc,
      token,
      onSynced() {
        setSynced(true)
      },
      onAuthenticationFailed() {
        console.error('Collab auth failed')
      },
    })

    providerRef.current = hp
    setProvider(hp)

    return () => {
      hp.destroy()
      providerRef.current = null
      setProvider(null)
      setSynced(false)
    }
  }, [documentId, tenantSlug, token, wsUrl, enabled, ydoc])

  return { ydoc, provider, synced }
}
