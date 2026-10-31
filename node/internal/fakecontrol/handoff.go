package fakecontrol

import (
	"crypto/sha256"
	"encoding/hex"
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

// UsageBodies returns detached, accepted TEST ONLY usage requests, in order.
func (s *Server) UsageBodies() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]byte, len(s.usageBodies))
	for i, b := range s.usageBodies {
		out[i] = append([]byte(nil), b...)
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
