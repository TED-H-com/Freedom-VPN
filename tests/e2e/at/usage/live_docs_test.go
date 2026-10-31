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
	fmt.Fprintf(&summary, "# Usage acceptance — Follow-up 6\n\nRun `%s`; harness `%s` (dirty `%t`); image build `%v`; origin/main product source `%v`. Stack `fvpn-w4-usage-*`, port base `48000`.\n\n", s.runID, s.build.GitCommit, s.build.GitDirty, s.firstServing["image_build_commit"], s.firstServing["image_origin_main_commit"])
	summary.WriteString("| Case | Status | Evidence | Live result |\n|---|---|---|---|\n")
	for _, r := range records {
		fmt.Fprintf(&summary, "| %s | %s | [%s.json](%s.json) / [%s.md](%s.md) | %s |\n", r.ID, r.Status, r.ID, r.ID, r.ID, r.ID, strings.ReplaceAll(r.Actual, "|", "\\|"))
	}
	timing.WriteString("## D-067 comparison\n\nThe previous blocked run measured lease issuance because serving never applied. This run records issuance and serving read-back separately; the original <=65000 ms bound remains on serving read-back. Agent-core follow-up 25 is in the image source. Devstack #240 measured **80622 ms** on its separate path.\n\n| Cycle | Follow-up 3 ready → lease issued | This run ready → lease issued | Ready → serving read-back | Ready → counted flow | Ready observation | Populated lease |\n|---|---|---|---|---|---|---|\n")
	for i, c := range []d067CycleMeasurement{s.d067BringUp, s.d067Restart} {
		old := []int{5021, 5025}[i]
		fmt.Fprintf(&timing, "| %s | %d ms | %d ms | %d ms | %d ms | `%s` at `%s` | `%s` at `%s` |\n", c.Cycle, old, c.ReadyToPopulatedLeaseIssuedMs, c.ReadyToPopulatedLeaseMs, c.ReadyToFirstCountedByteMs, c.ReadyObservationReportID, c.ReadyObservationReceivedAt, c.PopulatedLeaseID, c.PopulatedLeaseIssuedAt)
	}
	timing.WriteString("\nBoth provisional observations require `AWAITING_MEMBER_GRANTS` on `lease`, `qos`, and `source_binding`, exactly one `errors[]` entry (`stage=lease`), and zero `LEASE_EXPIRED`. Full lease/observation IDs and checks are in [AT16.json](AT16.json).\n")
	fmt.Fprintf(&summary, "\n%s\n## Fair-Use convergence\n\n| Trigger | Clearing observation | Trigger → applying false | Tier 2 installed on both exits |\n|---|---|---|---|\n", timing.String())
	for _, tr := range []fupTransitionMeasurement{s.fupActivationTransition, s.fupUsageIngestTransition} {
		fmt.Fprintf(&summary, "| %s | `%s` | %d ms | %t |\n", tr.Transition, tr.ClearingObservationReportID, tr.WallClockMsToApplyingFalse, tr.BothExitsInstalledTier2TC)
	}
	summary.WriteString("\nThe usage-threshold timer starts with real traffic and includes the normal sample/upload cadence; the activation timer starts with policy activation. Follow-up 3 used synthetic batch/ACK fixtures (5387 / 6151 ms), so those timings are not identical paths.\n")
	r := s.incarnationRetention
	fmt.Fprintf(&summary, "\n## ENR-03 and UTC boundary\n\nNode `%s`: retired incarnation `%s`, new incarnation `%s`; old usage fixed `%t`; member total unchanged at reinstall `%t`; new traffic accumulated `%t`. Member totals: `%s` → `%s` → `%s` bytes. Detailed per-incarnation bytes and timestamps are in [AT19.json](AT19.json). [AT17.json](AT17.json) re-runs the UTC split (1001/2003 bytes → October 500/1001 and November 501/1002), late events and clock rollback.\n", r.NodeID, r.OldIncarnationID, r.NewIncarnationID, r.OldIncarnationUsageFixed, r.MemberTotalUnchangedOnReenroll, r.NewTrafficAccumulatedSameMonth, r.DBPeriodTotalsBeforeReinstall["total_bytes"], r.DBPeriodTotalsAfterReinstall["total_bytes"], r.DBPeriodTotalsAfterNewTraffic["total_bytes"])
	summary.WriteString("\nPrior runs: [Follow-up 5](history/follow-up-5/ARCHIVE.md), [Follow-up 4](history/follow-up-4/ARCHIVE.md), [Follow-up 3](history/followup3/summary.md), [Follow-up 2](history/followup2/summary.md), [initial](history/initial/).\n")
	report := s.firstServing["observation"].(map[string]any)
	fix := fmt.Sprintf("## Which serving fix engaged\n\n%v. The first real serving observation `%v` (received `%v`) is preserved in each case's `first_serving_apply.observation`, alongside `serving 1`, the admitted-device counter deltas and the ruleset read-back. %v.\n\n```nft\n%v```\n", s.firstServing["foreign_rule_classification"], report["report_id"], s.firstServing["received_at"], s.firstServing["observation_classification_field"], s.firstServing["nft_ruleset_excerpt"])
	run := "# Run usage acceptance\n\n```bash\nmkdir -p .agent/tmp\nexport TMPDIR=\"$PWD/.agent/tmp\" GOTMPDIR=\"$PWD/.agent/tmp\"\n# Before stack startup and the live suite, wait while two other suites are active.\npgrep -af 'gates.sh|mk/at-' || true\nmake -f mk/at-usage.mk at-usage-lint at-usage-test at-usage-build\nbash tests/e2e/at/usage/stack.sh up\nFVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/at-usage.mk at-usage\nbash tests/e2e/at/usage/stack.sh down\n```\n\nThe lane wrapper calls the shared driver and scopes image tags to the owned prefix. `e2e.PrepareServingBaseline` builds/signs/registers real amd64 and arm64 releases. Member group, plan `exit_group_ids`, publication `group_ids`, grants and native admission all use product APIs. A real serving heartbeat and counted forward flow precede acceptance.\n\nPostgreSQL access to the live stack is read-only. The existing mechanism fixtures use `internal/db/dbtest` to create/drop isolated databases on the assigned lab PostgreSQL when `FVPN_TEST_DATABASE_URL` is configured; their SQL fixture setup is unchanged from Follow-up 3. Audited usage adjustment and ledger recovery use `fvpn-usage-admin`. AT15 deliberately submits signed mTLS batch fixtures after pausing the agent. D-067 retains the prior mTLS applying-observation setup; its provisional/serving observations are real agent reports.\n\nFollow-up 3's synthetic ACK workaround is removed: Fair-Use convergence now waits for actual agent apply and kernel tc read-back. Counted traffic is sampled/uploaded by the agent, including both incarnations. AT14 uses the shared ingress injector plus a temporary node-netns private-destination /32 route over fvup0 (removed immediately) to exercise ctdeny; no default route or foreign nft table is changed. The same component checks remain; deterministic month-boundary/clock fixtures do not change the host clock.\n\n"
	run += summary.String() + "\n" + fix
	requests.WriteString("# Requests — Follow-up 6\n\n## Closed: FLOWTABLE_BYPASS (AT14–AT19)\n\n[PR #240](https://github.com/TED-H-com/Freedom-VPN/pull/240) removes Docker DNS NAT before the lab agent starts. [PR #241](https://github.com/TED-H-com/Freedom-VPN/pull/241) adds D-079 loopback classification. Both are in this run; [PR #246](https://github.com/TED-H-com/Freedom-VPN/pull/246) previously established the lab-only result. The old blocked observations remain in [history/followup3/requests.md](history/followup3/requests.md).\n\n")
	requests.WriteString(fix)
	requests.WriteString("\n## Current requests\n\n")
	blocked := false
	for _, r := range records {
		for _, b := range r.Blockers {
			blocked = true
			fmt.Fprintf(&requests, "- %s: %s\n", r.ID, b)
		}
	}
	requests.WriteString("Follow-up 5 request 1 closed by [devstack 19](../../RESPONSES.md#usage--2026-10-31-follow-up-19): retained canonical payloads and origin sequence/hash/ACK are the adoption predicate. [Its eight of eight rows](../../runs/2026-10-31/followup19/usage-retention.md) are independent local retention evidence; this run adds before/after attribution in [AT20.json](AT20.json). No assertion required a file-equality fixture change.\n")
	if !blocked {
		requests.WriteString("No new requests. AT20 is the usage lane ENR-05 extension; it is not the DNS case numbered AT20 in §19. Installer hardware/platform and restore-runbook campaigns remain with their owners.\n")
	}
	requests.WriteString("\n" + timing.String())
	for _, record := range records {
		if record.ID == "AT20" {
			paragraph := "\n### For the enrollment lane\n\n" + record.Actual + " See [AT20.json](AT20.json) for the original CLI, controlled reinstall outcome, immutable payloads, central samples, stored batch replay, independent ACKs, configuration and refusal audit. This follows the devstack follow-up 19 corrected path in [RESPONSES.md](../../RESPONSES.md#usage--2026-10-31). The systemd installer node is quiesced in documented unit order; pause-agent applies only to ordinary supervised nodes. No ledger rows or retirement timestamps are edited.\n"
			requests.WriteString(paragraph)
			summary.WriteString(paragraph)
			run += paragraph
			precision := "\n### D-130 scope and known gaps\n\nAT20 observes the fresh unknown-continuity gap at ACK 1 before producing counted event 2. Its `d130_*` observations retain the actual agent payloads/batches, original member/device/period, exact kernel direction counters, member totals before/after/replay and read-only event/sample/gap interval projections plus the three installed finite `<=` constraints. Same-key replays of both saved batches preserve the stored results, including ACK 1 after the stream reaches 2. Newly minted intervals must be UTC whole microseconds with width at least 1 µs. See [agent D-130](../../../node/usage-meter.md#interval-precision-and-compatibility-d-130) and [central D-130](../../../usage/2026-10-31-d130-interval-precision.md).\n\nThis harness builds the current node source; already sealed pre-D-130 1 ns retry/recovery belongs to enrollment 9 and is not claimed here. Enrollment 9's controlled_reinstall qualification and devstack 20's F10 diagnostics are not re-measured. Physical arm64 remains unmeasured (amd64/arm64 cross-builds only). AT17 UTC boundary/clock and power-loss mechanisms retain their deterministic fixture scope, not a real clock-change or physical power-loss claim.\n\nD-118: [devstack 19 retained 8/8 canonical origin payload/hash pairs](../../runs/2026-10-31/followup19/usage-retention.md), with origin ACKs 4→4 and 3→4. [RESPONSES.md](../../RESPONSES.md#usage--2026-10-31-follow-up-19) answers follow-up 5 request 1. Our [AT20.json](AT20.json) records `held_events_before_reenrollment`, `reinstall_original_payload_bytes_equal`, `handed_off_events`, origin ACKs and `member_period_before` / `member_period_after` / `member_period_after_replay`. DB/WAL hashes may change during adoption; canonical payloads, origin sequence/hash and the independent origin ACK establish retention. Whole-file equality remains only a pre-adoption quiescence check. Existing assertions are unchanged.\n"
			summary.WriteString(precision)
			run += precision
		}
	}
	for name, content := range map[string]string{"summary.md": summary.String(), "README.md": summary.String(), "RUN.md": run, "operator-run.md": run, "requests.md": requests.String()} {
		if err := os.WriteFile(filepath.Join(s.evidence, name), []byte(Redact(content, s.extraRedactions...)), 0o644); err != nil {
			s.t.Fatal(err)
		}
	}
}
