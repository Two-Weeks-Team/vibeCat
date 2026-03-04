package graph

import "testing"

func TestNew(t *testing.T) {
	g, err := New(nil, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
}
