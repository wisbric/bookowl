import { createFileRoute } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api, type ProfileResponse } from '@/api/client'
import { useAuth } from '@/auth/auth-provider'
import { UserAvatar } from '@/components/UserAvatar'
import { Loader2, Check } from 'lucide-react'

export const Route = createFileRoute('/profile')({
  component: ProfilePage,
})

function ProfilePage() {
  const { profile, refreshProfile } = useAuth()
  const [displayName, setDisplayName] = useState('')
  const [timezone, setTimezone] = useState('UTC')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (profile) {
      setDisplayName(profile.display_name || '')
      setTimezone(profile.timezone || 'UTC')
    }
  }, [profile])

  const updateMutation = useMutation({
    mutationFn: (data: { display_name: string; timezone: string }) =>
      api.put<ProfileResponse>('/profile', data),
    onSuccess: async () => {
      await refreshProfile()
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateMutation.mutate({ display_name: displayName, timezone })
  }

  if (!profile) {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-muted-foreground">Loading...</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-6 text-2xl font-bold text-foreground">Your Profile</h1>

      {/* Avatar + info */}
      <div className="mb-8 flex items-center gap-4">
        <UserAvatar profile={profile} size="lg" />
        <div>
          <div className="text-lg font-medium text-foreground">
            {profile.display_name || profile.email}
          </div>
          <div className="text-sm text-muted-foreground">{profile.email}</div>
        </div>
      </div>

      {/* Edit form */}
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label htmlFor="display-name" className="mb-1 block text-sm font-medium text-foreground">
            Display name
          </label>
          <input
            id="display-name"
            type="text"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>

        <div>
          <label htmlFor="timezone" className="mb-1 block text-sm font-medium text-foreground">
            Timezone
          </label>
          <select
            id="timezone"
            value={timezone}
            onChange={(e) => setTimezone(e.target.value)}
            className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          >
            {Intl.supportedValuesOf('timeZone').map((tz) => (
              <option key={tz} value={tz}>
                {tz}
              </option>
            ))}
          </select>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">Role</label>
            <div className="rounded-lg border border-border bg-muted/50 px-3 py-2 text-sm text-muted-foreground">
              {profile.role}
            </div>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">Auth method</label>
            <div className="rounded-lg border border-border bg-muted/50 px-3 py-2 text-sm text-muted-foreground">
              {profile.auth_method === 'local' ? 'Local admin' : profile.auth_method.toUpperCase()}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={updateMutation.isPending}
            className="flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-accent-foreground transition-colors hover:bg-accent/90 disabled:opacity-50"
          >
            {updateMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : saved ? (
              <Check className="h-4 w-4" />
            ) : null}
            {saved ? 'Saved' : 'Save changes'}
          </button>
          {updateMutation.isError && (
            <span className="text-sm text-red-400">Failed to save</span>
          )}
        </div>
      </form>
    </div>
  )
}
