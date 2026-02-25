import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { useAuth } from '@/auth/auth-provider'

interface CollabTokenResponse {
  token: string
}

export function useCollabToken(enabled: boolean): string | null {
  const { devMode } = useAuth()

  const { data } = useQuery({
    queryKey: ['collab-token'],
    queryFn: () => api.get<CollabTokenResponse>('/collab/token'),
    enabled: enabled && !devMode,
    refetchInterval: 4 * 60 * 1000, // refresh every 4 min (token TTL is 5 min)
    staleTime: 3 * 60 * 1000,
  })

  if (!enabled) return null
  if (devMode) return 'dev-token'
  return data?.token ?? null
}
