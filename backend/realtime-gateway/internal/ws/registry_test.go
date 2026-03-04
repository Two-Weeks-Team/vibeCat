package ws

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected registry, got nil")
	}
	if got := r.Count(); got != 0 {
		t.Fatalf("Count() = %d, want 0", got)
	}
}

func TestRegistryAddRemoveCount(t *testing.T) {
	r := NewRegistry()
	c := &Conn{ID: "c1"}

	r.Add(c)
	if got := r.Count(); got != 1 {
		t.Fatalf("Count() after Add = %d, want 1", got)
	}

	r.Remove(c.ID)
	if got := r.Count(); got != 0 {
		t.Fatalf("Count() after Remove = %d, want 0", got)
	}
}

func TestRegistryRemoveNonExistent(t *testing.T) {
	r := NewRegistry()
	r.Add(&Conn{ID: "c1"})

	r.Remove("does-not-exist")

	if got := r.Count(); got != 1 {
		t.Fatalf("Count() after removing non-existent = %d, want 1", got)
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	const n = 200

	var addWG sync.WaitGroup
	for i := range n {
		addWG.Add(1)
		go func(i int) {
			defer addWG.Done()
			r.Add(&Conn{ID: fmt.Sprintf("conn-%d", i)})
		}(i)
	}
	addWG.Wait()

	if got := r.Count(); got != n {
		t.Fatalf("Count() after concurrent Add = %d, want %d", got, n)
	}

	var removeWG sync.WaitGroup
	for i := range n {
		removeWG.Add(1)
		go func(i int) {
			defer removeWG.Done()
			r.Remove(fmt.Sprintf("conn-%d", i))
		}(i)
	}
	removeWG.Wait()

	if got := r.Count(); got != 0 {
		t.Fatalf("Count() after concurrent Remove = %d, want 0", got)
	}
}
