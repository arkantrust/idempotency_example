// Package models defines the core domain types for the idempotency example.
package models

import "time"

// Chargeback represents a financial dispute record.
//
// The ID field doubles as the idempotency key: every operation that references
// the same ID is guaranteed to produce the same outcome regardless of how many
// times it is executed. This is critical in distributed systems where retries
// due to network failures or timeouts are common – without an idempotency key
// a client retry could create duplicate chargebacks and result in double-charges.
type Chargeback struct {
	// ID is the unique identifier and the idempotency key used in POST
	// /chargebacks/{id}. Clients should generate this value (e.g. a UUID)
	// before sending the request so that retries always reference the same key.
	ID string `json:"id"`

	// Amount is the disputed amount expressed in the smallest currency unit
	// (e.g. cents for USD). Using integer arithmetic avoids floating-point
	// rounding issues that matter in financial systems.
	Amount int64 `json:"amount"`

	// Currency is the ISO 4217 three-letter currency code (e.g. "USD", "EUR").
	Currency string `json:"currency"`

	// Reason describes why the chargeback was raised.
	Reason string `json:"reason"`

	// CreatedAt is the UTC timestamp of the first write.
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is the UTC timestamp of the most recent write.
	// For idempotent POSTs this stays equal to CreatedAt because the record is
	// never mutated after creation.
	UpdatedAt time.Time `json:"updatedAt"`
}
