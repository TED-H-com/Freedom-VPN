//go:build e2e

package members

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type flowTick struct {
	At      time.Time `json:"at"`
	Receipt bool      `json:"origin_receipt"`
	Reused  bool      `json:"reused_tcp"`
}
type flowMonitor struct {
	cancel context.CancelFunc
	ticks  chan flowTick
	done   chan struct{}
	first  flowTick
}

func (l *live) startFlow(r *record, service string) *flowMonitor {
	ctx, cancel := context.WithTimeout(l.ctx, 90*time.Second)
	cmd := exec.CommandContext(ctx, "docker", "exec", l.name(service), "/tmp/at-members-probe", "monitor", "bounded")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		r.block("Controlled flow observer unavailable")
		return nil
	}
	f := &flowMonitor{cancel: cancel, ticks: make(chan flowTick, 200), done: make(chan struct{})}
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
	raw, _ := json.Marshal(ticks)
	r.check("persistent TCP refund withdrawal", "positive/reused TCP stops within 60 seconds of full-refund input", fmt.Sprintf("event_utc=%s; last_good=%s; first_failure=%s; reused_tcp=%t; stop_seconds=%.3f; ticks=%s", event.Format(time.RFC3339Nano), lastGood.Format(time.RFC3339Nano), firstBad.Format(time.RFC3339Nano), reused, firstBad.Sub(event).Seconds(), raw), "persistent CA-verified HTTPS GET every 500ms; actual full refund hook; two-second request deadline; maximum 85 seconds", reused && !firstBad.IsZero() && firstBad.After(event) && firstBad.Sub(event) <= 60*time.Second)
}
