/**
 * React Query client configuration.
 *
 * Retry logic: when a mutation fails React Query will retry it up to 3 times
 * with exponential back-off. A toast notification is shown before each retry
 * so the user can see the idempotency machinery in action: the same request
 * is re-sent multiple times and the backend handles it correctly every time.
 */
import { QueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Retry failed queries once after 1 s.
      retry: 1,
      retryDelay: 1000,
      staleTime: 5_000,
    },
    mutations: {
      // Retry failed mutations up to 3 times with exponential back-off.
      // Because the backend is idempotent each retry is safe.
      retry: 3,
      retryDelay: (attempt) => {
        const delay = Math.min(1000 * 2 ** attempt, 10_000)
        const seconds = Math.round(delay / 1000)
        // Show a toast so the user can observe the retry behaviour.
        toast.warning(`Request failed. Retrying in ${seconds} second${seconds === 1 ? '' : 's'}…`)
        return delay
      },
    },
  },
})
