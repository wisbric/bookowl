import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from 'react'
import { api, type ProfileResponse } from '@/api/client'

interface AuthContextValue {
  isLoading: boolean
  isAuthenticated: boolean
  profile: ProfileResponse | null
  devMode: boolean
  oidcEnabled: boolean
  login: (user: ProfileResponse) => void
  logout: () => Promise<void>
  refreshProfile: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

interface AuthProviderProps {
  children: ReactNode
  oidcEnabled: boolean
}

export function AuthProvider({ children, oidcEnabled }: AuthProviderProps) {
  const devMode = !oidcEnabled && import.meta.env.DEV
  const [isLoading, setIsLoading] = useState(true)
  const [profile, setProfile] = useState<ProfileResponse | null>(null)

  const fetchProfile = useCallback(async () => {
    try {
      const p = await api.get<ProfileResponse>('/profile')
      setProfile(p)
      return true
    } catch {
      setProfile(null)
      return false
    }
  }, [])

  useEffect(() => {
    // On mount: check for existing session via /auth/me (cookie-based).
    fetchProfile().finally(() => setIsLoading(false))
  }, [fetchProfile])

  const login = useCallback((user: ProfileResponse) => {
    setProfile(user)
  }, [])

  const logout = useCallback(async () => {
    try {
      await fetch('/auth/logout', { method: 'POST', credentials: 'same-origin' })
    } catch {
      // Ignore network errors.
    }
    setProfile(null)
    window.location.href = '/login'
  }, [])

  const refreshProfile = useCallback(async () => {
    await fetchProfile()
  }, [fetchProfile])

  const isAuthenticated = devMode || profile !== null

  return (
    <AuthContext.Provider
      value={{
        isLoading,
        isAuthenticated,
        profile,
        devMode,
        oidcEnabled,
        login,
        logout,
        refreshProfile,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return ctx
}
