//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os/exec"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace"
	"github.com/google/uuid"
)

type FirstOfferEvidence = offertrace.Evidence

// ArmFirstDesiredOffer holds actual desired GETs and starts a new capture.
// Pause the agent before arming if a previous request may be in flight. After
// restore, resume observations while held, then ReleaseDesiredOffer. Do not send
// diagnostic desired GETs through the proxy while armed: they would be captured.
func (s *Stack) ArmFirstDesiredOffer(ctx context.Context, slot string) (string, error) {
	c, _, err := s.usageSlot(slot)
	if err != nil {
		return "", err
	}
	if err = writeContainerFile(ctx, c, "/var/lib/vpn-node/e2e/hold-desired", "600", []byte("hold\n")); err != nil {
		return "", err
	}
	id := uuid.NewString()
	return id, writeContainerFile(ctx, c, "/var/lib/vpn-node/e2e/first-offer-arm", "600", []byte(id))
}
func (s *Stack) ReleaseDesiredOffer(ctx context.Context, slot string) error {
	c, _, err := s.usageSlot(slot)
	if err != nil {
		return err
	}
	return exec.CommandContext(ctx, "docker", "exec", c, "rm", "-f", "/var/lib/vpn-node/e2e/hold-desired").Run()
}
func (s *Stack) FirstDesiredOffer(ctx context.Context, slot string) (FirstOfferEvidence, error) {
	var out FirstOfferEvidence
	c, _, err := s.usageSlot(slot)
	if err != nil {
		return out, err
	}
	raw, err := exec.CommandContext(ctx, "docker", "exec", c, "cat", "/var/lib/vpn-node/e2e/first-offer.json").Output()
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	return out, err
}
