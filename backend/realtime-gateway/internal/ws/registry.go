package ws

import (
	"sync"
	"sync/atomic"
)

// Registry tracks active WebSocket connections.
type Registry struct {
	mu    sync.RWMutex
	conns map[string]*Conn
	count atomic.Int64
}

// NewRegistry creates a new connection registry.
func NewRegistry() *Registry {
	return &Registry{
		conns: make(map[string]*Conn),
	}
}

// Add registers a connection.
func (r *Registry) Add(c *Conn) {
	r.mu.Lock()
	r.conns[c.ID] = c
	r.mu.Unlock()
	r.count.Add(1)
}

// Remove unregisters a connection.
func (r *Registry) Remove(id string) {
	r.mu.Lock()
	if _, ok := r.conns[id]; ok {
		delete(r.conns, id)
		r.mu.Unlock()
		r.count.Add(-1)
		return
	}
	r.mu.Unlock()
}

// Count returns the number of active connections.
func (r *Registry) Count() int64 {
	return r.count.Load()
}
