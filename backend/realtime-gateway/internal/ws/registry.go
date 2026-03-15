package ws

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

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

func (r *Registry) DispatchStep(step map[string]any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, conn := range r.conns {
		lockedSendJSON(conn, step)
		return nil
	}
	return fmt.Errorf("no active connection")
}

func BuildDebugStep(url, text, target string) map[string]any {
	taskID := "debug_" + fmt.Sprintf("%d", time.Now().UnixMilli())
	if url != "" {
		return map[string]any{
			"type":    "navigator.stepPlanned",
			"taskId":  taskID,
			"step":    map[string]any{"id": taskID + "_open", "actionType": "open_url", "targetApp": "Chrome", "url": url, "proofLevel": "none"},
			"message": "Opening URL",
		}
	}
	return map[string]any{
		"type":   "navigator.stepPlanned",
		"taskId": taskID,
		"step": map[string]any{
			"id": taskID + "_text", "actionType": "paste_text", "targetApp": target,
			"inputText": text, "proofLevel": "none",
			"targetDescriptor": map[string]any{"appName": target},
		},
		"message": "Typing text",
	}
}
