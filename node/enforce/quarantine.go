package enforce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// Empty, unreferenced ifname sets in our table persist quarantine across
// processes and baseline/serving renders without changing counters or links'
// aliases. Only a verified Apply removes them.
const quarantinePrefix = "quarantine_"

// LinkState separates administrative UP from operstate (a tunnel may report
// UNKNOWN while administratively up). Neither addresses nor routes are kept.
type LinkState struct {
	AdminUp   bool   `json:"admin_up"`
	OperState string `json:"operstate"`
}

type QuarantineLink struct {
	Interface string     `json:"interface"`
	Before    *LinkState `json:"before,omitempty"`
	After     *LinkState `json:"after,omitempty"`
	Error     string     `json:"error,omitempty"`
}

type BaselineEvidence struct {
	Installed bool   `json:"installed"`
	Verified  bool   `json:"verified"`
	Error     string `json:"error,omitempty"`
}

// QuarantineEvidence distinguishes read-back containment from attempted work.
// Quarantined requires every member link down and a verified nft baseline.
// Recorded says the recovery marker was read back; errors are fixed statuses.
type QuarantineEvidence struct {
	Quarantined bool             `json:"quarantined"`
	Recorded    bool             `json:"recorded"`
	Links       []QuarantineLink `json:"links"`
	Baseline    BaselineEvidence `json:"baseline"`
}

// Quarantine implements D-090. Pass the retained applied model, not an empty
// or newly desired model, and the executor for that model's network namespace.
// The current model has one member-facing link: Datapath.MemberIngress. The
// uplink must be distinct and must carry the node's control path independently.
// Call on EVERY step while serving verification fails with foreign tc state,
// before publishing the heartbeat, even when already blocked: another daemon
// can raise the link. Serialize with Apply/baseline/verification calls.
//
// Quarantine attempts link-down and the no-probe nft baseline independently,
// even after a read or command failure. It never mutates tc, routes, the uplink
// or host DNS. A non-nil error means containment or its durable marker was not
// fully established; report the partial evidence and remain non-serving.
// Only a clean, fully verified Apply may raise a recorded quarantined link.
func Quarantine(ctx context.Context, ex Executor, m Model) (ev QuarantineEvidence, err error) {
	if err := validateDatapath(m.Datapath); err != nil {
		return ev, err
	}
	iface := m.Datapath.MemberIngress
	// Record before link-down so a process exit cannot lose the recovery intent.
	marker := fmt.Sprintf("add table inet fvpn\nadd set inet fvpn %q { type ifname; }\n", quarantinePrefix+iface)
	markerErr := ex.NftBatch(ctx, marker)
	link, linkErr := lowerMemberLink(ctx, ex, iface)
	ev.Links = []QuarantineLink{link}
	baselineModel := Model{Datapath: m.Datapath}
	state, stateErr := loadState(ctx, ex)
	if stateErr == nil {
		var script string
		script, stateErr = RenderBaseline(baselineModel, state)
		if stateErr == nil {
			stateErr = ex.NftBatch(ctx, script)
			ev.Baseline.Installed = stateErr == nil
		}
	}
	if stateErr != nil {
		ev.Baseline.Error = "install_failed"
	}
	raw, readErr := ex.Output(ctx, "nft", "-j", "list", "ruleset")
	if readErr == nil {
		rs, parseErr := ParseRuleset(raw)
		ev.Recorded = parseErr == nil && rs.Sets[quarantinePrefix+iface]
		readErr = VerifyBaseline(raw, baselineModel)
	}
	if ev.Baseline.Installed {
		ev.Baseline.Verified = readErr == nil
		if readErr != nil {
			ev.Baseline.Error = "verify_failed"
		}
	}
	ev.Quarantined = link.After != nil && !link.After.AdminUp && ev.Baseline.Verified
	if markerErr != nil || !ev.Recorded {
		err = checkf(ReasonIsolation, "quarantine marker was not recorded")
	}
	if !ev.Baseline.Verified {
		err = errors.Join(err, checkf(ReasonIsolationBaselineMissing, "quarantine baseline %s", ev.Baseline.Error))
	}
	return ev, errors.Join(err, linkErr)
}

func lowerMemberLink(ctx context.Context, ex Executor, iface string) (ev QuarantineLink, err error) {
	ev.Interface = iface
	ev.Before, err = readLink(ctx, ex, iface)
	if err != nil {
		ev.Error = "prior_read_failed"
	}
	// Always retry, including if the prior read failed or said DOWN.
	if _, cmdErr := ex.Output(ctx, "ip", "link", "set", "dev", iface, "down"); cmdErr != nil {
		ev.Error = "down_failed"
	}
	var afterErr error
	ev.After, afterErr = readLink(ctx, ex, iface)
	if afterErr != nil || ev.After.AdminUp {
		ev.Error = "down_verify_failed"
	}
	if ev.Error != "" {
		return ev, checkf(ReasonIsolation, "quarantine interface %s %s", iface, ev.Error)
	}
	return ev, nil
}

func readLink(ctx context.Context, ex Executor, iface string) (*LinkState, error) {
	raw, err := ex.Output(ctx, "ip", "-j", "link", "show", "dev", iface)
	var links []struct {
		Name      string   `json:"ifname"`
		Flags     []string `json:"flags"`
		OperState string   `json:"operstate"`
	}
	if err != nil || json.Unmarshal(raw, &links) != nil || len(links) != 1 || links[0].Name != iface || links[0].Flags == nil || links[0].OperState == "" {
		return nil, checkf(ReasonIsolation, "interface %s link read failed", iface)
	}
	st := &LinkState{OperState: links[0].OperState}
	for _, flag := range links[0].Flags {
		st.AdminUp = st.AdminUp || flag == "UP"
	}
	return st, nil
}

func quarantinedLinks(sets map[string]bool) []string {
	var links []string
	for name := range sets {
		if iface, ok := strings.CutPrefix(name, quarantinePrefix); ok {
			links = append(links, iface)
		}
	}
	sort.Strings(links)
	return links
}

func quarantinedLinksFor(m Model, q QuarantineEvidence) []string {
	if q.Recorded {
		return []string{m.Datapath.MemberIngress}
	}
	return nil
}

// Called only after a full verification with the member link held down.
func releaseQuarantine(ctx context.Context, ex Executor, m Model, assignments map[uuid.UUID]uint16) (ServingEvidence, error) {
	iface := m.Datapath.MemberIngress
	if _, err := ex.Output(ctx, "ip", "link", "set", "dev", iface, "up"); err != nil {
		return ServingEvidence{}, checkf(ReasonIsolation, "quarantine interface %s up_failed", iface)
	}
	link, err := readLink(ctx, ex, iface)
	if err != nil || !link.AdminUp {
		return ServingEvidence{}, checkf(ReasonIsolation, "quarantine interface %s up_verify_failed", iface)
	}
	ev, err := verifyServing(ctx, ex, m, assignments, true)
	if err != nil {
		return ev, err
	}
	if err := ex.NftBatch(ctx, fmt.Sprintf("delete set inet fvpn %q\n", quarantinePrefix+iface)); err != nil {
		return ev, checkf(ReasonIsolation, "quarantine marker clear failed")
	}
	// Confirm the marker deletion; a successful command alone is not evidence.
	state, err := loadState(ctx, ex)
	if err != nil || len(quarantinedLinks(state.Sets)) > 0 {
		return ev, checkf(ReasonIsolation, "quarantine marker clear verification failed")
	}
	ev.Quarantined, ev.QuarantinedLinks = false, nil
	return ev, nil
}
