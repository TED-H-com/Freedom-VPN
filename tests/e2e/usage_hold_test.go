//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"time"
)

func (r *runner) exportsUsageHold(ctx context.Context, s *Stack) {
	rec := Record{ID: "usage-hold", Title: "Withhold usage acknowledgements and read the ledger snapshot", Spec: []string{"§10", "ENR-05"}, Status: StatusFail}
	note := recPathNote(&rec)
	defer func() { r.record(rec) }()
	if err := s.Service(ctx, "stop", ""); err != nil {
		note("agent_stop", err.Error())
		return
	}
	defer func() { _ = s.Service(context.Background(), "start", "/usr/local") }()
	first, err := s.UsageLedgerSnapshotBytes(ctx, "node")
	if err != nil {
		note("snapshot", err.Error())
		return
	}
	second, err := s.UsageLedgerSnapshotBytes(ctx, "node")
	if err != nil || !bytes.Equal(first, second) {
		note("snapshot_repeat", "diverged")
		return
	}
	snap, err := s.UsageLedgerSnapshot(ctx, "node")
	if err != nil {
		note("snapshot_parse", err.Error())
		return
	}
	note("schema_version", itoa(snap.SchemaVersion))
	note("unacked_count", itoa(snap.UnackedCount))
	note("snapshot_bytes", itoa(len(first)))
	if err = s.HoldUsageAcknowledgements(ctx, "node", true); err != nil {
		note("hold", err.Error())
		return
	}
	defer func() { _ = s.HoldUsageAcknowledgements(context.Background(), "node", false) }()
	held, heldBody := s.probeUsageRoute(ctx, "node", httpPostUsage)
	note("held_status", itoa(held))
	note("held_body", clipText(heldBody, 240))
	if held != 503 || !strings.Contains(heldBody, `"code":"DEPENDENCY_UNAVAILABLE"`) {
		note("owner", "w3-devstack-e2e: usage proxy did not withhold the batch route")
		return
	}
	otherStatus, other := s.probeUsageRoute(ctx, "node", httpGetOther)
	if otherStatus == 0 || strings.Contains(other, "usage acknowledgements withheld") {
		note("other_route", "withhold body")
		return
	}
	note("other_route", "not withheld")
	if err = s.HoldUsageAcknowledgements(ctx, "node", false); err != nil {
		note("release", err.Error())
		return
	}
	released, releasedBody := s.probeUsageRoute(ctx, "node", httpPostUsage)
	note("released_status", itoa(released))
	note("released_body", clipText(releasedBody, 240))
	if released == 0 || released == 503 || strings.Contains(releasedBody, "usage acknowledgements withheld") {
		note("owner", "w3-devstack-e2e: usage proxy still withheld after release")
		return
	}
	third, err := s.UsageLedgerSnapshotBytes(ctx, "node")
	fourth, err2 := s.UsageLedgerSnapshotBytes(ctx, "node")
	if err != nil || err2 != nil || !bytes.Equal(first, third) || !bytes.Equal(third, fourth) {
		note("snapshot_after_hold", "changed")
		return
	}
	note("knob", "network-side marker on the node resolver proxy; control-side freeze cannot be released without a new ledger epoch")
	rec.Status = StatusPass
}

// The agent produces real outbox rows while the network route is held; no row
// mutation or synthetic ACK is used to drive this proof.
func (r *runner) exportsUsageBacklog(ctx context.Context, s *Stack) {
	rec := Record{ID: "usage-hold-backlog", Title: "Usage production, stable restart snapshot and ACK release", Spec: []string{"§10.5", "AT07", "ENR-05"}, Status: StatusFail}
	note := recPathNote(&rec)
	defer func() { r.record(rec) }()
	cleanup := func() {
		c, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = s.HoldUsageAcknowledgements(c, "node", false)
		_ = s.Service(c, "start", "/usr/local")
		_ = s.StopEstablishedTransports(c)
	}
	defer cleanup()
	if err := s.HoldUsageAcknowledgements(ctx, "node", true); err != nil {
		note("hold", err.Error())
		return
	}
	initial, err := s.UsageLedgerSnapshot(ctx, "node")
	if err != nil {
		note("snapshot", err.Error())
		return
	}
	if _, err = s.StartEstablishedTransports(ctx, r.usageDevice); err != nil {
		note("transport", err.Error())
		return
	}
	wait, cancel := context.WithTimeout(ctx, 150*time.Second)
	var held UsageSnapshot
	produced := waitUntil(wait, 2*time.Second, func() bool {
		held, err = s.UsageLedgerSnapshot(wait, "node")
		return err == nil && held.UnackedCount > initial.UnackedCount
	})
	cancel()
	note("produced_while_held", boolText(produced))
	note("acked_before", initial.AckedThrough)
	note("acked_held", held.AckedThrough)
	note("unacked_held", itoa(held.UnackedCount))
	if !produced || held.AckedThrough != initial.AckedThrough {
		return
	}
	if err = s.Service(ctx, "stop", ""); err != nil {
		note("stop", err.Error())
		return
	}
	first, err := s.UsageLedgerSnapshotBytes(ctx, "node")
	second, err2 := s.UsageLedgerSnapshotBytes(ctx, "node")
	if err != nil || err2 != nil || !bytes.Equal(first, second) {
		note("snapshot_repeat", "changed")
		return
	}
	stable, err := s.UsageLedgerSnapshot(ctx, "node")
	if err != nil {
		return
	}
	if err = s.Service(ctx, "start", "/usr/local"); err != nil {
		note("restart", err.Error())
		return
	}
	resumed, err := s.UsageLedgerSnapshot(ctx, "node")
	if err != nil {
		return
	}
	preserved := resumed.AckedThrough == stable.AckedThrough
	for _, old := range stable.UnackedEvents {
		found := false
		for _, current := range resumed.UnackedEvents {
			if old.EventSeq == current.EventSeq && old.PayloadSHA256 == current.PayloadSHA256 {
				found = true
			}
		}
		preserved = preserved && found
	}
	note("restart_preserved_hashes_and_ack", boolText(preserved))
	if !preserved {
		return
	}
	_ = s.StopEstablishedTransports(ctx)
	if err = s.HoldUsageAcknowledgements(ctx, "node", false); err != nil {
		return
	}
	target, _ := strconv.ParseUint(stable.LastEventSeq, 10, 64)
	wait, cancel = context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	var released UsageSnapshot
	caught := waitUntil(wait, 2*time.Second, func() bool {
		released, err = s.UsageLedgerSnapshot(wait, "node")
		ack, _ := strconv.ParseUint(released.AckedThrough, 10, 64)
		return err == nil && ack >= target
	})
	note("acked_after_release", released.AckedThrough)
	note("released_caught_up", boolText(caught))
	if caught {
		rec.Status = StatusPass
	}
}
