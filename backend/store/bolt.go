// Package store provides a BoltDB-backed persistence layer.
//
// BoltDB is an embedded key/value store. All data is stored in a single file
// which makes it ideal for demonstration projects – no external database
// process is required.
//
// Idempotency rationale
// ---------------------
// Every write operation (Create, Update) must be safe to call multiple times
// with the same arguments. The store achieves this by:
//   - Create: checking for an existing record *before* inserting. If the key
//     already exists the stored value is returned unchanged and no write is
//     performed.
//   - Update: comparing the incoming payload with the stored data byte-for-byte.
//     If they are identical the write is skipped entirely. This is an important
//     optimisation: unnecessary writes increase disk I/O, can cause cache
//     invalidation in downstream systems, and may trigger spurious event streams
//     in architectures that observe writes (CDC, audit logs, etc.).
package store

import (
	"encoding/json"
	"errors"
	"time"

	bolt "github.com/boltdb/bolt"

	"github.com/arkantrust/idempotency-example/backend/models"
)

const bucketName = "chargebacks"

// ErrNotFound is returned when a requested chargeback does not exist.
var ErrNotFound = errors.New("chargeback not found")

// Store wraps a BoltDB database and exposes CRUD operations for Chargeback
// records. All operations are idempotent by design.
type Store struct {
	db *bolt.DB
}

// New opens (or creates) a BoltDB database at the given path and ensures the
// chargebacks bucket exists.
func New(path string) (*Store, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	// Create the bucket if it does not yet exist. This is idempotent by
	// definition – calling CreateBucketIfNotExists is safe to run on every
	// startup.
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close releases the database file lock.
func (s *Store) Close() error {
	return s.db.Close()
}

// List returns all chargebacks stored in the database.
// This is a pure read – always idempotent.
func (s *Store) List() ([]models.Chargeback, error) {
	var items []models.Chargeback

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		return b.ForEach(func(k, v []byte) error {
			var c models.Chargeback
			if err := json.Unmarshal(v, &c); err != nil {
				return err
			}
			items = append(items, c)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// Return an empty slice rather than nil so the JSON encoder emits [] instead
	// of null. This makes client-side handling simpler.
	if items == nil {
		items = []models.Chargeback{}
	}
	return items, nil
}

// Get retrieves a single chargeback by ID.
// Returns ErrNotFound if the key does not exist.
func (s *Store) Get(id string) (*models.Chargeback, error) {
	var c models.Chargeback

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		v := b.Get([]byte(id))
		if v == nil {
			return ErrNotFound
		}
		return json.Unmarshal(v, &c)
	})
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// Create persists a new chargeback ONLY if one with the same ID does not
// already exist.
//
// Idempotency guarantee: if the record already exists the stored value is
// returned unchanged and no write is performed. This means a client can safely
// retry a failed POST without risking a duplicate record.
//
// Returns (existing, false, nil) when the record already existed.
// Returns (new, true, nil) when the record was successfully created.
func (s *Store) Create(c *models.Chargeback) (*models.Chargeback, bool, error) {
	var result models.Chargeback
	created := false

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		// --- Idempotency check ---
		// If the key already exists we return the stored value and skip the
		// write. This is the core of POST idempotency: the same request ID
		// always returns the same response regardless of retry count.
		existing := b.Get([]byte(c.ID))
		if existing != nil {
			return json.Unmarshal(existing, &result)
		}

		// First-time creation: stamp timestamps and persist.
		now := time.Now().UTC()
		c.CreatedAt = now
		c.UpdatedAt = now

		data, err := json.Marshal(c)
		if err != nil {
			return err
		}

		result = *c
		created = true
		return b.Put([]byte(c.ID), data)
	})
	if err != nil {
		return nil, false, err
	}

	return &result, created, nil
}

// Update persists changes to an existing chargeback ONLY if the payload
// differs from the stored data.
//
// Write-avoidance rationale: unnecessary writes are harmful in real systems
// because they:
//   - Increase I/O pressure on the database
//   - May trigger CDC (Change-Data-Capture) events downstream
//   - Pollute audit logs with no-op changes
//   - Invalidate caches prematurely
//
// By comparing the incoming payload to the stored record before writing we
// make PUT effectively idempotent: retrying with the same data is a no-op.
//
// Returns (updated, true, nil) when a write occurred.
// Returns (existing, false, nil) when the payload was identical (write skipped).
func (s *Store) Update(id string, incoming *models.Chargeback) (*models.Chargeback, bool, error) {
	var result models.Chargeback
	written := false

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		existingBytes := b.Get([]byte(id))
		if existingBytes == nil {
			return ErrNotFound
		}

		var existing models.Chargeback
		if err := json.Unmarshal(existingBytes, &existing); err != nil {
			return err
		}

		// --- Write-avoidance check ---
		// Compare the mutable fields. If nothing changed we skip the write
		// entirely and return the existing record. This is the write-avoidance
		// form of idempotency: the same PUT payload is safe to retry any number
		// of times.
		if existing.Amount == incoming.Amount &&
			existing.Currency == incoming.Currency &&
			existing.Reason == incoming.Reason {
			result = existing
			return nil
		}

		// At least one field changed – apply the update and bump UpdatedAt.
		existing.Amount = incoming.Amount
		existing.Currency = incoming.Currency
		existing.Reason = incoming.Reason
		existing.UpdatedAt = time.Now().UTC()

		data, err := json.Marshal(existing)
		if err != nil {
			return err
		}

		written = true
		result = existing
		return b.Put([]byte(id), data)
	})
	if err != nil {
		return nil, false, err
	}

	return &result, written, nil
}

// Delete removes a chargeback by ID.
//
// Idempotency guarantee: deleting a record that does not exist is not treated
// as an error. This means a client can safely retry a DELETE without receiving
// a spurious 404 on subsequent attempts. In distributed systems the initial
// DELETE may succeed on the server but the client may never receive the
// response – a retry is the only safe recovery strategy, and it must succeed.
func (s *Store) Delete(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		// If the key does not exist bolt.Delete is a no-op, which is exactly
		// the idempotent behaviour we want.
		return b.Delete([]byte(id))
	})
}
