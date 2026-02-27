import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useAuth } from '@/auth/auth-provider'

export const Route = createFileRoute('/callback')({
  component: CallbackPage,
})

function CallbackPage() {
  const { login } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    // After OIDC callback, the session cookie is already set by the server.
    // Validate it by calling /auth/me, then redirect to /.
    fetch('/auth/me', { credentials: 'same-origin' })
      .then((res) => {
        if (!res.ok) throw new Error('no session')
        return res.json()
      })
      .then((user) => {
        login(user)
        navigate({ to: '/' })
      })
      .catch(() => {
        navigate({ to: '/login' })
      })
  }, [login, navigate])

  return (
    <div className="flex h-screen items-center justify-center bg-background">
      <p className="text-muted-foreground">Completing sign in...</p>
    </div>
  )
}
