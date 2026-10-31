//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RelayPath is a member's NetBird path reduced to peer counts.
// Endpoint addresses and peer identifiers are not retained.
type RelayPath struct {
	Relayed int
	Direct  int
	Forced  bool
}

// ReadMemberPath reads netbird status inside the member container.
// An empty container is the lab member. The container must be that member
// or a peer this prefix started under the member name.
func (s *Stack) ReadMemberPath(ctx context.Context, container string) (RelayPath, error) {
	name, err := s.memberContainer(container)
	if err != nil {
		return RelayPath{}, err
	}
	raw, err := exec.CommandContext(ctx, "docker", "exec", name, "netbird", "status", "--json").CombinedOutput()
	relayed, direct := classifyNetbird(raw)

	if err != nil {
		return RelayPath{}, fmt.Errorf("netbird JSON status unavailable")
	}
	return RelayPath{Relayed: relayed, Direct: direct}, nil
}

// ForceMemberRelay drops UDP between the member and the exit underlay so
// hole-punch fails, then waits until status shows a relayed peer and no
// direct peer. force false deletes that table and waits for the direct path.
// The table is inet e2e_relay in the member. inet fvpn is not changed.
func (s *Stack) ForceMemberRelay(ctx context.Context, container string, force bool) (RelayPath, error) {
	name, err := s.memberContainer(container)
	if err != nil {
		return RelayPath{}, err
	}
	if s.lab.labPrefix == "" {
		return RelayPath{}, fmt.Errorf("lab prefix unavailable")
	}
	if force {
		if err = s.installRelayDrop(ctx, name); err != nil {
			return RelayPath{}, err
		}
	} else if err = s.clearRelayDrop(ctx, name); err != nil {
		return RelayPath{}, err
	}
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var path RelayPath
	ok := waitUntil(wait, 2*time.Second, func() bool {
		var readErr error
		path, readErr = s.ReadMemberPath(wait, name)
		if readErr != nil {
			return false
		}
		path.Forced = force
		if force {
			return path.Relayed >= 1 && path.Direct == 0
		}
		return path.Direct >= 1 && path.Relayed == 0
	})
	if !ok {
		return path, fmt.Errorf("member path relayed=%d direct=%d", path.Relayed, path.Direct)
	}
	return path, nil
}

func (s *Stack) memberContainer(name string) (string, error) {
	if name == "" {
		name = s.lab.member
	}
	if name == s.lab.member || strings.HasPrefix(name, s.lab.prefix+"-member-") {
		return name, nil
	}
	return "", fmt.Errorf("relay control is limited to the lab member")
}

func (s *Stack) installRelayDrop(ctx context.Context, container string) error {
	if err := s.clearRelayDrop(ctx, container); err != nil {
		return err
	}
	// The match expires even if the caller is interrupted. Only outer UDP on
	// the container underlay is held; relay TCP and the fvnb0 inner path remain.
	script := `add table inet e2e_relay
add set inet e2e_relay held { type ifname; flags timeout; elements = { "eth0" timeout 180s }; }
add chain inet e2e_relay out { type filter hook output priority 0; policy accept; }
add chain inet e2e_relay in { type filter hook input priority 0; policy accept; }
add rule inet e2e_relay out oifname @held meta l4proto udp drop
add rule inet e2e_relay in iifname @held meta l4proto udp drop
`
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", container, "nft", "-f", "-")
	cmd.Stdin = strings.NewReader(script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("relay drop: %s", clipText(string(out), 200))
	}
	return nil
}

func (s *Stack) clearRelayDrop(ctx context.Context, container string) error {
	if exec.CommandContext(ctx, "docker", "exec", container, "nft", "list", "table", "inet", "e2e_relay").Run() != nil {
		return ctx.Err()
	}
	return exec.CommandContext(ctx, "docker", "exec", container, "nft", "delete", "table", "inet", "e2e_relay").Run()
}
