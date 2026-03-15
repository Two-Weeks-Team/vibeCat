package ws

import (
	"sync"
	"sync/atomic"

	"vibecat/realtime-gateway/internal/live"
)

type Registry struct {
	mu       sync.RWMutex
	conns    map[string]*Conn
	sessions map[string]*live.Session
	count    atomic.Int64
}

func NewRegistry() *Registry {
	return &Registry{
		conns:    make(map[string]*Conn),
		sessions: make(map[string]*live.Session),
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

func (r *Registry) Count() int64 {
	return r.count.Load()
}

func (r *Registry) SetSession(id string, s *live.Session) {
	r.mu.Lock()
	r.sessions[id] = s
	r.mu.Unlock()
}

func (r *Registry) RemoveSession(id string) {
	r.mu.Lock()
	delete(r.sessions, id)
	r.mu.Unlock()
}

func (r *Registry) InjectText(text string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.sessions {
		return s.SendText(text)
	}
	return nil
}
