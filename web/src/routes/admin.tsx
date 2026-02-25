import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { Settings, Plug, Activity, Key, CheckCircle, XCircle, LoaderCircle, Shield, ChevronDown, ChevronRight, Copy, Eye, EyeOff, AlertTriangle } from 'lucide-react'
import { api } from '@/api/client'
import type { AdminConfig, OIDCConfig, OIDCTestResponse, HealthStatus } from '@/api/client'

export const Route = createFileRoute('/admin')({
  component: AdminPage,
})

function AdminPage() {
  const [activeTab, setActiveTab] = useState<'nightowl' | 'auth' | 'apikeys' | 'system'>('nightowl')

  const tabs = [
    { id: 'nightowl' as const, label: 'NightOwl' },
    { id: 'auth' as const, label: 'Authentication' },
    { id: 'apikeys' as const, label: 'API Keys' },
    { id: 'system' as const, label: 'System' },
  ]

  return (
    <div className="max-w-[860px] px-8 py-8">
      <div className="mb-8 flex items-center gap-3">
        <Settings className="h-6 w-6 text-accent" />
        <h1 className="text-2xl font-bold tracking-tight">Admin</h1>
      </div>

      <div className="mb-6 flex gap-1 rounded-lg border border-border bg-muted/50 p-1">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex-1 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? 'bg-card text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="space-y-8">
        {activeTab === 'nightowl' && <NightOwlConfigSection />}
        {activeTab === 'auth' && <AuthenticationSection />}
        {activeTab === 'apikeys' && <ApiKeysSection />}
        {activeTab === 'system' && <SystemStatusSection />}
      </div>

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

function AuthenticationSection() {
  const queryClient = useQueryClient()

  const { data: config, isLoading } = useQuery({
    queryKey: ['admin-oidc-config'],
    queryFn: () => api.get<OIDCConfig>('/admin/config/oidc'),
  })

  const [issuerUrl, setIssuerUrl] = useState('')
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [adminGroups, setAdminGroups] = useState('')
  const [editorGroups, setEditorGroups] = useState('')
  const [allowedGroups, setAllowedGroups] = useState('')
  const [enabled, setEnabled] = useState(false)
  const [showSecret, setShowSecret] = useState(false)
  const [guideOpen, setGuideOpen] = useState(false)
  const [testResult, setTestResult] = useState<OIDCTestResponse | null>(null)

  useEffect(() => {
    if (config) {
      setIssuerUrl(config.issuer_url || '')
      setClientId(config.client_id || '')
      setClientSecret(config.client_secret || '')
      setAdminGroups((config.admin_groups || []).join(', '))
      setEditorGroups((config.editor_groups || []).join(', '))
      setAllowedGroups((config.allowed_groups || []).join(', '))
      setEnabled(config.enabled)
    }
  }, [config])

  const parseGroups = (s: string): string[] =>
    s.split(',').map((g) => g.trim()).filter(Boolean)

  const saveMutation = useMutation({
    mutationFn: () => {
      const body: Record<string, unknown> = {
        issuer_url: issuerUrl,
        client_id: clientId,
        admin_groups: parseGroups(adminGroups),
        editor_groups: parseGroups(editorGroups),
        allowed_groups: parseGroups(allowedGroups),
        enabled,
      }
      // Only send client_secret if it was changed from the masked value.
      if (clientSecret && !clientSecret.startsWith('bw_****')) {
        body.client_secret = clientSecret
      }
      return api.put<OIDCConfig>('/admin/config/oidc', body)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-oidc-config'] })
    },
  })

  const testMutation = useMutation({
    mutationFn: () => {
      const body: Record<string, string> = {
        issuer_url: issuerUrl,
        client_id: clientId,
      }
      if (clientSecret && !clientSecret.startsWith('bw_****')) {
        body.client_secret = clientSecret
      }
      return api.post<OIDCTestResponse>('/admin/config/oidc/test', body)
    },
    onSuccess: (data) => {
      setTestResult(data)
    },
  })

  const redirectUri = typeof window !== 'undefined'
    ? `${window.location.origin}/callback`
    : ''

  const copyRedirectUri = () => {
    navigator.clipboard.writeText(redirectUri)
  }

  if (isLoading) {
    return <SectionSkeleton title="Authentication" />
  }

  return (
    <section className="rounded-lg border border-border bg-card p-6">
      <div className="mb-4 flex items-center gap-2">
        <Shield className="h-5 w-5 text-accent" />
        <h2 className="text-lg font-semibold">Authentication — Keycloak / OIDC</h2>
      </div>

      <div className="space-y-4">
        <div>
          <label className="mb-1 block text-sm font-medium">Issuer URL</label>
          <input
            type="text"
            value={issuerUrl}
            onChange={(e) => setIssuerUrl(e.target.value)}
            placeholder="https://keycloak.example.com/realms/wisbric"
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium">Client ID</label>
            <input
              type="text"
              value={clientId}
              onChange={(e) => setClientId(e.target.value)}
              placeholder="bookowl"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">Client Secret</label>
            <div className="flex gap-2">
              <input
                type={showSecret ? 'text' : 'password'}
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
                placeholder="••••••••••"
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm font-mono focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
              <button
                type="button"
                onClick={() => setShowSecret(!showSecret)}
                className="rounded-md border border-border px-2 text-muted-foreground hover:text-foreground"
                title={showSecret ? 'Hide' : 'Reveal'}
              >
                {showSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium">Redirect URI (read-only)</label>
          <div className="flex gap-2">
            <input
              type="text"
              value={redirectUri}
              readOnly
              className="w-full rounded-md border border-border bg-muted px-3 py-2 text-sm text-muted-foreground"
            />
            <button
              type="button"
              onClick={copyRedirectUri}
              className="rounded-md border border-border px-2 text-muted-foreground hover:text-foreground"
              title="Copy"
            >
              <Copy className="h-4 w-4" />
            </button>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            Copy this URL into your Keycloak client config.
          </p>
        </div>

        <div className="border-t border-border pt-4">
          <h3 className="mb-3 text-sm font-semibold">Role Mapping</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-sm font-medium">Admin groups</label>
              <input
                type="text"
                value={adminGroups}
                onChange={(e) => setAdminGroups(e.target.value)}
                placeholder="bookowl-admins"
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Editor groups</label>
              <input
                type="text"
                value={editorGroups}
                onChange={(e) => setEditorGroups(e.target.value)}
                placeholder="bookowl-editors"
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
            </div>
          </div>
          <div className="mt-4">
            <label className="mb-1 block text-sm font-medium">
              Allowed groups (leave empty to allow all)
            </label>
            <input
              type="text"
              value={allowedGroups}
              onChange={(e) => setAllowedGroups(e.target.value)}
              placeholder="bookowl-users"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            Comma-separated Keycloak group names
          </p>
        </div>

        <div className="flex items-center gap-3 border-t border-border pt-4">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="h-4 w-4 rounded border-border accent-accent"
            />
            Enable OIDC authentication
          </label>
        </div>

        <div className="flex items-center gap-3 border-t border-border pt-4">
          <button
            onClick={() => testMutation.mutate()}
            disabled={testMutation.isPending || !issuerUrl}
            className="inline-flex items-center gap-1.5 rounded-md border border-border px-4 py-2 text-sm hover:bg-muted disabled:opacity-50"
          >
            {testMutation.isPending ? (
              <LoaderCircle className="h-4 w-4 animate-spin" />
            ) : (
              <Activity className="h-4 w-4" />
            )}
            Test Connection
          </button>

          <button
            onClick={() => saveMutation.mutate()}
            disabled={saveMutation.isPending}
            className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-foreground hover:bg-accent/90 disabled:opacity-50"
          >
            {saveMutation.isPending ? 'Saving...' : 'Save Settings'}
          </button>

          {saveMutation.isSuccess && (
            <span className="text-sm text-accent">Saved</span>
          )}
        </div>

        {saveMutation.isError && (
          <div className="flex items-center gap-2 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
            <XCircle className="h-4 w-4" />
            <span>Failed to save settings</span>
          </div>
        )}

        {testMutation.isPending && (
          <div className="flex items-center gap-2 rounded-md bg-muted px-3 py-2 text-sm text-muted-foreground">
            <LoaderCircle className="h-4 w-4 animate-spin" />
            <span>Testing connection...</span>
          </div>
        )}

        {testResult && <OIDCTestResult result={testResult} />}

        {config?.last_tested_at && !testResult && (
          <div className="text-xs text-muted-foreground">
            Last tested: {new Date(config.last_tested_at).toLocaleString()} — {config.last_test_result}
          </div>
        )}

        <div className="border-t border-border pt-4">
          <button
            onClick={() => setGuideOpen(!guideOpen)}
            className="flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground"
          >
            {guideOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            Keycloak Setup Guide
          </button>
          {guideOpen && (
            <div className="mt-3 rounded-md bg-muted px-4 py-3 text-sm text-muted-foreground">
              <ol className="list-inside list-decimal space-y-2">
                <li>In your Keycloak realm, go to <strong>Clients &rarr; Create client</strong></li>
                <li>Client ID: <code className="rounded bg-background px-1 py-0.5 font-mono text-xs">{clientId || 'bookowl'}</code></li>
                <li>Client authentication: <strong>ON</strong> (confidential client)</li>
                <li>Valid redirect URIs: <code className="rounded bg-background px-1 py-0.5 font-mono text-xs">{redirectUri}</code></li>
                <li>
                  Add client scope: Add a mapper of type <strong>Group Membership</strong>
                  <br />
                  Token Claim Name: <code className="rounded bg-background px-1 py-0.5 font-mono text-xs">groups</code>
                  <br />
                  Full group path: <strong>OFF</strong>
                </li>
                <li>Copy the client secret from the <strong>Credentials</strong> tab and paste it above.</li>
              </ol>
            </div>
          )}
        </div>
      </div>
    </section>
  )
}

function OIDCTestResult({ result }: { result: OIDCTestResponse }) {
  const { ok, latency_ms, error, details } = result

  return (
    <div className={`rounded-md px-4 py-3 text-sm ${
      ok ? 'bg-accent/10' : details.discovery_ok ? 'bg-yellow-500/10' : 'bg-destructive/10'
    }`}>
      <div className="flex items-center gap-2 font-medium">
        {ok ? (
          <>
            <CheckCircle className="h-4 w-4 text-accent" />
            <span>Connected &middot; {latency_ms}ms</span>
          </>
        ) : details.discovery_ok ? (
          <>
            <AlertTriangle className="h-4 w-4 text-yellow-500" />
            <span>Partially connected &mdash; {error}</span>
          </>
        ) : (
          <>
            <XCircle className="h-4 w-4 text-destructive" />
            <span>Connection failed &mdash; {error}</span>
          </>
        )}
      </div>
      <div className="mt-2 space-y-1 pl-6 text-xs">
        <TestCheckRow ok={details.discovery_ok} label="Discovery document" skipped={false} />
        <TestCheckRow ok={details.jwks_ok} label="JWKS endpoint" skipped={!details.discovery_ok} />
        <TestCheckRow
          ok={details.client_credentials_ok}
          label="Client credentials"
          skipped={!details.discovery_ok}
        />
        {details.discovery_ok && (
          details.has_groups_scope ? (
            <TestCheckRow ok={true} label="groups scope available" skipped={false} />
          ) : (
            <div className="flex items-center gap-1.5 text-yellow-500">
              <AlertTriangle className="h-3 w-3" />
              <span>groups scope not advertised — role mapping may not work</span>
            </div>
          )
        )}
      </div>
    </div>
  )
}

function TestCheckRow({ ok, label, skipped }: { ok: boolean; label: string; skipped: boolean }) {
  if (skipped) {
    return (
      <div className="flex items-center gap-1.5 text-muted-foreground">
        <span className="inline-block h-3 w-3 text-center">&ndash;</span>
        <span>{label}: not checked</span>
      </div>
    )
  }
  return (
    <div className={`flex items-center gap-1.5 ${ok ? 'text-accent' : 'text-destructive'}`}>
      {ok ? <CheckCircle className="h-3 w-3" /> : <XCircle className="h-3 w-3" />}
      <span>{label}</span>
    </div>
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
    queryFn: () => api.get<HealthStatus>('/health'),
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
  const isNeutral = status === 'unknown' || status === 'not configured'
  return (
    <div className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2">
      <span className="text-sm">{label}</span>
      <div className="flex items-center gap-1.5">
        <span
          className="inline-block h-2 w-2 rounded-full"
          style={{ backgroundColor: isHealthy ? '#00e5a0' : isNeutral ? '#6B7280' : '#DC2626' }}
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
