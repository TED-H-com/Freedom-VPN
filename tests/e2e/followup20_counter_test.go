//go:build e2e

package e2e

import (
	"context"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestE2E_Followup20CounterExpiry(t *testing.T) {
	if os.Getenv("FVPN_E2E_COUNTER20") != "1" {
		t.Skip("set FVPN_E2E_COUNTER20=1 after Followup20Grants on the retained stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	head, _ := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	id := uuid.NewString()
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup20", id)}
	defer r.finish()
	s := &Stack{lab: l}
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	member, e := s.Login(ctx, "e2e-member", "member")
	if e != nil {
		t.Fatal(e)
	}
	db, e := l.db(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close(ctx)
	memberID := uuid.MustParse(member.MemberID)
	var device, plan uuid.UUID
	e = db.QueryRow(ctx, `SELECT d.id,e.plan_revision_id FROM member_devices d JOIN entitlements e ON e.member_id=d.member_id WHERE d.member_id=$1 AND d.status='authorized' LIMIT 1`, memberID).Scan(&device, &plan)
	if e != nil {
		t.Fatal(e)
	}
	rec := Record{ID: "finite-expiry-reader-diagnostics", Title: "F10-02 persistent finite expiry with every reader category and enforcement heartbeat", Status: StatusFail, Spec: []string{"AT02", "F10-02", "§9.2"}}
	defer func() { r.record(rec) }()
	deadline := time.Now().UTC().Add(70 * time.Second).Truncate(time.Second)
	change, e := s.ReplaceFiniteGrant(ctx, BrowserSession{Client: admin.http, CSRF: admin.csrf}, memberID, plan, deadline)
	noteJSON(&rec, "finite_grant", change)
	if e != nil {
		recPathNote(&rec)("error", e.Error())
		return
	}
	_, e = s.StartEstablishedTransports(ctx, device)
	if e != nil {
		recPathNote(&rec)("error", e.Error())
		return
	}
	defer func() { _ = s.StopEstablishedTransports(context.WithoutCancel(ctx)) }()
	type tick struct {
		At       time.Time             `json:"at"`
		Status   MemberTransportStatus `json:"status"`
		Counters DeviceCounters        `json:"counters"`
		ReadOK   bool                  `json:"read_ok"`
	}
	ticks := []tick{}
	categories := map[string]int{}
	all := true
	load, _ := l.docker(l.node, "sh", "-c", "awk '{print $1,$2,$3}' /proc/loadavg")
	recPathNote(&rec)("host_load_from_owned_namespace", strings.TrimSpace(string(load)))
	for time.Now().Before(deadline.Add(6 * time.Second)) {
		status, se := s.EstablishedStatus(ctx)
		counter, ce := s.DeviceByteCounters(ctx, device)
		ticks = append(ticks, tick{time.Now().UTC(), status, counter, se == nil && ce == nil})
		categories[counter.Diagnostic.Category]++
		all = all && se == nil && ce == nil
		time.Sleep(250 * time.Millisecond)
	}
	noteJSON(&rec, "samples", ticks)
	noteJSON(&rec, "counter_categories", categories)
	noteJSON(&rec, "event_at", deadline)
	lastTCP, lastUDP := ticks[0].At, ticks[0].At
	for i := 1; i < len(ticks); i++ {
		if ticks[i].Status.TCPEchoes > ticks[i-1].Status.TCPEchoes {
			lastTCP = ticks[i].At
		}
		if ticks[i].Status.UDPReplies > ticks[i-1].Status.UDPReplies {
			lastUDP = ticks[i].At
		}
	}
	noteJSON(&rec, "last_reply_observed", map[string]any{"tcp": lastTCP, "udp": lastUDP, "all_reads_ok": all, "samples": len(ticks)})
	final := ticks[len(ticks)-1]
	stable := false
	for i := len(ticks) - 2; i >= 0; i-- {
		if final.At.Sub(ticks[i].At) >= 2*time.Second {
			a := ticks[i].Counters
			b := final.Counters
			stable = (!a.Present && !b.Present) || (a.Present && b.Present && a.Upload == b.Upload && a.Download == b.Download)
			break
		}
	}
	if all && stable && final.Status.Running && final.Status.UDPAttempts > ticks[0].Status.UDPAttempts && !lastTCP.After(deadline.Add(12*time.Second)) && !lastUDP.After(deadline.Add(12*time.Second)) {
		rec.Status = StatusPass
	}
	// The recorded deadline is the product finite-grant deadline. Heartbeat is
	// sampled before/after each nft command; it is not an exact nft commit stamp.
}
