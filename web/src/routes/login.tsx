import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { useAuth } from '@/auth/auth-provider'
import { setSessionMode } from '@/api/client'
import { LogIn, AlertCircle, Loader2 } from 'lucide-react'

export const Route = createFileRoute('/login')({
  component: LoginPage,
})

function LoginPage() {
  const { devMode, isAuthenticated, oidcEnabled, login, refreshProfile, profile } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [rateLimitRetry, setRateLimitRetry] = useState(0)

  useEffect(() => {
    // Redirect away from login if already authenticated via session.
    // In dev mode, only redirect if there's a real session (profile loaded via session cookie).
    if (isAuthenticated && profile && !devMode) {
      navigate({ to: '/' })
    }
  }, [isAuthenticated, profile, devMode, navigate])

  // Rate limit countdown timer.
  useEffect(() => {
    if (rateLimitRetry <= 0) return
    const timer = setInterval(() => {
      setRateLimitRetry((prev) => {
        if (prev <= 1) {
          setError('')
          return 0
        }
        return prev - 1
      })
    }, 1000)
    return () => clearInterval(timer)
  }, [rateLimitRetry])

  const handleLocalLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSubmitting(true)

    try {
      const res = await fetch('/auth/local', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
        credentials: 'same-origin',
      })

      const data = await res.json()

      if (res.status === 429) {
        setRateLimitRetry(data.retry_after || 60)
        setError(`Too many login attempts. Try again in ${data.retry_after || 60}s.`)
        return
      }

      if (res.status === 403 && data.error === 'must_change_password') {
        setSessionMode(true)
        await refreshProfile()
        navigate({ to: '/change-password' as string })
        return
      }

      if (!res.ok) {
        setError('Invalid username or password')
        return
      }

      // Login succeeded — session cookie is set.
      setSessionMode(true)
      await refreshProfile()
      const params = new URLSearchParams(window.location.search)
      const returnTo = params.get('return') || '/'
      navigate({ to: returnTo as string })
    } catch {
      setError('Network error. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-8">
        {/* Header */}
        <div className="text-center">
          <img src="/owl.png" alt="BookOwl" className="mx-auto mb-4 h-12 brightness-0 dark:brightness-100" />
          <h1 className="text-2xl font-bold text-foreground">BookOwl</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Where your operational knowledge lives.
          </p>
        </div>

        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          {/* OIDC button */}
          {oidcEnabled ? (
            <>
              <button
                onClick={() => login()}
                className="flex w-full items-center justify-center gap-2 rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-accent-foreground transition-colors hover:bg-accent/90"
              >
                <LogIn className="h-4 w-4" />
                Sign in with Keycloak
              </button>

              <div className="relative my-6">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t border-border" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">or</span>
                </div>
              </div>
            </>
          ) : (
            <div className="mb-4 rounded-lg border border-border bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
              OIDC not configured. Sign in with the local admin account.
            </div>
          )}

          {/* Local admin form */}
          <form onSubmit={handleLocalLogin} className="space-y-4">
            <div>
              <label htmlFor="username" className="mb-1 block text-sm font-medium text-foreground">
                Username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                autoComplete="username"
                required
                disabled={rateLimitRetry > 0}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:opacity-50"
              />
            </div>

            <div>
              <label htmlFor="password" className="mb-1 block text-sm font-medium text-foreground">
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                required
                disabled={rateLimitRetry > 0}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:opacity-50"
              />
            </div>

            {error && (
              <div className="flex items-start gap-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
                <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <button
              type="submit"
              disabled={submitting || rateLimitRetry > 0}
              className="flex w-full items-center justify-center gap-2 rounded-lg bg-foreground px-4 py-2.5 text-sm font-medium text-background transition-colors hover:bg-foreground/90 disabled:opacity-50"
            >
              {submitting ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : rateLimitRetry > 0 ? (
                `Try again in ${rateLimitRetry}s`
              ) : (
                'Sign in'
              )}
            </button>
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-muted-foreground">
          A Wisbric product
        </p>
      </div>
    </div>
  )
}
