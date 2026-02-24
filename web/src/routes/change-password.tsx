import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { AlertCircle, Check, Loader2 } from 'lucide-react'

export const Route = createFileRoute('/change-password')({
  component: ChangePasswordPage,
})

function ChangePasswordPage() {
  const navigate = useNavigate()
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const hasLength = newPassword.length >= 12
  const hasCase = /[a-z]/.test(newPassword) && /[A-Z]/.test(newPassword)
  const hasDigitOrSymbol = /[0-9]/.test(newPassword) || /[^a-zA-Z0-9]/.test(newPassword)
  const passwordsMatch = newPassword === confirmPassword && confirmPassword.length > 0
  const isValid = hasLength && hasCase && hasDigitOrSymbol && passwordsMatch

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!isValid) return
    setError('')
    setSubmitting(true)

    try {
      const res = await fetch('/auth/change-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_password: newPassword }),
        credentials: 'same-origin',
      })

      if (!res.ok) {
        const data = await res.json().catch(() => ({ error: 'Failed to change password' }))
        setError(data.error || data.message || 'Failed to change password')
        return
      }

      navigate({ to: '/' })
    } catch {
      setError('Network error. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-8">
        <div className="text-center">
          <img src="/owl.png" alt="BookOwl" className="mx-auto mb-4 h-12 brightness-0 dark:brightness-100" />
          <h1 className="text-2xl font-bold text-foreground">Change your password</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            You're using the default admin password. Please set a new password before continuing.
          </p>
        </div>

        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="new-password" className="mb-1 block text-sm font-medium text-foreground">
                New password
              </label>
              <input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                autoComplete="new-password"
                required
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
            </div>

            <div>
              <label htmlFor="confirm-password" className="mb-1 block text-sm font-medium text-foreground">
                Confirm new password
              </label>
              <input
                id="confirm-password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                autoComplete="new-password"
                required
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
            </div>

            {/* Password requirements */}
            <div className="space-y-1 rounded-lg border border-border bg-muted/50 px-3 py-2.5 text-sm">
              <p className="mb-1.5 font-medium text-muted-foreground">Password requirements:</p>
              <Requirement met={hasLength}>At least 12 characters</Requirement>
              <Requirement met={hasCase}>Contains uppercase and lowercase</Requirement>
              <Requirement met={hasDigitOrSymbol}>Contains a number or symbol</Requirement>
              <Requirement met={passwordsMatch}>Passwords match</Requirement>
            </div>

            {error && (
              <div className="flex items-start gap-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
                <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <button
              type="submit"
              disabled={!isValid || submitting}
              className="flex w-full items-center justify-center gap-2 rounded-lg bg-foreground px-4 py-2.5 text-sm font-medium text-background transition-colors hover:bg-foreground/90 disabled:opacity-50"
            >
              {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Set password'}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}

function Requirement({ met, children }: { met: boolean; children: React.ReactNode }) {
  return (
    <div className={`flex items-center gap-2 ${met ? 'text-emerald-400' : 'text-muted-foreground'}`}>
      <Check className={`h-3.5 w-3.5 ${met ? 'opacity-100' : 'opacity-30'}`} />
      <span>{children}</span>
    </div>
  )
}
