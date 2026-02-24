import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'

export const Route = createFileRoute('/callback')({
  component: CallbackPage,
})

function CallbackPage() {
  const navigate = useNavigate()

  useEffect(() => {
    // The OIDC callback is handled by the UserManager in main.tsx before
    // the app renders. If we reach this component, the callback has already
    // been processed — just redirect home.
    navigate({ to: '/' })
  }, [navigate])

  return (
    <div className="flex h-screen items-center justify-center bg-background">
      <p className="text-muted-foreground">Signing in...</p>
    </div>
  )
}
