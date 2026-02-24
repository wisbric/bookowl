export interface AuthConfig {
  oidc_enabled: boolean
  oidc_authority?: string
  oidc_client_id?: string
}

export async function fetchAuthConfig(): Promise<AuthConfig> {
  const res = await fetch('/api/v1/auth/config')
  if (!res.ok) {
    // If auth config endpoint is unavailable, assume dev mode.
    return { oidc_enabled: false }
  }
  return res.json()
}
