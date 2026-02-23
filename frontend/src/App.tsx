/**
 * App is the root component.
 *
 * It sets up the React Query provider, the Sonner toast container, and renders
 * the main dashboard layout.
 *
 * Architecture overview:
 * - React Query manages all server state (fetching, caching, retrying).
 * - Sonner provides toast notifications.
 * - The backend at /chargebacks exposes idempotent REST endpoints.
 * - All mutations use the retry logic configured in src/lib/queryClient.ts:
 *   up to 3 retries with exponential back-off and a toast on each retry.
 */
import { QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { queryClient } from '@/lib/queryClient'
import { ChargebackTable } from '@/components/ChargebackTable'
import { AddChargebackDialog } from '@/components/AddChargebackDialog'

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      {/* Toaster renders toast notifications. It is placed at the root so it
          is available from any component in the tree. */}
      <Toaster position="top-right" richColors />

      <div className="min-h-screen bg-slate-50">
        <header className="bg-white border-b border-slate-200 px-6 py-4">
          <div className="max-w-6xl mx-auto flex items-center justify-between">
            <div>
              <h1 className="text-xl font-bold text-slate-900">Idempotency Example</h1>
              <p className="text-sm text-slate-500 mt-0.5">
                Demonstrating safe retries with idempotent REST APIs
              </p>
            </div>
            <AddChargebackDialog />
          </div>
        </header>

        <main className="max-w-6xl mx-auto px-6 py-8">
          {/* Info banner */}
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6 text-sm text-blue-800">
            <strong>How idempotency works here:</strong>
            <ul className="list-disc list-inside mt-1 space-y-1">
              <li>
                <strong>POST /chargebacks/&#123;id&#125;</strong> – uses the ID as an idempotency
                key. Re-submitting with the same ID returns the existing record, no duplicate is
                created.
              </li>
              <li>
                <strong>PUT /chargebacks/&#123;id&#125;</strong> – compares payload to stored data
                and skips the write when nothing changed (write-avoidance).
              </li>
              <li>
                <strong>DELETE /chargebacks/&#123;id&#125;</strong> – safe to call multiple times;
                returns 200 OK even when the record is already gone.
              </li>
              <li>
                <strong>Retry logic</strong> – failed requests are retried up to 3 times. A toast
                notification appears before each retry.
              </li>
            </ul>
          </div>

          <ChargebackTable />
        </main>
      </div>
    </QueryClientProvider>
  )
}
