package ws

import (
	"fmt"
	"sync"
	"sync/atomic"

	"vibecat/realtime-gateway/internal/live"
)

type Registry struct {
	mu        sync.RWMutex
	conns     map[string]*Conn
	sessGetFn map[string]func() *live.Session
	count     atomic.Int64
}

func NewRegistry() *Registry {
	return &Registry{
		conns:     make(map[string]*Conn),
		sessGetFn: make(map[string]func() *live.Session),
	}
}

func (r *Registry) Add(c *Conn) {
	r.mu.Lock()
	r.conns[c.ID] = c
	r.mu.Unlock()
	r.count.Add(1)
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	if _, ok := r.conns[id]; ok {
		delete(r.conns, id)
		delete(r.sessGetFn, id)
		r.mu.Unlock()
		r.count.Add(-1)
		return
	}
	r.mu.Unlock()
}

func (r *Registry) Count() int64 {
	return r.count.Load()
}

func (r *Registry) SetSessionGetter(id string, fn func() *live.Session) {
	r.mu.Lock()
	r.sessGetFn[id] = fn
	r.mu.Unlock()
}

func (r *Registry) InjectText(text string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, fn := range r.sessGetFn {
		s := fn()
		if s != nil {
			return s.SendText(text)
		}
	}
	return fmt.Errorf("no active session")
}
