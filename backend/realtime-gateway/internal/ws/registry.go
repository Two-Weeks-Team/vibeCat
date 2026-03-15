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

func (r *Registry) DispatchStep(url, text, target string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, conn := range r.conns {
		taskID := "debug_" + fmt.Sprintf("%d", time.Now().UnixMilli())
		command := "navigate_open_url: " + url
		actionType := "open_url"
		if url == "" {
			command = "navigate_text_entry: " + text
			actionType = "paste_text"
		}
		lockedSendJSON(conn, map[string]any{
			"type":             "navigator.commandAccepted",
			"taskId":           taskID,
			"command":          command,
			"intentClass":      "execute_now",
			"intentConfidence": 0.95,
			"source":           "debug_execute",
		})
		stepID := taskID + "_step"
		step := map[string]any{
			"id": stepID, "actionType": actionType, "targetApp": "Chrome",
			"proofLevel": "none", "macroID": "fc_open_url",
			"narration": "Opening URL", "timeoutMs": 3000,
		}
		if url != "" {
			step["url"] = url
		} else {
			step["targetApp"] = target
			step["inputText"] = text
			step["actionType"] = "paste_text"
			step["targetDescriptor"] = map[string]any{"appName": target}
		}
		lockedSendJSON(conn, map[string]any{
			"type":    "navigator.stepPlanned",
			"taskId":  taskID,
			"step":    step,
			"message": command,
		})
		return nil
	}
	return fmt.Errorf("no active connection")
}
