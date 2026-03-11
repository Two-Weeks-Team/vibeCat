package ws

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ActionStateStore interface {
	Load(context.Context, string) (navigatorSessionState, bool, error)
	Save(context.Context, string, navigatorSessionState) error
	Delete(context.Context, string) error
}

type ChainedActionStateStore struct {
	stores []ActionStateStore
}

func NewChainedActionStateStore(stores ...ActionStateStore) *ChainedActionStateStore {
	filtered := make([]ActionStateStore, 0, len(stores))
	for _, store := range stores {
		if store != nil {
			filtered = append(filtered, store)
		}
	}
	return &ChainedActionStateStore{stores: filtered}
}

func (s *ChainedActionStateStore) Load(ctx context.Context, owner string) (navigatorSessionState, bool, error) {
	for idx, store := range s.stores {
		state, ok, err := store.Load(ctx, owner)
		if err != nil {
			return navigatorSessionState{}, false, err
		}
		if !ok {
			continue
		}
		for warmIdx := 0; warmIdx < idx; warmIdx++ {
			_ = s.stores[warmIdx].Save(ctx, owner, state)
		}
		return state, true, nil
	}
	return navigatorSessionState{}, false, nil
}

func (s *ChainedActionStateStore) Save(ctx context.Context, owner string, state navigatorSessionState) error {
	for _, store := range s.stores {
		if err := store.Save(ctx, owner, state); err != nil {
			return err
		}
	}
	return nil
}

func (s *ChainedActionStateStore) Delete(ctx context.Context, owner string) error {
	for _, store := range s.stores {
		if err := store.Delete(ctx, owner); err != nil {
			return err
		}
	}
	return nil
}

type InMemoryActionStateStore struct {
	mu     sync.RWMutex
	states map[string]navigatorSessionState
}

func NewInMemoryActionStateStore() *InMemoryActionStateStore {
	return &InMemoryActionStateStore{
		states: map[string]navigatorSessionState{},
	}
}

func (s *InMemoryActionStateStore) Load(_ context.Context, owner string) (navigatorSessionState, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[owner]
	if !ok {
		return navigatorSessionState{}, false, nil
	}
	return cloneNavigatorSessionState(state), true, nil
}

func (s *InMemoryActionStateStore) Save(_ context.Context, owner string, state navigatorSessionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[owner] = cloneNavigatorSessionState(state)
	return nil
}

func (s *InMemoryActionStateStore) Delete(_ context.Context, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, owner)
	return nil
}

func cloneNavigatorSessionState(state navigatorSessionState) navigatorSessionState {
	cloned := state
	if len(state.steps) > 0 {
		cloned.steps = append([]navigatorStep(nil), state.steps...)
	}
	if len(state.stepHistory) > 0 {
		cloned.stepHistory = append([]navigatorStepTrace(nil), state.stepHistory...)
	}
	return cloned
}

const defaultActionStateCollection = "navigator_action_states"

type FirestoreActionStateStore struct {
	client     *firestore.Client
	collection string
}

type firestoreActionStateRecord struct {
	Owner                       string                   `firestore:"owner"`
	DeviceID                    string                   `firestore:"deviceId,omitempty"`
	ConnectionID                string                   `firestore:"connectionId,omitempty"`
	ActiveTaskID                string                   `firestore:"activeTaskId,omitempty"`
	ActiveCommand               string                   `firestore:"activeCommand,omitempty"`
	PendingClarificationKind    string                   `firestore:"pendingClarificationKind,omitempty"`
	PendingClarificationCommand string                   `firestore:"pendingClarificationCommand,omitempty"`
	PendingRiskyCommand         string                   `firestore:"pendingRiskyCommand,omitempty"`
	InitialContext              navigatorContextSnapshot `firestore:"initialContext,omitempty"`
	InitialContextHash          string                   `firestore:"initialContextHash,omitempty"`
	InitialAppName              string                   `firestore:"initialAppName,omitempty"`
	InitialWindowTitle          string                   `firestore:"initialWindowTitle,omitempty"`
	Steps                       []navigatorStep          `firestore:"steps,omitempty"`
	StepHistory                 []navigatorStepTrace     `firestore:"stepHistory,omitempty"`
	NextStepIndex               int                      `firestore:"nextStepIndex,omitempty"`
	CurrentStepID               string                   `firestore:"currentStepId,omitempty"`
	StepIndex                   int                      `firestore:"stepIndex,omitempty"`
	Status                      string                   `firestore:"status,omitempty"`
	RiskState                   string                   `firestore:"riskState,omitempty"`
	PromptState                 string                   `firestore:"promptState,omitempty"`
	LastVerifiedContextHash     string                   `firestore:"lastVerifiedContextHash,omitempty"`
	CreatedAt                   time.Time                `firestore:"createdAt,omitempty"`
	UpdatedAt                   time.Time                `firestore:"updatedAt,omitempty"`
}

func NewFirestoreActionStateStore(ctx context.Context, projectID, collection string) (*FirestoreActionStateStore, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("project id is required")
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		collection = defaultActionStateCollection
	}
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create firestore action state client: %w", err)
	}
	return &FirestoreActionStateStore{
		client:     client,
		collection: collection,
	}, nil
}

func (s *FirestoreActionStateStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

func (s *FirestoreActionStateStore) Load(ctx context.Context, owner string) (navigatorSessionState, bool, error) {
	if s == nil || s.client == nil {
		return navigatorSessionState{}, false, nil
	}
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return navigatorSessionState{}, false, nil
	}
	docSnap, err := s.doc(owner).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return navigatorSessionState{}, false, nil
		}
		return navigatorSessionState{}, false, fmt.Errorf("load action state: %w", err)
	}

	var record firestoreActionStateRecord
	if err := docSnap.DataTo(&record); err != nil {
		return navigatorSessionState{}, false, fmt.Errorf("decode action state: %w", err)
	}
	return record.toNavigatorSessionState(), true, nil
}

