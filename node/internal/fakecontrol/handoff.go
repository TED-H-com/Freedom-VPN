package fakecontrol

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/google/uuid"
)

// AddReenrollmentToken is a TEST ONLY enrollment fixture for a verified new
// incarnation of an existing node. Issuing its certificate retires old clients.
func (s *Server) AddReenrollmentToken(token string, node uuid.UUID) {
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reenrollTargets == nil {
		s.reenrollTargets = map[string]uuid.UUID{}
	}
	s.tokens[hash] = nil
	s.reenrollTargets[hash] = node
}

// UsageAttempt records a processed TEST ONLY request. Accepted and Duplicates
// count event receipts, so a retried POST is distinguishable from attribution.
type UsageAttempt struct {
	IdempotencyKey string
	Body           []byte
	Accepted       int
	Duplicates     int
}

// UsageAttempts returns detached TEST ONLY request evidence in processing order.
func (s *Server) UsageAttempts() []UsageAttempt {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]UsageAttempt(nil), s.usageAttempts...)
	for i := range out {
		out[i].Body = append([]byte(nil), out[i].Body...)
	}
	return out
}

// HoldNextUsageResponse pauses one TEST ONLY response after ledger acceptance.
// accepted closes before any response is sent. The handler resumes on release
// or request cancellation, without holding the server mutex. Release is safe
// to call again during cleanup.
func (s *Server) HoldNextUsageResponse() (accepted <-chan struct{}, release func()) {
	ready, resume := make(chan struct{}), make(chan struct{})
	s.mu.Lock()
	s.usageResponseHold = func(ctx context.Context) {
		close(ready)
		select {
		case <-resume:
		case <-ctx.Done():
		}
	}
	s.mu.Unlock()
	return ready, sync.OnceFunc(func() { close(resume) })
}

// UsageBodies returns detached, processed TEST ONLY usage requests, in order.
// This includes retries whose event receipts are all duplicates.
func (s *Server) UsageBodies() [][]byte {
	attempts := s.UsageAttempts()
	out := make([][]byte, len(attempts))
	for i, attempt := range attempts {
		out[i] = attempt.Body
	}
	return out
}

// SetMaxOfflineSeconds selects the TEST ONLY lease cadence fixture. signLease
// retains its ordinary 60..86400 second bounds. Drop sticky replay explicitly.
func (s *Server) SetMaxOfflineSeconds(seconds int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxOffline = seconds
	s.stickyRaw = nil
	s.stickyETag = ""
}
