import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, type PersonalToken, type PersonalTokenCreated } from '@/api/client'
import { Key, Plus, Trash2, Copy, Check, Loader2, AlertCircle } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'

export const Route = createFileRoute('/profile/tokens')({
  component: PersonalTokensPage,
})

function PersonalTokensPage() {
  const queryClient = useQueryClient()
  const [showCreate, setShowCreate] = useState(false)
  const [createdToken, setCreatedToken] = useState<PersonalTokenCreated | null>(null)

  const { data: tokens, isLoading } = useQuery({
    queryKey: ['personal-tokens'],
    queryFn: () => api.get<PersonalToken[]>('/profile/tokens'),
  })

  const revokeMutation = useMutation({
    mutationFn: (id: string) => api.del(`/profile/tokens/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['personal-tokens'] })
    },
  })

  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Personal Access Tokens</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Use personal tokens to authenticate API calls from scripts, curl, or other tools.
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 rounded-lg bg-accent px-3 py-2 text-sm font-medium text-accent-foreground transition-colors hover:bg-accent/90"
        >
          <Plus className="h-4 w-4" />
          New token
        </button>
      </div>

      {/* Created token banner (one-time reveal) */}
      {createdToken && (
        <TokenRevealBanner token={createdToken} onDismiss={() => setCreatedToken(null)} />
      )}

      {/* Token list */}
      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : !tokens?.length ? (
        <div className="rounded-xl border border-border bg-card px-6 py-12 text-center">
          <Key className="mx-auto mb-3 h-10 w-10 text-muted-foreground/50" />
          <p className="text-muted-foreground">No personal tokens yet.</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Create one to authenticate API calls.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {tokens.map((token) => (
            <div
              key={token.id}
              className="rounded-xl border border-border bg-card px-4 py-3"
            >
              <div className="flex items-start justify-between">
                <div>
                  <div className="font-medium text-foreground">{token.name}</div>
                  <div className="mt-0.5 text-xs text-muted-foreground">
                    Created{' '}
                    {formatDistanceToNow(new Date(token.created_at), { addSuffix: true })}
                    {token.last_used_at && (
                      <>
                        {' · Last used '}
                        {formatDistanceToNow(new Date(token.last_used_at), { addSuffix: true })}
                      </>
                    )}
                    {!token.last_used_at && ' · Never used'}
                  </div>
                  <div className="mt-0.5 text-xs text-muted-foreground">
                    {token.expires_at
                      ? `Expires ${formatDistanceToNow(new Date(token.expires_at), { addSuffix: true })}`
                      : 'Expires never'}
                  </div>
                </div>
                <button
                  onClick={() => {
                    if (confirm('Revoke this token? This cannot be undone.')) {
                      revokeMutation.mutate(token.id)
                    }
                  }}
                  disabled={revokeMutation.isPending}
                  className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-red-500/10 hover:text-red-400"
                  title="Revoke token"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create token dialog */}
      {showCreate && (
        <CreateTokenDialog
          onClose={() => setShowCreate(false)}
          onCreated={(t) => {
            setCreatedToken(t)
            setShowCreate(false)
            queryClient.invalidateQueries({ queryKey: ['personal-tokens'] })
          }}
        />
      )}
    </div>
  )
}

function TokenRevealBanner({
  token,
  onDismiss,
}: {
  token: PersonalTokenCreated
  onDismiss: () => void
}) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    await navigator.clipboard.writeText(token.plaintext)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="mb-6 rounded-xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3">
      <div className="mb-2 flex items-center gap-2 text-sm font-medium text-emerald-400">
        <Check className="h-4 w-4" />
        Token created
      </div>
      <div className="flex items-center gap-2">
        <code className="flex-1 rounded border border-border bg-background px-3 py-2 font-mono text-xs text-foreground">
          {token.plaintext}
        </code>
        <button
          onClick={handleCopy}
          className="rounded-md border border-border p-2 text-muted-foreground transition-colors hover:bg-muted"
        >
          {copied ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4" />}
        </button>
      </div>
      <div className="mt-2 flex items-center justify-between">
        <p className="text-xs text-muted-foreground">
          This token will not be shown again. Store it somewhere safe.
        </p>
        <button onClick={onDismiss} className="text-xs text-muted-foreground hover:text-foreground">
          Done
        </button>
      </div>
    </div>
  )
}

function CreateTokenDialog({
  onClose,
  onCreated,
}: {
  onClose: () => void
  onCreated: (t: PersonalTokenCreated) => void
}) {
  const [name, setName] = useState('')
  const [expiresIn, setExpiresIn] = useState<string>('30d')
  const [error, setError] = useState('')

  const createMutation = useMutation({
    mutationFn: (data: { name: string; expires_in?: string }) =>
      api.post<PersonalTokenCreated>('/profile/tokens', data),
    onSuccess: (token) => {
      onCreated(token)
    },
    onError: () => {
      setError('Failed to create token')
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    setError('')
    createMutation.mutate({
      name: name.trim(),
      expires_in: expiresIn === 'never' ? undefined : expiresIn,
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-lg font-semibold text-foreground">New Personal Access Token</h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="token-name" className="mb-1 block text-sm font-medium text-foreground">
              Token name
            </label>
            <input
              id="token-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. runbook-sync-script"
              required
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>

          <div>
            <label className="mb-2 block text-sm font-medium text-foreground">Expiration</label>
            <div className="space-y-2">
              {[
                { value: 'never', label: 'Never' },
                { value: '30d', label: '30 days' },
                { value: '90d', label: '90 days' },
                { value: '1y', label: '1 year' },
              ].map((opt) => (
                <label key={opt.value} className="flex items-center gap-2 text-sm text-foreground">
                  <input
                    type="radio"
                    name="expires_in"
                    value={opt.value}
                    checked={expiresIn === opt.value}
                    onChange={(e) => setExpiresIn(e.target.value)}
                    className="accent-accent"
                  />
                  {opt.label}
                </label>
              ))}
            </div>
          </div>

          {error && (
            <div className="flex items-center gap-2 text-sm text-red-400">
              <AlertCircle className="h-4 w-4" />
              {error}
            </div>
          )}

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-muted"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!name.trim() || createMutation.isPending}
              className="flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-accent-foreground transition-colors hover:bg-accent/90 disabled:opacity-50"
            >
              {createMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              Create token
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