func (s *FirestoreActionStateStore) Save(ctx context.Context, owner string, state navigatorSessionState) error {
	if s == nil || s.client == nil {
		return nil
	}
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return fmt.Errorf("owner is required")
	}
	record := newFirestoreActionStateRecord(owner, state)
	if _, err := s.doc(owner).Set(ctx, record); err != nil {
		return fmt.Errorf("save action state: %w", err)
	}
	return nil
}

func (s *FirestoreActionStateStore) Delete(ctx context.Context, owner string) error {
	if s == nil || s.client == nil {
		return nil
	}
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return nil
	}
	if _, err := s.doc(owner).Delete(ctx); err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("delete action state: %w", err)
	}
	return nil
}

func (s *FirestoreActionStateStore) doc(owner string) *firestore.DocumentRef {
	return s.client.Collection(s.collection).Doc(actionStateDocumentID(owner))
}

func newFirestoreActionStateRecord(owner string, state navigatorSessionState) firestoreActionStateRecord {
	record := firestoreActionStateRecord{
		Owner:                       owner,
		DeviceID:                    strings.TrimSpace(state.deviceID),
		ConnectionID:                strings.TrimSpace(state.connectionID),
		ActiveTaskID:                strings.TrimSpace(state.activeTaskID),
		ActiveCommand:               strings.TrimSpace(state.activeCommand),
		PendingClarificationKind:    strings.TrimSpace(string(state.pendingClarificationKind)),
		PendingClarificationCommand: strings.TrimSpace(state.pendingClarificationCommand),
		PendingRiskyCommand:         strings.TrimSpace(state.pendingRiskyCommand),
		InitialContext:              state.initialContext,
		InitialContextHash:          strings.TrimSpace(state.initialContextHash),
		InitialAppName:              strings.TrimSpace(state.initialAppName),
		InitialWindowTitle:          strings.TrimSpace(state.initialWindowTitle),
		Steps:                       append([]navigatorStep(nil), state.steps...),
		StepHistory:                 append([]navigatorStepTrace(nil), state.stepHistory...),
		NextStepIndex:               state.nextStepIndex,
		CurrentStepID:               strings.TrimSpace(state.currentStepID),
		LastVerifiedContextHash:     strings.TrimSpace(state.lastVerifiedContextHash),
		CreatedAt:                   state.createdAt.UTC(),
		UpdatedAt:                   state.updatedAt.UTC(),
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	record.StepIndex = actionStateStepIndex(state)
	record.Status = actionStateStatus(state)
	record.RiskState = actionStateRiskState(state)
	record.PromptState = actionStatePromptState(state)
	return record
}

func (r firestoreActionStateRecord) toNavigatorSessionState() navigatorSessionState {
	state := navigatorSessionState{
		activeTaskID:                strings.TrimSpace(r.ActiveTaskID),
		activeCommand:               strings.TrimSpace(r.ActiveCommand),
		pendingClarificationKind:    navigatorPromptKind(strings.TrimSpace(r.PendingClarificationKind)),
		pendingClarificationCommand: strings.TrimSpace(r.PendingClarificationCommand),
		pendingRiskyCommand:         strings.TrimSpace(r.PendingRiskyCommand),
		initialContext:              r.InitialContext,
		initialContextHash:          strings.TrimSpace(r.InitialContextHash),
		initialAppName:              strings.TrimSpace(r.InitialAppName),
		initialWindowTitle:          strings.TrimSpace(r.InitialWindowTitle),
		steps:                       append([]navigatorStep(nil), r.Steps...),
		stepHistory:                 append([]navigatorStepTrace(nil), r.StepHistory...),
		nextStepIndex:               r.NextStepIndex,
		currentStepID:               strings.TrimSpace(r.CurrentStepID),
		deviceID:                    strings.TrimSpace(r.DeviceID),
		connectionID:                strings.TrimSpace(r.ConnectionID),
		lastVerifiedContextHash:     strings.TrimSpace(r.LastVerifiedContextHash),
		createdAt:                   r.CreatedAt.UTC(),
		updatedAt:                   r.UpdatedAt.UTC(),
	}
	if state.createdAt.IsZero() {
		state.createdAt = state.updatedAt
	}
	return state
}

func actionStateStatus(state navigatorSessionState) string {
	switch {
	case strings.TrimSpace(state.pendingRiskyCommand) != "":
		return "awaiting_risk_confirmation"
	case strings.TrimSpace(state.pendingClarificationCommand) != "":
		return "awaiting_clarification"
	case strings.TrimSpace(state.activeTaskID) == "":
		return ""
	case strings.TrimSpace(state.currentStepID) != "":
		return "running"
	default:
		return "planned"
	}
}

func actionStateRiskState(state navigatorSessionState) string {
	switch {
	case strings.TrimSpace(state.pendingRiskyCommand) != "":
		return "pending_confirmation"
	case strings.TrimSpace(state.activeTaskID) != "":
		return "active"
	default:
		return ""
	}
}

func actionStatePromptState(state navigatorSessionState) string {
	switch {
	case strings.TrimSpace(state.pendingRiskyCommand) != "":
		return "risky_action_confirmation"
	case state.pendingClarificationKind != "":
		return string(state.pendingClarificationKind)
	default:
		return ""
	}
}

func actionStateStepIndex(state navigatorSessionState) int {
	if strings.TrimSpace(state.currentStepID) != "" && state.nextStepIndex > 0 {
		return state.nextStepIndex - 1
	}
	return state.nextStepIndex
}

func actionStateDocumentID(owner string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(owner)))
	return hex.EncodeToString(sum[:])
}
