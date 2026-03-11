package store

import (
	"context"
	"strings"
	"testing"
)

func TestCollectionConstants(t *testing.T) {
	if SessionsCollection != "sessions" || MetricsCollection != "metrics" || HistoryCollection != "history" || SearchesCollection != "searches" || NavigatorReplaysCollection != "navigator_replays" || UsersCollection != "users" || MemorySubcollection != "memory" {
		t.Fatal("collection constants changed unexpectedly")
	}
}

func TestClientStoresProjectID(t *testing.T) {
	c := &Client{projectID: "vibecat-test"}
	if c.projectID != "vibecat-test" {
		t.Fatalf("projectID = %q", c.projectID)
	}
}

func TestNilInputGuards(t *testing.T) {
	ctx := context.Background()
	c := &Client{}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "UpdateSessionMetrics nil metrics", err: c.UpdateSessionMetrics(ctx, "s1", nil), want: "metrics cannot be nil"},
		{name: "UpdateMemory nil entry", err: c.UpdateMemory(ctx, "u1", nil), want: "memory entry cannot be nil"},
		{name: "CreateSession nil session", err: c.CreateSession(ctx, nil), want: "session cannot be nil"},
		{name: "UpdateSession nil session", err: c.UpdateSession(ctx, nil), want: "session cannot be nil"},
		{name: "AddHistoryEntry nil entry", err: c.AddHistoryEntry(ctx, "s1", nil), want: "history entry cannot be nil"},
		{name: "CacheSearchResult nil entry", err: c.CacheSearchResult(ctx, "s1", nil), want: "search cache entry cannot be nil"},
		{name: "StoreNavigatorReplay nil entry", err: c.StoreNavigatorReplay(ctx, nil), want: "navigator replay cannot be nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("expected error containing %q", tt.want)
			}
			if !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("error = %q, want to contain %q", tt.err.Error(), tt.want)
			}
		})
	}
}
