import { UserManager, WebStorageStateStore } from 'oidc-client-ts'
import type { AuthConfig } from './auth-config'

export function createUserManager(config: AuthConfig): UserManager | null {
  if (!config.oidc_enabled || !config.oidc_authority || !config.oidc_client_id) {
    return null
  }

  return new UserManager({
    authority: config.oidc_authority,
    client_id: config.oidc_client_id,
    redirect_uri: `${window.location.origin}/callback`,
    post_logout_redirect_uri: window.location.origin,
    response_type: 'code',
    scope: 'openid profile email',
    userStore: new WebStorageStateStore({ store: window.localStorage }),
    automaticSilentRenew: true,
  })
}
