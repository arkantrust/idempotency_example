// Package handlers provides HTTP handler functions for the chargeback REST API.
//
// Every handler is designed to be idempotent:
//
//   - GET  /chargebacks      – pure read, trivially idempotent.
//   - POST /chargebacks/{id} – returns the existing record without writing if
//     the ID already exists.
//   - PUT  /chargebacks/{id} – skips the write when the incoming payload is
//     identical to the stored data (write-avoidance idempotency).
//   - DELETE /chargebacks/{id} – succeeds even when the record does not exist.
//
// Why does idempotency matter?
// In any networked system a request may fail *after* the server has processed
// it but *before* the client receives the response (e.g. a TCP reset, a load-
// balancer timeout, a mobile client losing connectivity). The only safe
// recovery strategy for the client is to retry. If the API is not idempotent
// those retries can produce duplicate records, double charges, or corrupt state.
// Making every mutating endpoint idempotent makes retries unconditionally safe.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/arkantrust/idempotency-example/backend/models"
	"github.com/arkantrust/idempotency-example/backend/store"
)

// Handler holds the dependencies for all chargeback HTTP handlers.
type Handler struct {
	store *store.Store
}

// New creates a new Handler with the given store.
func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

// writeJSON serialises v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ServeHTTP routes requests to the appropriate sub-handler based on the HTTP
// method. The mux in main.go maps this handler to /chargebacks/{id} and
// /chargebacks patterns.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	case http.MethodPut:
		h.update(w, r)
	case http.MethodDelete:
		h.delete(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// list handles GET /chargebacks.
// Returns all chargebacks as a JSON array. Pure read – always safe to retry.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list chargebacks")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// create handles POST /chargebacks/{id}.
//
// The {id} path parameter is the idempotency key. The server uses it to detect
// duplicate requests:
//   - First call  → creates the record, returns 201 Created.
//   - Retry calls → returns the SAME record, returns 200 OK (no write).
//
// This pattern is used by payment processors like Stripe and Adyen to make
// charge operations safe to retry without risk of double-charging.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id in path")
		return
	}

	var body models.Chargeback
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.ID = id

	result, created, err := h.store.Create(&body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create chargeback")
		return
	}

	if created {
		// New record – return 201 Created.
		writeJSON(w, http.StatusCreated, result)
	} else {
		// Duplicate request detected – return existing record with 200 OK.
		// The client receives the same data it would have received on the first
		// call, making the overall operation transparent to retry logic.
		writeJSON(w, http.StatusOK, result)
	}
}

// update handles PUT /chargebacks/{id}.
//
// Write-avoidance: the handler compares the incoming payload with the stored
// record. If they are identical no write is performed and 200 OK is returned
// with the existing record. This means:
//   - Clients can retry a PUT without causing unnecessary database writes.
//   - Downstream systems (audit logs, CDC streams) are not polluted with
//     no-op changes.
//   - The response is always deterministic for the same input.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id in path")
		return
	}

	var body models.Chargeback
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, written, err := h.store.Update(id, &body)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "chargeback not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update chargeback")
		return
	}

	// Return a custom header so the caller can observe whether a write
	// actually occurred. This is useful for debugging and demonstrates the
	// write-avoidance optimisation in action.
	if written {
		w.Header().Set("X-Idempotency-Write", "true")
	} else {
		w.Header().Set("X-Idempotency-Write", "false")
	}

	writeJSON(w, http.StatusOK, result)
}

// delete handles DELETE /chargebacks/{id}.
//
// Idempotent delete: if the resource does not exist the handler still returns
// 200 OK. This is intentional – the desired end state (record does not exist)
// is already achieved regardless of whether a delete was actually performed.
//
// Alternative design: some APIs return 204 No Content for both cases. We
// return 200 OK with a small JSON body so the client can log a meaningful
// message without special-casing status codes.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id in path")
		return
	}

	if err := h.store.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete chargeback")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}
