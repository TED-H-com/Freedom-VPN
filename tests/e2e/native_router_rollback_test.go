//go:build e2e

package e2e

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAT02_NativeRouterRollbackEveryStep(t *testing.T) {
	const prefix = "fvpn-test-only"
	setup := [][]string{
		{"network", "create", "--ipv6", "--subnet", "192.0.2.128/28", "--subnet", "2001:db8:30:23::/64", "--opt", "com.docker.network.bridge.enable_ip_masquerade=false", "--opt", "com.docker.network.driver.mtu=1280", prefix + "-node-router-link"},
		{"run", "-d", "--name", prefix + "-node-router", "--privileged", "--network", prefix, "--ip", "172.28.30.60", "--entrypoint", "sleep", "TEST_ONLY_image", "infinity"},
		{"network", "connect", "--ip6", "2001:db8:30::60", prefix + "-node-underlay", prefix + "-node-router"},
		{"network", "connect", "--ip", "192.0.2.130", "--ip6", "2001:db8:30:23::2", prefix + "-node-router-link", prefix + "-node-router"},
		{"network", "connect", "--ip", "192.0.2.131", "--ip6", "2001:db8:30:23::3", prefix + "-node-router-link", prefix + "-member"},
		{"exec", prefix + "-node-router", "sysctl", "-w", "net.ipv4.ip_forward=1", "net.ipv6.conf.all.forwarding=1"},
		{"exec", prefix + "-node", "ip", "route", "add", "192.0.2.131/32", "via", "172.28.30.60"},
		{"exec", prefix + "-node", "ip", "-6", "route", "add", "2001:db8:30:23::3/128", "via", "2001:db8:30::60"},
		{"exec", prefix + "-member", "ip", "route", "add", "172.28.30.40/32", "via", "192.0.2.130"},
		{"exec", prefix + "-member", "ip", "-6", "route", "add", "2001:db8:30::40/128", "via", "2001:db8:30:23::2"},
	}
	undo := [][]string{
		{"network", "rm", prefix + "-node-router-link"},
		{"rm", "-fv", prefix + "-node-router"},
		{"network", "disconnect", prefix + "-node-underlay", prefix + "-node-router"},
		{"network", "disconnect", prefix + "-node-router-link", prefix + "-node-router"},
		{"network", "disconnect", prefix + "-node-router-link", prefix + "-member"},
		nil,
		{"exec", prefix + "-node", "ip", "route", "del", "192.0.2.131/32", "via", "172.28.30.60"},
		{"exec", prefix + "-node", "ip", "-6", "route", "del", "2001:db8:30:23::3/128", "via", "2001:db8:30::60"},
		{"exec", prefix + "-member", "ip", "route", "del", "172.28.30.40/32", "via", "192.0.2.130"},
		{"exec", prefix + "-member", "ip", "-6", "route", "del", "2001:db8:30::40/128", "via", "2001:db8:30:23::2"},
	}
	for fail := 0; fail <= 10; fail++ {
		for _, rollbackFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("step_%d/rollback_fails_%t", fail, rollbackFails), func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				stack := &Stack{lab: &lab{prefix: prefix, node: prefix + "-node", member: prefix + "-member", labPrefix: "172.28.30"}}
				original := errors.New("TEST ONLY setup failure")
				var calls [][]string
				failed := false
				got, err := stack.prepareNativeRouter1280(ctx, func(ctx context.Context, args ...string) ([]byte, error) {
					if args[0] == "inspect" {
						return []byte("TEST_ONLY_image\n"), nil
					}
					if ctx.Err() != nil {
						t.Fatal("rollback inherited cancellation")
					}
					calls = append(calls, append([]string(nil), args...))
					if failed {
						deadline, ok := ctx.Deadline()
						if !ok || time.Until(deadline) > 10*time.Second {
							t.Error("rollback has no bound")
						}
						if rollbackFails {
							return nil, errors.New("TEST ONLY inverse failure")
						}
					} else if len(calls)-1 == fail {
						failed = true
						cancel()
						return nil, original
					}
					return nil, nil
				})
				if fail == 10 {
					if err != nil || got == nil || !reflect.DeepEqual(calls, setup) {
						t.Fatalf("normal ten steps changed: %v %v", calls, err)
					}
					return
				}
				var setupErr *NativeRouterSetupError
				if got != nil || !errors.Is(err, original) || !errors.As(err, &setupErr) || setupErr.Step != fail || len(setupErr.Rollback) != fail || !strings.Contains(err.Error(), "rollback=") {
					t.Fatalf("missing cause/receipts: %v", err)
				}
				want := append([][]string(nil), setup[:fail+1]...)
				for i := fail - 1; i >= 0; i-- {
					if undo[i] != nil {
						want = append(want, undo[i])
					}
					row := setupErr.Rollback[fail-1-i]
					outcome := "succeeded"
					if rollbackFails {
						outcome = "failed"
					}
					if i == 5 {
						outcome = "container_removal_" + outcome
					}
					if row.Step != i || !reflect.DeepEqual(row.Command, undo[i]) || row.Outcome != outcome || row.StartedAt.Location() != time.UTC || row.FinishedAt.Before(row.StartedAt) {
						t.Fatalf("rollback receipt: %+v", row)
					}
				}
				if !reflect.DeepEqual(calls, want) {
					t.Fatalf("rollback order\ngot %v\nwant %v", calls, want)
				}
			})
		}
	}
}
