import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { router } from './router'
import { AuthProvider } from './auth/auth-provider'
import { fetchAuthConfig } from './auth/auth-config'
import { createUserManager } from './auth/oidc-manager'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

async function bootstrap() {
  const config = await fetchAuthConfig()
  const userManager = createUserManager(config)

  // Handle OIDC callback before rendering the app.
  if (userManager && window.location.pathname === '/callback') {
    try {
      await userManager.signinRedirectCallback()
      window.history.replaceState({}, '', '/')
    } catch (err) {
      console.error('OIDC callback failed:', err)
      window.history.replaceState({}, '', '/')
    }
  }

  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <AuthProvider userManager={userManager} oidcEnabled={config.oidc_enabled}>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </AuthProvider>
    </StrictMode>,
  )
}

bootstrap()
