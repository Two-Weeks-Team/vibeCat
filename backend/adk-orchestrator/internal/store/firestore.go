package store

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// Collection names
	SessionsCollection         = "sessions"
	MetricsCollection          = "metrics"
	HistoryCollection          = "history"
	SearchesCollection         = "searches"
	NavigatorReplaysCollection = "navigator_replays"
	UsersCollection            = "users"
	MemorySubcollection        = "memory"
)

// Client wraps the Firestore client with typed helpers.
type Client struct {
	client    *firestore.Client
	projectID string
}

// NewClient creates a new Firestore client for the given project.
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}

	return &Client{
		client:    client,
		projectID: projectID,
	}, nil
}

// Close closes the Firestore client.
func (c *Client) Close() error {
	return c.client.Close()
}

// GetSessionMetrics reads session metrics from Firestore.
// Returns (nil, nil) if the document does not exist.
func (c *Client) GetSessionMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error) {
	docRef := c.client.Collection(SessionsCollection).Doc(sessionID).Collection(MetricsCollection).Doc("current")
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session metrics: %w", err)
	}

	var metrics SessionMetrics
	if err := docSnap.DataTo(&metrics); err != nil {
		return nil, fmt.Errorf("failed to decode session metrics: %w", err)
	}

	return &metrics, nil
}

// UpdateSessionMetrics writes or updates session metrics in Firestore.
func (c *Client) UpdateSessionMetrics(ctx context.Context, sessionID string, metrics *SessionMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	docRef := c.client.Collection(SessionsCollection).Doc(sessionID).Collection(MetricsCollection).Doc("current")
	_, err := docRef.Set(ctx, metrics)
	if err != nil {
		return fmt.Errorf("failed to update session metrics: %w", err)
	}

	return nil
}

// GetMemory reads a user's cross-session memory from Firestore.
// Returns (nil, nil) if the document does not exist.
func (c *Client) GetMemory(ctx context.Context, userID string) (*MemoryEntry, error) {
	docRef := c.client.Collection(UsersCollection).Doc(userID).Collection(MemorySubcollection).Doc("data")
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user memory: %w", err)
	}

	var entry MemoryEntry
	if err := docSnap.DataTo(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode memory entry: %w", err)
	}

	return &entry, nil
}

// UpdateMemory writes or updates a user's cross-session memory in Firestore.
func (c *Client) UpdateMemory(ctx context.Context, userID string, entry *MemoryEntry) error {
	if entry == nil {
		return fmt.Errorf("memory entry cannot be nil")
	}

	docRef := c.client.Collection(UsersCollection).Doc(userID).Collection(MemorySubcollection).Doc("data")
	_, err := docRef.Set(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to update user memory: %w", err)
	}

	return nil
}

// CreateSession creates a new session document in Firestore.
func (c *Client) CreateSession(ctx context.Context, session *Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	docRef := c.client.Collection(SessionsCollection).Doc(session.ID)
	_, err := docRef.Create(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession reads a session from Firestore.
// Returns (nil, nil) if the document does not exist.
func (c *Client) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	docRef := c.client.Collection(SessionsCollection).Doc(sessionID)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	if err := docSnap.DataTo(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	return &session, nil
}

// UpdateSession updates an existing session in Firestore.
func (c *Client) UpdateSession(ctx context.Context, session *Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	docRef := c.client.Collection(SessionsCollection).Doc(session.ID)
	_, err := docRef.Set(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// AddHistoryEntry adds a history entry to a session.
func (c *Client) AddHistoryEntry(ctx context.Context, sessionID string, entry *HistoryEntry) error {
	if entry == nil {
		return fmt.Errorf("history entry cannot be nil")
	}

	collectionRef := c.client.Collection(SessionsCollection).Doc(sessionID).Collection(HistoryCollection)
	_, _, err := collectionRef.Add(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to add history entry: %w", err)
	}

	return nil
}

// CacheSearchResult caches a search result to avoid duplicates.
func (c *Client) CacheSearchResult(ctx context.Context, sessionID string, entry *SearchCacheEntry) error {
	if entry == nil {
		return fmt.Errorf("search cache entry cannot be nil")
	}

	docRef := c.client.Collection(SessionsCollection).Doc(sessionID).Collection(SearchesCollection).Doc(entry.Query)
	_, err := docRef.Set(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to cache search result: %w", err)
	}

	return nil
}

// GetCachedSearch retrieves a cached search result by query.
// Returns (nil, nil) if the document does not exist.
func (c *Client) GetCachedSearch(ctx context.Context, sessionID string, query string) (*SearchCacheEntry, error) {
	docRef := c.client.Collection(SessionsCollection).Doc(sessionID).Collection(SearchesCollection).Doc(query)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cached search: %w", err)
	}

	var entry SearchCacheEntry
	if err := docSnap.DataTo(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode search cache entry: %w", err)
	}

	return &entry, nil
}

// StoreNavigatorReplay writes a navigator replay summary keyed by task ID.
func (c *Client) StoreNavigatorReplay(ctx context.Context, replay *NavigatorReplay) error {
	if replay == nil {
		return fmt.Errorf("navigator replay cannot be nil")
	}
	docID := replay.TaskID
	if docID == "" {
		return fmt.Errorf("navigator replay taskId cannot be empty")
	}

	docRef := c.client.Collection(NavigatorReplaysCollection).Doc(docID)
	_, err := docRef.Set(ctx, replay)
	if err != nil {
		return fmt.Errorf("failed to store navigator replay: %w", err)
	}
	return nil
}
