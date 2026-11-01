//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

func (r *runner) native27(ctx context.Context, s *Stack, c *NativeCapture) {
	initial := Record{ID: "initial-selection", Title: "First selection after proven serving; corrected member-only observer", Spec: []string{"AT01", "F11-01"}, Status: StatusFail}
	r.nativeTransfer26(ctx, s, &initial, c)
	r.record(initial)
	stream := Record{ID: "member-stream", Title: "150-second serving member stream with every socket-write timestamp", Spec: []string{"AT02"}, Status: StatusFail}
	defer func() { r.record(stream) }()
	if !r.servingGate26(ctx, "stream-serving", c) {
		return
	}
	program := filepath.Join(r.lab.root, ".agent/tmp/followup27/stream")
	if exec.CommandContext(ctx, "go", "build", "-o", program, "./tests/e2e/cmd/fvpn-e2e-stream").Run() != nil {
		return
	}
	if exec.CommandContext(ctx, "docker", "cp", program, r.lab.member+":/var/lib/vpn-node/e2e/stream").Run() != nil {
		return
	}
	if c.Sample(ctx, "before-stream") != nil {
		return
	}
	type result struct {
		raw []byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		raw, e := exec.CommandContext(ctx, "docker", "exec", r.lab.member, "/var/lib/vpn-node/e2e/stream", "-addr", r.lab.originIP+":40002", "-seconds", "150").Output()
		done <- result{raw, e}
	}()
	var out result
running:
	for i := 0; ; i++ {
		if e := c.Sample(ctx, fmt.Sprintf("stream-%d", i)); e != nil {
			recPathNote(&stream)("sample_error", "unavailable")
			out = <-done
			break
		}
		select {
		case out = <-done:
			break running
		case <-ctx.Done():
			out = <-done
			break running
		case <-time.After(200 * time.Millisecond):
		}
	}
	if json.Valid(out.raw) {
		noteJSON(&stream, "sends", json.RawMessage(out.raw))
	}
	noteJSON(&stream, "sender_completed", out.err == nil)
	if c.Sample(ctx, "after-stream") != nil {
		return
	}
	// Observe late handshake/data without another request or reconnect.
	for i := 0; i < 12; i++ {
		if c.Sample(ctx, fmt.Sprintf("passive-tail-%d", i)) != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
	if out.err == nil && r.servingGate26(ctx, "post-stream-serving", c) {
		stream.Status = StatusPass
	}
}
