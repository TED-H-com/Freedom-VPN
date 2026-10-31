package enforce

import (
	"context"
	"strings"
	"testing"
)

// TestReviewR15_ApplyListsTheRulesetOnceBeforeVerified counts nft -j list
// ruleset across one Apply. The serving heartbeat is written from the
// engine after Apply returns; this process does not read the ruleset again
// before that heartbeat. A change that lands after this read is the next
// Step's Apply, one heartbeat later.
func TestReviewR15_ApplyListsTheRulesetOnceBeforeVerified(t *testing.T) {
	m := smallModel()
	m.Revoke = nil
	var ruleset, table int
	f := servingExec(m)
	inner := f.handler
	f.handler = func(name string, args []string) ([]byte, error) {
		if name == "nft" && len(args) >= 3 && args[0] == "-j" && args[1] == "list" {
			switch args[2] {
			case "ruleset":
				ruleset++
			case "table":
				table++
			}
		}
		return inner(name, args)
	}
	res, err := Apply(context.Background(), f, m, nil)
	if err != nil || !res.Verified {
		t.Fatalf("verified=%v err=%v", res.Verified, err)
	}
	if ruleset != 1 {
		t.Fatalf("Apply issued nft -j list ruleset %d times, want 1", ruleset)
	}
	if table < 1 {
		t.Fatalf("Apply did not list table inet fvpn (%d)", table)
	}
	if strings.Contains(strings.Join(f.nft, "\n"), "list ruleset") {
		t.Fatal("the nft script itself listed the ruleset; the count above would be short")
	}
}
