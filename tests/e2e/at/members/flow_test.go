//go:build e2e

package members

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

type flowTick struct {
	At      time.Time `json:"at"`
	Receipt bool      `json:"origin_receipt"`
	Reused  bool      `json:"reused_tcp"`
}
type flowMonitor struct {
	node         string
	streamBefore timingStreamReceipt
	service      string
	cancel       context.CancelFunc
	ticks        chan flowTick
	done         chan struct{}
	first        flowTick
	persistent   *persistentFlow
}

func (l *live) startFlow(r *record, service string) *flowMonitor {
	node := "node"
	if service == "member-paid2" || service == "member-expiry2" || service == "member-suspend2" {
		node = "node2"
	}
	_, before, streamErr := l.readEnforcementTimingAt(node)
	r.jsonCheck("AT04 monitor producer stream before "+service, "whole stream parsed, including beyond 1 MiB", before, "streamed ExportContainerFile before starting the bounded monitor", streamErr == nil)
	ctx, cancel := context.WithTimeout(l.ctx, 90*time.Second)
	cmd := exec.CommandContext(ctx, "docker", "exec", l.name(service), "/tmp/at-members-probe", "monitor", "bounded")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		r.block("Controlled flow observer unavailable")
		return nil
	}
	f := &flowMonitor{node: node, streamBefore: before, service: service, cancel: cancel, ticks: make(chan flowTick, 200), done: make(chan struct{})}
	if cmd.Start() != nil {
		cancel()
		r.block("Controlled flow could not start")
		return nil
	}
	go func() {
		defer close(f.done)
		defer close(f.ticks)
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			var tick flowTick
			if json.Unmarshal(scanner.Bytes(), &tick) == nil {
				f.ticks <- tick
			}
		}
		_ = cmd.Wait()
	}()
	timer := time.NewTimer(8 * time.Second)
	defer timer.Stop()
	for {
		select {
		case tick, ok := <-f.ticks:
			if !ok {
				cancel()
				r.block("Controlled flow ended before a positive response")
				return nil
			}
			if tick.Receipt {
				f.first = tick
				var id uuid.UUID
				ip := l.overlayIPv4(service)
				if l.pool.QueryRow(l.ctx, `SELECT device_id FROM device_bindings WHERE overlay_ip=$1::inet AND valid_until>clock_timestamp() ORDER BY valid_from DESC LIMIT 1`, ip).Scan(&id) == nil {
					f.persistent = l.startPersistent(r, service, id)
				} else {
					r.block("F4-04: no live device binding for persistent UDP cohort " + service)
				}
				return f
			}
		case <-timer.C:
			cancel()
			<-f.done
			r.block("No positive controlled flow before fault; withdrawal timing not claimed")
			return nil
		}
	}
}
func (l *live) finishFlow(r *record, f *flowMonitor, event time.Time) {
	if f == nil {
		return
	}
	defer f.cancel()
	defer func() { l.finishPersistent(r, f.persistent, event, 60*time.Second) }()
	ticks := []flowTick{f.first}
	lastGood := f.first.At
	var firstBad time.Time
	reused := false
	for tick := range f.ticks {
		ticks = append(ticks, tick)
		reused = reused || tick.Reused
		if tick.Receipt {
			lastGood = tick.At
		} else if firstBad.IsZero() {
			firstBad = tick.At
		}
	}
	_, after, streamErr := l.readEnforcementTimingAt(f.node)
	r.jsonCheck("AT04 monitor producer stream after "+f.service, "whole stream parsed; original bounded monitor overlaps the rights change", map[string]any{"before": f.streamBefore, "after": after, "event_at": event, "first_sample": ticks[0].At, "last_sample": ticks[len(ticks)-1].At}, "streamed ExportContainerFile after bounded monitor; no diagnostic-buffer limit", streamErr == nil && ticks[0].At.Before(event) && ticks[len(ticks)-1].At.After(event))
	raw, _ := json.Marshal(ticks)
	r.check("persistent TCP withdrawal "+f.service, "positive/reused TCP stops within 60 seconds of rights-change input or real expiry", fmt.Sprintf("event_utc=%s; last_good=%s; first_failure=%s; reused_tcp=%t; stop_seconds=%.3f; ticks=%s", event.Format(time.RFC3339Nano), lastGood.Format(time.RFC3339Nano), firstBad.Format(time.RFC3339Nano), reused, firstBad.Sub(event).Seconds(), raw), "persistent CA-verified HTTPS GET every 500ms; actual rights change or real UTC expiry; two-second request deadline; maximum 85 seconds", reused && !firstBad.IsZero() && firstBad.After(event) && firstBad.Sub(event) <= 60*time.Second)
}
