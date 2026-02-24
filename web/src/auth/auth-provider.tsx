import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from 'react'
import type { UserManager, User } from 'oidc-client-ts'
import { setTokenProvider, setSessionMode, api, type ProfileResponse } from '@/api/client'

interface AuthContextValue {
  isLoading: boolean
  isAuthenticated: boolean
  user: User | null
  profile: ProfileResponse | null
  devMode: boolean
  oidcEnabled: boolean
  login: () => Promise<void>
  logout: () => Promise<void>
  refreshProfile: () => Promise<void>
  getAccessToken: () => Promise<string | null>
}

const AuthContext = createContext<AuthContextValue | null>(null)

interface AuthProviderProps {
  children: ReactNode
  userManager: UserManager | null
  oidcEnabled: boolean
}

export function AuthProvider({ children, userManager, oidcEnabled }: AuthProviderProps) {
  const devMode = !oidcEnabled && userManager === null
  const [isLoading, setIsLoading] = useState(true)
  const [user, setUser] = useState<User | null>(null)
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
    if (devMode) {
      // Dev mode: no auth needed, just fetch profile for display.
      setSessionMode(false)
      fetchProfile().finally(() => setIsLoading(false))
      return
    }

    if (userManager) {
      // OIDC mode: wire token provider.
      setTokenProvider(async () => {
        const u = await userManager.getUser()
        return u?.access_token ?? null
      })

      userManager.getUser().then(async (u) => {
        if (u && !u.expired) {
          setUser(u)
          await fetchProfile()
        }
        setIsLoading(false)
      })

      const onUserLoaded = (u: User) => {
        setUser(u)
        fetchProfile()
      }
      const onUserUnloaded = () => {
        setUser(null)
        setProfile(null)
      }

      userManager.events.addUserLoaded(onUserLoaded)
      userManager.events.addUserUnloaded(onUserUnloaded)

      return () => {
        userManager.events.removeUserLoaded(onUserLoaded)
        userManager.events.removeUserUnloaded(onUserUnloaded)
      }
    }

    // No OIDC, not dev mode: try session cookie.
    setSessionMode(true)
    fetchProfile().then((ok) => {
      if (!ok) {
        setSessionMode(false)
      }
      setIsLoading(false)
    })
  }, [userManager, devMode, fetchProfile])

  const login = useCallback(async () => {
    if (userManager) {
      await userManager.signinRedirect()
    }
  }, [userManager])

  const logout = useCallback(async () => {
    if (userManager) {
      await userManager.signoutRedirect()
    } else {
      // Session-based logout.
      try {
        await fetch('/auth/logout', { method: 'POST', credentials: 'same-origin' })
      } catch {
        // Ignore network errors.
      }
      setProfile(null)
      setSessionMode(false)
      window.location.href = '/login'
    }
  }, [userManager])

  const refreshProfile = useCallback(async () => {
    await fetchProfile()
  }, [fetchProfile])

  const getAccessToken = useCallback(async () => {
    if (!userManager) return null
    const u = await userManager.getUser()
    return u?.access_token ?? null
  }, [userManager])

  const isAuthenticated = devMode || (user !== null && !user.expired) || profile !== null

  return (
    <AuthContext.Provider
      value={{
        isLoading,
        isAuthenticated,
        user,
        profile,
        devMode,
        oidcEnabled,
        login,
        logout,
        refreshProfile,
        getAccessToken,
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
