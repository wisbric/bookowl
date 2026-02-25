export interface ClientConfig {
  collab_ws_url?: string
}

let _config: ClientConfig = {}

export async function fetchClientConfig(): Promise<ClientConfig> {
  try {
    const res = await fetch('/api/v1/config/client')
    if (!res.ok) return {}
    _config = await res.json()
  } catch {
    _config = {}
  }
  return _config
}

export function getClientConfig(): ClientConfig {
  return _config
}
