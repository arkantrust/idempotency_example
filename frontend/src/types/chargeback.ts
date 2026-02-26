/** Chargeback represents a financial dispute record returned by the API. */
export interface Chargeback {
  id: string
  /** Amount in smallest currency unit (e.g. cents). */
  amount: number
  /** ISO 4217 currency code (e.g. "USD"). */
  currency: string
  reason: string
  createdAt: string
  updatedAt: string
}

/** ChargebackInput is the payload sent when creating or updating a chargeback. */
export interface ChargebackInput {
  amount: number
  currency: string
  reason: string
}
