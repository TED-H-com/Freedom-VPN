package netbird

import (
	"context"
	"sync"
	"time"
)

// Binding maps external_bindings (§15.3). Hash/time are set only after read-back.
type Binding struct {
	DeploymentID, Kind, InternalID, ExternalID, DesiredHash string
	LastObservedAt                                          time.Time
}

// BindingStore must enforce both unique keys (deployment,kind,internal ID) and
// (deployment,kind,external ID). Lock serializes ALL writers for a deployment,
// including read/modify/write of shared groups. A DB implementation must use a
// worker lease/advisory lock, not hold a request transaction across HTTP (§14.5).
// NetBird has no ETags for these resources: dashboard writes must be excluded
// from adapter-owned resources. All adapters for a deployment share one store.
type BindingStore interface {
	Lock(context.Context, string) (func(), error)
	Get(context.Context, string, string, string) (Binding, bool, error)
	Put(context.Context, Binding) error
	Delete(context.Context, string, string, string) error
}
type MemoryBindingStore struct {
	mu       sync.Mutex
	bindings map[[3]string]Binding
	locks    map[string]chan struct{}
}

func NewMemoryBindingStore() *MemoryBindingStore {
	return &MemoryBindingStore{bindings: make(map[[3]string]Binding), locks: make(map[string]chan struct{})}
}
func (s *MemoryBindingStore) Lock(ctx context.Context, d string) (func(), error) {
	s.mu.Lock()
	ch := s.locks[d]
	if ch == nil {
		ch = make(chan struct{}, 1)
		s.locks[d] = ch
	}
	s.mu.Unlock()
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (s *MemoryBindingStore) Get(ctx context.Context, d, k, id string) (Binding, bool, error) {
	if e := ctx.Err(); e != nil {
		return Binding{}, false, e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.bindings[[3]string{d, k, id}]
	return b, ok, nil
}
func (s *MemoryBindingStore) Put(ctx context.Context, b Binding) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := [3]string{b.DeploymentID, b.Kind, b.InternalID}
	for k, v := range s.bindings {
		if k != key && v.DeploymentID == b.DeploymentID && v.Kind == b.Kind && v.ExternalID == b.ExternalID {
			return failure(Conflict)
		}
	}
	if old, ok := s.bindings[key]; ok && old.ExternalID != b.ExternalID {
		return failure(Conflict)
	}
	s.bindings[key] = b
	return nil
}
func (s *MemoryBindingStore) Delete(ctx context.Context, d, k, id string) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.bindings, [3]string{d, k, id})
	return nil
}
