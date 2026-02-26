/**
 * API client for the idempotency-example backend.
 *
 * All mutations are designed to be called multiple times safely because the
 * backend implements idempotency:
 *   - createChargeback: uses the caller-supplied ID as the idempotency key.
 *   - updateChargeback: the backend skips the write when nothing changed.
 *   - deleteChargeback: the backend succeeds even if the record is already gone.
 */
import type { Chargeback, ChargebackInput } from '@/types/chargeback'

const BASE = '/chargebacks'

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.text()
    throw new Error(body || `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

/** Fetch all chargebacks. */
export async function listChargebacks(): Promise<Chargeback[]> {
  const res = await fetch(BASE)
  return handleResponse<Chargeback[]>(res)
}

/**
 * Create a chargeback using `id` as the idempotency key.
 *
 * If the server already has a record with this ID it returns the existing
 * record without creating a duplicate. This makes retries unconditionally safe.
 */
export async function createChargeback(
  id: string,
  input: ChargebackInput,
): Promise<Chargeback> {
  const res = await fetch(`${BASE}/${id}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return handleResponse<Chargeback>(res)
}

/**
 * Update a chargeback.
 *
 * The backend compares the incoming payload to the stored record and skips
 * the write when they are identical. The response header X-Idempotency-Write
 * indicates whether a write actually occurred ("true"/"false").
 */
export async function updateChargeback(
  id: string,
  input: ChargebackInput,
): Promise<Chargeback> {
  const res = await fetch(`${BASE}/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return handleResponse<Chargeback>(res)
}

/**
 * Delete a chargeback.
 *
 * The backend returns 200 OK even when the record does not exist, so retrying
 * a delete is always safe.
 */
export async function deleteChargeback(id: string): Promise<void> {
  const res = await fetch(`${BASE}/${id}`, { method: 'DELETE' })
  await handleResponse<unknown>(res)
}
