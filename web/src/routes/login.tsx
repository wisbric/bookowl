import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { useAuth } from '@/auth/auth-provider'
import { Loader2 } from 'lucide-react'

export const Route = createFileRoute('/login')({
  component: LoginPage,
})

function LoginPage() {
  const { devMode, isAuthenticated, oidcEnabled, refreshProfile, profile } = useAuth()
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
        await refreshProfile()
        navigate({ to: '/change-password' as string })
        return
      }

      if (!res.ok) {
        setError('Invalid username or password')
        return
      }

      // Login succeeded — session cookie is set.
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
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-3">
          <img src="/owl.png" alt="BookOwl" className="h-16 w-auto" />
          <h1 className="text-2xl font-bold tracking-tight">BookOwl</h1>
          <p className="text-sm text-muted-foreground">Sign in to continue</p>
        </div>

        <div className="rounded-lg border border-border bg-card shadow-sm">
          <div className="border-b border-border px-6 py-4">
            <h2 className="text-center text-lg font-semibold">Sign in</h2>
          </div>
          <div className="space-y-4 p-6">
            {oidcEnabled ? (
              <>
                <button
                  onClick={() => { window.location.href = '/auth/oidc/login' }}
                  className="w-full rounded-md border border-border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
                >
                  Sign in with SSO
                </button>

                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <span className="w-full border-t border-border" />
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
                    <span className="bg-card px-2 text-muted-foreground">or</span>
                  </div>
                </div>
              </>
            ) : (
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">OIDC not configured</span>
                </div>
              </div>
            )}

            <form onSubmit={handleLocalLogin} className="space-y-3">
              <div className="space-y-1">
                <label htmlFor="username" className="text-sm font-medium text-foreground">
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
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:opacity-50"
                />
              </div>

              <div className="space-y-1">
                <label htmlFor="password" className="text-sm font-medium text-foreground">
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
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:opacity-50"
                />
              </div>

              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}

              <button
                type="submit"
                disabled={submitting || rateLimitRetry > 0}
                className="inline-flex h-9 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
              >
                {submitting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : rateLimitRetry > 0 ? (
                  `Try again in ${rateLimitRetry}s`
                ) : (
                  'Sign in'
                )}
              </button>

              <p className="text-center text-xs text-muted-foreground">
                Rate limit: 10 attempts / 15 min
              </p>
            </form>
          </div>
        </div>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          BookOwl — A Wisbric product
        </p>
      </div>
    </div>
  )
}
