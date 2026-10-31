//go:build e2e

package usage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *liveSuite) writeDocs(records []Record) {
	var summary, timing, requests strings.Builder
	fmt.Fprintf(&summary, "# Usage acceptance — Follow-up 4\n\nRun `%s`; harness `%s` (dirty `%t`); image build `%v`; origin/main product source `%v`. Stack `fvpn-w4-usage-*`, port base `48000`.\n\n", s.runID, s.build.GitCommit, s.build.GitDirty, s.firstServing["image_build_commit"], s.firstServing["image_origin_main_commit"])
	summary.WriteString("| Case | Status | Evidence | Live result |\n|---|---|---|---|\n")
	for _, r := range records {
		fmt.Fprintf(&summary, "| %s | %s | [%s.json](%s.json) / [%s.md](%s.md) | %s |\n", r.ID, r.Status, r.ID, r.ID, r.ID, r.ID, strings.ReplaceAll(r.Actual, "|", "\\|"))
	}
	timing.WriteString("## D-067 comparison\n\nThe previous blocked run measured lease issuance because serving never applied. This run records issuance and serving read-back separately; the original <=65000 ms bound remains on serving read-back. Agent-core follow-up 20 was not in the image source. Devstack #240 measured **80622 ms** on its separate path.\n\n| Cycle | Follow-up 3 ready → lease issued | This run ready → lease issued | Ready → serving read-back | Ready → counted flow | Ready observation | Populated lease |\n|---|---|---|---|---|---|---|\n")
	for i, c := range []d067CycleMeasurement{s.d067BringUp, s.d067Restart} {
		old := []int{5021, 5025}[i]
		fmt.Fprintf(&timing, "| %s | %d ms | %d ms | %d ms | %d ms | `%s` at `%s` | `%s` at `%s` |\n", c.Cycle, old, c.ReadyToPopulatedLeaseIssuedMs, c.ReadyToPopulatedLeaseMs, c.ReadyToFirstCountedByteMs, c.ReadyObservationReportID, c.ReadyObservationReceivedAt, c.PopulatedLeaseID, c.PopulatedLeaseIssuedAt)
	}
	timing.WriteString("\nBoth provisional observations require `AWAITING_MEMBER_GRANTS` on `lease`, `qos`, and `source_binding`, exactly one `errors[]` entry (`stage=lease`), and zero `LEASE_EXPIRED`. Full lease/observation IDs and checks are in [AT16.json](AT16.json).\n")
	fmt.Fprintf(&summary, "\n%s\n## Fair-Use convergence\n\n| Trigger | Clearing observation | Applying true → false | Tier 2 installed on both exits |\n|---|---|---|---|\n", timing.String())
	for _, tr := range []fupTransitionMeasurement{s.fupActivationTransition, s.fupUsageIngestTransition} {
		fmt.Fprintf(&summary, "| %s | `%s` | %d ms | %t |\n", tr.Transition, tr.ClearingObservationReportID, tr.WallClockMsToApplyingFalse, tr.BothExitsInstalledTier2TC)
	}
	r := s.incarnationRetention
	fmt.Fprintf(&summary, "\n## ENR-03 and UTC boundary\n\nNode `%s`: retired incarnation `%s`, new incarnation `%s`; old usage fixed `%t`; member total unchanged at reinstall `%t`; new traffic accumulated `%t`. Member totals: `%s` → `%s` → `%s` bytes. Detailed per-incarnation bytes and timestamps are in [AT19.json](AT19.json). [AT17.json](AT17.json) re-runs the UTC split (1001/2003 bytes → October 500/1001 and November 501/1002), late events and clock rollback.\n", r.NodeID, r.OldIncarnationID, r.NewIncarnationID, r.OldIncarnationUsageFixed, r.MemberTotalUnchangedOnReenroll, r.NewTrafficAccumulatedSameMonth, r.DBPeriodTotalsBeforeReinstall["total_bytes"], r.DBPeriodTotalsAfterReinstall["total_bytes"], r.DBPeriodTotalsAfterNewTraffic["total_bytes"])
	summary.WriteString("\nPrior runs: [Follow-up 3](history/followup3/summary.md), [Follow-up 2](history/followup2/summary.md), [initial](history/initial/).\n")
	fix := fmt.Sprintf("## Which serving fix engaged\n\n%v. The first real serving observation is preserved in each case's `first_serving_apply.observation`, alongside `serving 1`, the admitted-device counter deltas and the ruleset read-back. %v.\n\n```nft\n%v```\n", s.firstServing["foreign_rule_classification"], s.firstServing["observation_classification_field"], s.firstServing["nft_ruleset_excerpt"])
	run := "# Run usage acceptance\n\n```bash\nmake -f mk/at-usage.mk at-usage-lint at-usage-test at-usage-build\nbash tests/e2e/at/usage/stack.sh up\nFVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/at-usage.mk at-usage\nbash tests/e2e/at/usage/stack.sh down\n```\n\nThe lane wrapper calls the shared driver and scopes image tags to the owned prefix. `e2e.PrepareServingBaseline` builds/signs/registers real amd64 and arm64 releases. Member group, plan `exit_group_ids`, publication `group_ids`, grants and native admission all use product APIs. A real serving heartbeat and counted forward flow precede acceptance.\n\nPostgreSQL access to the live stack is read-only. The existing mechanism fixtures use `internal/db/dbtest` to create/drop isolated databases on the assigned lab PostgreSQL when `FVPN_TEST_DATABASE_URL` is configured; their SQL fixture setup is unchanged from Follow-up 3. Audited usage adjustment and ledger recovery use `fvpn-usage-admin`. AT15 deliberately submits signed mTLS batch fixtures after pausing the agent. D-067 retains the prior mTLS applying-observation setup; its provisional/serving observations are real agent reports.\n\nFollow-up 3's synthetic ACK workaround is removed: Fair-Use convergence now waits for actual agent apply and kernel tc read-back. Counted traffic is sampled/uploaded by the agent, including both incarnations. The same component checks remain; deterministic month-boundary/clock fixtures do not change the host clock.\n\n"
	run += summary.String() + "\n" + fix
	requests.WriteString("# Requests — Follow-up 4\n\n## Closed: FLOWTABLE_BYPASS (AT14–AT19)\n\n[PR #240](https://github.com/TED-H-com/Freedom-VPN/pull/240) removes Docker DNS NAT before the lab agent starts. [PR #241](https://github.com/TED-H-com/Freedom-VPN/pull/241) adds D-079 loopback classification. Both are in this run; [PR #246](https://github.com/TED-H-com/Freedom-VPN/pull/246) previously established the lab-only result. The old blocked observations remain in [history/followup3/requests.md](history/followup3/requests.md).\n\n")
	requests.WriteString(fix)
	requests.WriteString("\n## Current requests\n\n")
	blocked := false
	for _, r := range records {
		for _, b := range r.Blockers {
			blocked = true
			fmt.Fprintf(&requests, "- %s: %s\n", r.ID, b)
		}
	}
	if !blocked {
		requests.WriteString("None. D-067 timings are comparative evidence for agent-core, not a verdict on its in-progress follow-up 20. ENR-05/06 and the restore runbook remain outside this lane.\n")
	}
	requests.WriteString("\n" + timing.String())
	for name, content := range map[string]string{"summary.md": summary.String(), "README.md": summary.String(), "RUN.md": run, "operator-run.md": run, "requests.md": requests.String()} {
		if err := os.WriteFile(filepath.Join(s.evidence, name), []byte(Redact(content, s.extraRedactions...)), 0o644); err != nil {
			s.t.Fatal(err)
		}
	}
}
