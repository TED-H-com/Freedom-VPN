package fleetupdate

import (
	"github.com/google/uuid"
	"math/rand/v2"
	"testing"
)

func TestUPD05_CountingInvariantRandomSequences(t *testing.T) {
	random := rand.New(rand.NewPCG(23, 27))
	states := []string{"queued", "waiting_online", "downloading", "verifying", "applying", "checking", "succeeded", "already_current", "failed", "rolled_back", "superseded", "superseded_incarnation", "cancelled_revoked", "cancelled_retired"}
	targets := make([]string, 100)
	for i := range targets {
		targets[i] = "queued"
	}
	for step := 0; step < 10000; step++ {
		targets[random.IntN(len(targets))] = states[random.IntN(len(states))]
		c, _, e := CountStates(targets)
		if e != nil {
			t.Fatal(e)
		}
		sum := c.Queued + c.WaitingOnline + c.Downloading + c.Verifying + c.Applying + c.Checking + c.Succeeded + c.AlreadyCurrent + c.Failed + c.RolledBack + c.Superseded + c.Cancelled
		if sum != len(targets) {
			t.Fatalf("step %d sum %d", step, sum)
		}
	}
}
func TestAT27_ServingIndependentOfDesired(t *testing.T) {
	id := uuid.New()
	c := ServingConditions{IdentityValid: true, ProvisioningReady: true, PolicyValid: true, MemberEntitlementValid: true, PublicationIntent: "published", CurrentReleaseID: id, ServingReleaseIDs: []uuid.UUID{id}, UpdateState: "downloading"}
	if !ServingAllowed(c) {
		t.Fatal("supported old release blocked")
	}
	for _, phase := range []string{"applying", "checking"} {
		c.UpdateState = phase
		if ServingAllowed(c) {
			t.Fatal(phase)
		}
	}
	c.UpdateState = "verifying"
	c.PublicationIntent = "disabled"
	if ServingAllowed(c) {
		t.Fatal("disabled allowed")
	}
	c.PublicationIntent = "published"
	c.ServingReleaseIDs = nil
	if ServingAllowed(c) {
		t.Fatal("unsupported allowed")
	}
}
