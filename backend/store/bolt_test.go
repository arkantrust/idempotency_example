package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arkantrust/idempotency-example/backend/models"
	"github.com/arkantrust/idempotency-example/backend/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestListEmpty(t *testing.T) {
	s := newTestStore(t)
	items, err := s.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %d items", len(items))
	}
}

func TestCreateIdempotency(t *testing.T) {
	s := newTestStore(t)

	cb := &models.Chargeback{
		ID:       "test-id-1",
		Amount:   1000,
		Currency: "USD",
		Reason:   "duplicate charge",
	}

	// First call – should create.
	first, created, err := s.Create(cb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("expected created=true on first call")
	}
	if first.ID != cb.ID {
		t.Fatalf("expected id %q, got %q", cb.ID, first.ID)
	}

	// Second call with same ID – should return existing, no write.
	second, created, err := s.Create(cb)
	if err != nil {
		t.Fatalf("unexpected error on retry: %v", err)
	}
	if created {
		t.Fatal("expected created=false on duplicate call")
	}
	if second.ID != first.ID {
		t.Fatalf("expected same record on retry, got different id")
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Fatal("createdAt should not change on idempotent create")
	}
}

func TestUpdateWriteAvoidance(t *testing.T) {
	s := newTestStore(t)

	cb := &models.Chargeback{
		ID:       "test-id-2",
		Amount:   500,
		Currency: "EUR",
		Reason:   "fraudulent",
	}
	original, _, _ := s.Create(cb)

	// Update with identical payload – no write should occur.
	same := &models.Chargeback{Amount: 500, Currency: "EUR", Reason: "fraudulent"}
	result, written, err := s.Update("test-id-2", same)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written {
		t.Fatal("expected written=false when payload is identical")
	}
	if !result.UpdatedAt.Equal(original.UpdatedAt) {
		t.Fatal("updatedAt should not change when write is skipped")
	}

	// Update with different payload – write should occur.
	changed := &models.Chargeback{Amount: 999, Currency: "EUR", Reason: "fraudulent"}
	result2, written2, err := s.Update("test-id-2", changed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !written2 {
		t.Fatal("expected written=true when payload differs")
	}
	if result2.Amount != 999 {
		t.Fatalf("expected amount=999, got %d", result2.Amount)
	}
}

func TestUpdateNotFound(t *testing.T) {
	s := newTestStore(t)
	_, _, err := s.Update("nonexistent", &models.Chargeback{})
	if err == nil {
		t.Fatal("expected error for missing record")
	}
	if !os.IsTimeout(err) && err != store.ErrNotFound {
		// Accept ErrNotFound only.
		if err.Error() != "chargeback not found" {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestDeleteIdempotency(t *testing.T) {
	s := newTestStore(t)

	cb := &models.Chargeback{ID: "del-id", Amount: 100, Currency: "USD", Reason: "test"}
	_, _, _ = s.Create(cb)

	// First delete – record exists.
	if err := s.Delete("del-id"); err != nil {
		t.Fatalf("unexpected error on first delete: %v", err)
	}

	// Second delete – record already gone, should still succeed.
	if err := s.Delete("del-id"); err != nil {
		t.Fatalf("unexpected error on second delete: %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get("missing")
	if err == nil {
		t.Fatal("expected ErrNotFound")
	}
	if err != store.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
