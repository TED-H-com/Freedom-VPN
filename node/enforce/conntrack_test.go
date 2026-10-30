package enforce

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"
)

func TestApply_ConntrackFailureKeepsServing(t *testing.T) {
	m := smallModel()
	revoked := mustAddr("100.64.1.1")
	m.Revoke = []netip.Addr{revoked}
	f := servingExec(m)
	var deletes []string
	f.handler = conntrackHandler(m, f, func(key string) ([]byte, error) {
		deletes = append(deletes, key)
		return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "exec: conntrack: executable file not found in $PATH"}
	})
	_, err := Apply(context.Background(), f, m, nil)
	var ce *CheckError
	if !errors.As(err, &ce) || ce.Reason != ReasonIsolation || ce.Detail != "conntrack delete failed after the serving ruleset was verified" {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(err.Error(), revoked.String()) || strings.Contains(err.Error(), "conntrack -D") {
		t.Fatalf("error names the address or the command: %v", err)
	}
	for _, script := range f.nft {
		if strings.Contains(script, "fvpn_baseline=1") {
			t.Fatal("baseline replaced the verified ruleset")
		}
	}
	if len(deletes) != 1 {
		t.Fatalf("deletes=%v", deletes)
	}
	_, err = Apply(context.Background(), f, m, nil)
	if !errors.As(err, &ce) || ce.Reason != ReasonIsolation || ce.Detail != "conntrack delete failed after the serving ruleset was verified" {
		t.Fatalf("retry err=%v", err)
	}
	if len(deletes) != 2 {
		t.Fatalf("retry did not delete again: %v", deletes)
	}
	if strings.Contains(f.nft[len(f.nft)-1], "fvpn_baseline=1") {
		t.Fatal("retry installed the baseline")
	}
}

func TestApply_ConntrackEmptyIsSuccess(t *testing.T) {
	m := smallModel()
	m.Revoke = []netip.Addr{mustAddr("100.64.1.1")}
	f := servingExec(m)
	var deletes []string
	f.handler = conntrackHandler(m, f, func(key string) ([]byte, error) {
		deletes = append(deletes, key)
		return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "conntrack v1.4.7 (conntrack-tools): 0 flow entries have been deleted."}
	})
	if _, err := Apply(context.Background(), f, m, nil); err != nil {
		t.Fatal(err)
	}
	if len(deletes) != 2 {
		t.Fatalf("deletes=%v", deletes)
	}
	for _, script := range f.nft {
		if strings.Contains(script, "fvpn_baseline=1") {
			t.Fatal("empty conntrack installed the baseline")
		}
	}
}

func TestRequireConntrack(t *testing.T) {
	if err := RequireConntrack(nil); err == nil || ReasonCode(err) != ReasonIsolation || !strings.Contains(err.Error(), "conntrack is not installed") {
		t.Fatalf("nil executor: %v", err)
	}
	missing := &fakeExec{lookPath: func(string) (string, error) {
		return "", fmt.Errorf("exec: conntrack: executable file not found in $PATH")
	}}
	err := RequireConntrack(missing)
	if ReasonCode(err) != ReasonIsolation || err.Error() != "ISOLATION_POLICY_INVALID: conntrack is not installed" {
		t.Fatalf("missing: %v", err)
	}
	if strings.Contains(err.Error(), "PATH") || strings.Contains(err.Error(), "/usr") {
		t.Fatalf("missing error names a path: %v", err)
	}
	blank := &fakeExec{lookPath: func(string) (string, error) { return "  ", nil }}
	if err := RequireConntrack(blank); ReasonCode(err) != ReasonIsolation {
		t.Fatalf("blank path: %v", err)
	}
	if err := RequireConntrack(&fakeExec{}); err != nil {
		t.Fatal(err)
	}
}

func servingExec(m Model) *fakeExec {
	f := &fakeExec{}
	f.handler = conntrackHandler(m, f, func(key string) ([]byte, error) {
		return nil, fmt.Errorf("unexpected %s", key)
	})
	return f
}

func conntrackHandler(m Model, f *fakeExec, onConntrack func(string) ([]byte, error)) func(string, []string) ([]byte, error) {
	return func(name string, args []string) ([]byte, error) {
		key := name + " " + strings.Join(args, " ")
		if name == "conntrack" {
			return onConntrack(key)
		}
		if strings.Contains(key, "nft -j list") {
			// "verify" is the listing Apply just installed. Anything else is
			// loadState, including the next Apply after a conntrack failure.
			if f.phase == "verify" {
				f.phase = "qos"
				plan, err := Render(m, State{}, TCState{})
				if err != nil {
					return nil, err
				}
				return fixtureServing(m, plan.Assignments), nil
			}
			f.phase = "load"
			if !f.nftApplied {
				return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "Error: No such file or directory"}
			}
			plan, err := Render(m, State{}, TCState{})
			if err != nil {
				return nil, err
			}
			return fixtureServing(m, plan.Assignments), nil
		}
		if strings.Contains(key, "tc -j qdisc") || strings.Contains(key, "tc -j class") {
			if f.phase != "qos" {
				return []byte(`[]`), nil
			}
			plan, err := Render(m, State{}, TCState{})
			if err != nil {
				return nil, err
			}
			pair := fixtureQoS(m, plan.Assignments)
			if strings.Contains(key, "qdisc") {
				return pair[m.Datapath.Uplink][0], nil
			}
			return pair[m.Datapath.Uplink][1], nil
		}
		if strings.Contains(key, "qdisc del") {
			return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "No such file or directory"}
		}
		return nil, fmt.Errorf("unexpected %s", key)
	}
}
