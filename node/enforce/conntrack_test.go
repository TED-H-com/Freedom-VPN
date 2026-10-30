package enforce

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"
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
	res, err := Apply(context.Background(), f, m, nil)
	var ce *CheckError
	if !res.Verified || !IsConntrackDeleteFailure(err) || !errors.As(err, &ce) || ce.Reason != ReasonIsolation || ce.Detail != "conntrack delete failed after the serving ruleset was verified" {
		t.Fatalf("verified=%v err=%v", res.Verified, err)
	}
	var del *ConntrackDeleteError
	if !errors.As(err, &del) || del.Pending != 1 || del.Pending != len(del.Failed) || !slices.Equal(del.Failed, []netip.Addr{revoked}) {
		t.Fatalf("pending: %#v err=%v", del, err)
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
	res, err = Apply(context.Background(), f, m, nil)
	if !res.Verified || !IsConntrackDeleteFailure(err) || !errors.As(err, &ce) || ce.Reason != ReasonIsolation || ce.Detail != "conntrack delete failed after the serving ruleset was verified" {
		t.Fatalf("retry verified=%v err=%v", res.Verified, err)
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
	res, err := Apply(context.Background(), f, m, nil)
	if err != nil || !res.Verified || IsConntrackDeleteFailure(err) {
		t.Fatalf("verified=%v err=%v", res.Verified, err)
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
	if err := RequireConntrack(nil); err == nil || ReasonCode(err) != ReasonIsolation || !strings.Contains(err.Error(), "conntrack is not installed") || IsConntrackDeleteFailure(err) {
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

func TestApply_ConntrackPendingCountsRemaining(t *testing.T) {
	m := smallModel()
	first := mustAddr("100.64.1.1")
	second := mustAddr("100.64.1.2")
	m.Revoke = []netip.Addr{first, second}
	f := servingExec(m)
	f.handler = conntrackHandler(m, f, func(key string) ([]byte, error) {
		return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "exec: conntrack: executable file not found in $PATH"}
	})
	res, err := Apply(context.Background(), f, m, nil)
	var del *ConntrackDeleteError
	if !res.Verified || !IsConntrackDeleteFailure(err) || !errors.As(err, &del) || del.Pending != 2 || del.Pending != len(del.Failed) || !slices.Equal(del.Failed, []netip.Addr{first, second}) {
		t.Fatalf("verified=%v pending=%v err=%v", res.Verified, del, err)
	}
	if strings.Contains(err.Error(), first.String()) || strings.Contains(err.Error(), second.String()) {
		t.Fatalf("error names an address: %v", err)
	}
}

func TestApply_ConntrackFailedListsUnsuccessfulAddresses(t *testing.T) {
	first := mustAddr("100.64.1.1")
	second := mustAddr("100.64.1.2")
	partial := smallModel()
	partial.Revoke = []netip.Addr{first, second}
	f := servingExec(partial)
	f.handler = conntrackHandler(partial, f, func(key string) ([]byte, error) {
		if strings.Contains(key, second.String()) {
			return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "exec: conntrack: executable file not found in $PATH"}
		}
		return nil, nil
	})
	res, err := Apply(context.Background(), f, partial, nil)
	var del *ConntrackDeleteError
	if !res.Verified || !IsConntrackDeleteFailure(err) || !errors.As(err, &del) || del.Pending != 1 || !slices.Equal(del.Failed, []netip.Addr{second}) {
		t.Fatalf("partial verified=%v del=%#v err=%v", res.Verified, del, err)
	}
	if del.Pending != len(del.Failed) {
		t.Fatalf("pending %d failed %v", del.Pending, del.Failed)
	}
	if err.Error() != "ISOLATION_POLICY_INVALID: conntrack delete failed after the serving ruleset was verified" || strings.Contains(err.Error(), first.String()) || strings.Contains(err.Error(), second.String()) {
		t.Fatalf("error text: %v", err)
	}
	var ce *CheckError
	if !errors.As(err, &ce) || ce.Reason != ReasonIsolation || ce.Detail != "conntrack delete failed after the serving ruleset was verified" {
		t.Fatalf("unwrap: %v", err)
	}

	full := smallModel()
	full.Revoke = []netip.Addr{first, second}
	ff := servingExec(full)
	ff.handler = conntrackHandler(full, ff, func(key string) ([]byte, error) {
		return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "exec: conntrack: executable file not found in $PATH"}
	})
	res, err = Apply(context.Background(), ff, full, nil)
	if !res.Verified || !IsConntrackDeleteFailure(err) || !errors.As(err, &del) || del.Pending != 2 || !slices.Equal(del.Failed, []netip.Addr{first, second}) {
		t.Fatalf("full verified=%v del=%#v err=%v", res.Verified, del, err)
	}
	if strings.Contains(err.Error(), first.String()) || strings.Contains(err.Error(), second.String()) {
		t.Fatalf("full error names an address: %v", err)
	}

	okModel := smallModel()
	okModel.Revoke = []netip.Addr{first, second}
	okExec := servingExec(okModel)
	okExec.handler = conntrackHandler(okModel, okExec, func(string) ([]byte, error) {
		return nil, nil
	})
	res, err = Apply(context.Background(), okExec, okModel, nil)
	if err != nil || !res.Verified || IsConntrackDeleteFailure(err) {
		t.Fatalf("success verified=%v err=%v", res.Verified, err)
	}
}

func TestApply_VerifyServingFailureIsNotConntrack(t *testing.T) {
	m := smallModel()
	m.Revoke = []netip.Addr{mustAddr("100.64.1.1")}
	f := &fakeExec{}
	listed := 0
	f.handler = func(name string, args []string) ([]byte, error) {
		key := name + " " + strings.Join(args, " ")
		if strings.Contains(key, "nft -j list") {
			listed++
			if listed == 1 {
				return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "Error: No such file or directory"}
			}
			return []byte(`{"nftables":[{"table":{"family":"inet","name":"fvpn"}}]}`), nil
		}
		if strings.Contains(key, "tc -j qdisc show") || strings.Contains(key, "tc -j class show") {
			return []byte(`[]`), nil
		}
		if strings.Contains(key, "qdisc del") {
			return nil, &CmdError{Cmd: key, Err: fmt.Errorf("exit"), Out: "RTNETLINK answers: No such file or directory"}
		}
		return nil, fmt.Errorf("unexpected %s", key)
	}
	res, err := Apply(context.Background(), f, m, nil)
	if err == nil || res.Verified || IsConntrackDeleteFailure(err) {
		t.Fatalf("verified=%v err=%v", res.Verified, err)
	}
	if len(f.nft) < 2 || !strings.Contains(f.nft[len(f.nft)-1], "fvpn_baseline=1") {
		t.Fatal("baseline was not installed")
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
