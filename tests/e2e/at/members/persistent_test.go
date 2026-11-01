//go:build e2e

package members

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

type persistentTick struct {
	At       time.Time                      `json:"at"`
	Status   labstack.MemberTransportStatus `json:"status"`
	Counters labstack.DeviceCounters        `json:"counters"`
	ReadOK   bool                           `json:"read_ok"`
}
type persistentFlow struct {
	stack   *labstack.Stack
	service string
	cancel  context.CancelFunc
	done    chan struct{}
	ticks   []persistentTick
}

func (l *live) startPersistent(r *record, service string, id uuid.UUID) *persistentFlow {
	node := "node"
	if service == "member-paid2" || service == "member-expiry2" || service == "member-suspend2" {
		node = "node2"
	}
	s, err := l.hook.MemberTransportStack(l.name(service), node)
	if err != nil {
		r.check("persistent member setup", "owned member/exit", safeError(err), "MemberTransportStack", false)
		return nil
	}
	before, e := s.DeviceByteCounters(l.ctx, id)
	_, err = s.StartEstablishedTransports(l.ctx, id)
	status, se := s.EstablishedStatus(l.ctx)
	after, ae := s.DeviceByteCounters(l.ctx, id)
	r.jsonCheck("persistent TCP/UDP positive "+service, "both authenticated echoes and both device counters increase", map[string]any{"before": before, "after": after, "status": status, "error": safeError(err)}, "StartEstablishedTransports TCP/9446 UDP/9445; DeviceByteCounters", e == nil && err == nil && se == nil && ae == nil && status.Running && status.TCPEchoes > 0 && status.UDPReplies > 0 && after.Present && after.Upload > before.Upload && after.Download > before.Download)
	if err != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(l.ctx)
	f := &persistentFlow{stack: s, service: service, cancel: cancel, done: make(chan struct{}), ticks: []persistentTick{{time.Now().UTC(), status, after, se == nil && ae == nil}}}
	go func() {
		defer close(f.done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
				st, e := s.EstablishedStatus(ctx)
				ct, e2 := s.DeviceByteCounters(ctx, id)
				if ctx.Err() != nil {
					return
				}
				f.ticks = append(f.ticks, persistentTick{time.Now().UTC(), st, ct, e == nil && e2 == nil})
			}
		}
	}()
	l.t.Cleanup(func() {
		cancel()
		<-f.done
		c, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_ = s.StopEstablishedTransports(c)
	})
	return f
}
func (l *live) finishPersistent(r *record, f *persistentFlow, event time.Time, target time.Duration) {
	if f == nil {
		return
	}
	// Keep attempts alive after the required cutoff; frozen replies are not
	// evidence if the sender was stopped by the test itself.
	if remaining := time.Until(event.Add(target + 4*time.Second)); remaining > 0 {
		timer := time.NewTimer(remaining)
		select {
		case <-timer.C:
		case <-l.ctx.Done():
		}
		timer.Stop()
	}
	f.cancel()
	<-f.done
	_ = f.stack.StopEstablishedTransports(l.ctx)
	ticks := f.ticks
	if len(ticks) < 2 {
		r.check("persistent withdrawal "+f.service, "samples", "missing", "EstablishedStatus", false)
		return
	}
	lastTCP, lastUDP := ticks[0].At, ticks[0].At
	valid := ticks[0].ReadOK
	for i := 1; i < len(ticks); i++ {
		a, b := ticks[i-1], ticks[i]
		valid = valid && b.ReadOK
		if b.Status.TCPEchoes > a.Status.TCPEchoes {
			lastTCP = b.At
		}
		if b.Status.UDPReplies > a.Status.UDPReplies {
			lastUDP = b.At
		}
	}
	final := ticks[len(ticks)-1]
	active := final.Status.Running && final.Status.UDPAttempts > ticks[0].Status.UDPAttempts
	// Counter removal is explicit, not represented as a zero-byte sample.
	stable := false
	for i := len(ticks) - 2; i >= 0; i-- {
		if final.At.Sub(ticks[i].At) >= 2*time.Second {
			a := ticks[i]
			stable = (!final.Counters.Present && !a.Counters.Present) || (final.Counters.Present && a.Counters.Present && final.Counters.Upload == a.Counters.Upload && final.Counters.Download == a.Counters.Download)
			break
		}
	}
	r.jsonCheck("persistent withdrawal "+f.service, "active TCP/UDP senders; no replies later than target; counters frozen or removed", map[string]any{"event_at": event, "target_seconds": target.Seconds(), "last_tcp_reply_observed_at": lastTCP, "last_udp_reply_observed_at": lastUDP, "tcp_seconds": lastTCP.Sub(event).Seconds(), "udp_seconds": lastUDP.Sub(event).Seconds(), "poll_interval_ms": 250, "sender_active": active, "counters_stable_or_removed": stable, "samples": ticks, "parse_failure_receipts": l.parseFailureReceipts(ticks)}, "EstablishedStatus + DeviceByteCounters until target plus 4s; timestamps bound last observed increment, not packet capture", valid && active && stable && lastTCP.Sub(event) <= target && lastUDP.Sub(event) <= target && final.At.After(event.Add(target)))
}

func (l *live) parseFailureReceipts(ticks []persistentTick) []map[string]any {
	receipts := []map[string]any{}
	for _, tick := range ticks {
		d := tick.Counters.Diagnostic
		if d.Parse == nil {
			continue
		}
		p := d.Parse
		id, e := uuid.Parse(p.ID)
		if e != nil {
			receipts = append(receipts, map[string]any{"metadata": p, "private_verified": false})
			continue
		}
		dir := filepath.Join(l.root, ".agent/tmp/counterread", id.String())
		raw, re := os.ReadFile(filepath.Join(dir, "output.raw"))
		parser, pe := os.ReadFile(filepath.Join(dir, "parser-error.txt"))
		modeOK := true
		for _, name := range []string{"output.raw", "parser-error.txt"} {
			st, e := os.Stat(filepath.Join(dir, name))
			modeOK = modeOK && e == nil
			if e == nil {
				modeOK = modeOK && st.Mode().Perm() == 0600
			}
		}
		receipts = append(receipts, map[string]any{"at": tick.At, "metadata": p, "read_ok": tick.ReadOK, "stage": d.Stage, "category": d.Category, "exit": d.Exit, "private_verified": p.Exported && re == nil && pe == nil && len(parser) > 0 && len(raw) == p.Bytes && fmt.Sprintf("%x", sha256.Sum256(raw)) == p.SHA256 && modeOK})
	}
	return receipts
}
