import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { Settings, Plug, Activity, Key, CheckCircle, XCircle, LoaderCircle } from 'lucide-react'
import { api } from '@/api/client'
import type { AdminConfig } from '@/api/client'

export const Route = createFileRoute('/admin')({
  component: AdminPage,
})

function AdminPage() {
  return (
    <div className="mx-auto max-w-3xl px-8 py-8">
      <div className="mb-8 flex items-center gap-3">
        <Settings className="h-6 w-6 text-accent" />
        <h1 className="text-2xl font-bold tracking-tight">Admin</h1>
      </div>

      <div className="space-y-8">
        <NightOwlConfigSection />
        <ApiKeysSection />
        <SystemStatusSection />
      </div>

      <footer className="mt-16 text-center text-sm text-muted-foreground">
        BookOwl v0.1.0 — A Wisbric product
      </footer>
    </div>
  )
}

function NightOwlConfigSection() {
  const queryClient = useQueryClient()

  const { data: config, isLoading } = useQuery({
    queryKey: ['admin-config'],
    queryFn: () => api.get<AdminConfig>('/admin/config'),
  })

  const [url, setUrl] = useState('')
  const [apiKey, setApiKey] = useState('')

  useEffect(() => {
    if (config) {
      setUrl(config.nightowl_api_url || '')
      setApiKey(config.nightowl_api_key || '')
    }
  }, [config])

  const saveMutation = useMutation({
    mutationFn: () =>
      api.put('/admin/config', {
        nightowl_api_url: url,
        nightowl_api_key: apiKey,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-config'] })
    },
  })

  const testMutation = useMutation({
    mutationFn: () =>
      api.post<{ status: string; message: string }>('/admin/config/test-nightowl', {}),
  })

  if (isLoading) {
    return <SectionSkeleton title="NightOwl Integration" />
  }

  return (
    <section className="rounded-lg border border-border bg-card p-6">
      <div className="mb-4 flex items-center gap-2">
        <Plug className="h-5 w-5 text-accent" />
        <h2 className="text-lg font-semibold">NightOwl Integration</h2>
      </div>

      <div className="space-y-4">
        <div>
          <label className="mb-1 block text-sm font-medium">API URL</label>
          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://nightowl.example.com/api/v1"
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium">API Key</label>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="ow_..."
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm font-mono focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => saveMutation.mutate()}
            disabled={saveMutation.isPending}
            className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-foreground hover:bg-accent/90 disabled:opacity-50"
          >
            {saveMutation.isPending ? 'Saving...' : 'Save'}
          </button>

          <button
            onClick={() => testMutation.mutate()}
            disabled={testMutation.isPending}
            className="inline-flex items-center gap-1.5 rounded-md border border-border px-4 py-2 text-sm hover:bg-muted disabled:opacity-50"
          >
            {testMutation.isPending ? (
              <LoaderCircle className="h-4 w-4 animate-spin" />
            ) : (
              <Activity className="h-4 w-4" />
            )}
            Test Connection
          </button>

          {saveMutation.isSuccess && (
            <span className="text-sm text-accent">Saved</span>
          )}
        </div>

        {testMutation.isSuccess && (
          <div className="flex items-center gap-2 rounded-md bg-muted px-3 py-2 text-sm">
            <CheckCircle className="h-4 w-4 text-accent" />
            <span>Connection successful</span>
          </div>
        )}

        {testMutation.isError && (
          <div className="flex items-center gap-2 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
            <XCircle className="h-4 w-4" />
            <span>Connection failed — check URL and API key</span>
          </div>
        )}
      </div>
    </section>
  )
}

function ApiKeysSection() {
  return (
    <section className="rounded-lg border border-border bg-card p-6">
      <div className="mb-4 flex items-center gap-2">
        <Key className="h-5 w-5 text-accent" />
        <h2 className="text-lg font-semibold">API Keys</h2>
      </div>

      <div className="rounded-md bg-muted px-4 py-3 text-sm text-muted-foreground">
        <p>
          API keys are managed via environment variables. Use{' '}
          <code className="rounded bg-background px-1 py-0.5 font-mono text-xs">
            BOOKOWL_API_KEY
          </code>{' '}
          or configure keys in your deployment.
        </p>
        <p className="mt-2">
          Dev key:{' '}
          <code className="rounded bg-background px-1 py-0.5 font-mono text-xs">
            bw_dev_seed_key_do_not_use_in_production
          </code>
        </p>
      </div>
    </section>
  )
}

function SystemStatusSection() {
  const { data, isLoading } = useQuery({
    queryKey: ['system-health'],
    queryFn: () => api.get<{ status: string; checks?: Record<string, string> }>('/health'),
    refetchInterval: 30_000,
  })

  return (
    <section className="rounded-lg border border-border bg-card p-6">
      <div className="mb-4 flex items-center gap-2">
        <Activity className="h-5 w-5 text-accent" />
        <h2 className="text-lg font-semibold">System Status</h2>
      </div>

      {isLoading ? (
        <div className="text-sm text-muted-foreground">Checking health...</div>
      ) : (
        <div className="space-y-2">
          <StatusRow label="API" status={data?.status === 'ok' ? 'healthy' : 'unhealthy'} />
          <StatusRow
            label="Database"
            status={data?.checks?.database || 'unknown'}
          />
          <StatusRow
            label="Redis"
            status={data?.checks?.redis || 'unknown'}
          />
          <StatusRow
            label="NightOwl"
            status={data?.checks?.nightowl || 'unknown'}
          />
        </div>
      )}
    </section>
  )
}

function StatusRow({ label, status }: { label: string; status: string }) {
  const isHealthy = status === 'healthy' || status === 'ok'
  return (
    <div className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2">
      <span className="text-sm">{label}</span>
      <div className="flex items-center gap-1.5">
        <span
          className="inline-block h-2 w-2 rounded-full"
          style={{ backgroundColor: isHealthy ? '#00e5a0' : status === 'unknown' ? '#6B7280' : '#DC2626' }}
        />
        <span className="text-xs text-muted-foreground capitalize">{status}</span>
      </div>
    </div>
  )
}

function SectionSkeleton({ title }: { title: string }) {
  return (
    <section className="rounded-lg border border-border bg-card p-6">
      <h2 className="mb-4 text-lg font-semibold">{title}</h2>
      <div className="text-sm text-muted-foreground">Loading...</div>
    </section>
  )
}
