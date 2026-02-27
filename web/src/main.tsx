import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { router } from './router'
import { AuthProvider } from './auth/auth-provider'
import { fetchAuthConfig } from './auth/auth-config'
import { fetchClientConfig } from './config/client-config'
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
  const [config] = await Promise.all([fetchAuthConfig(), fetchClientConfig()])

  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <AuthProvider oidcEnabled={config.oidc_enabled}>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </AuthProvider>
    </StrictMode>,
  )
}

bootstrap()
