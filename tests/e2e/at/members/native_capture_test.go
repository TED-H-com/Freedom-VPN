//go:build e2e

package members

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

// Only the owner export observes endpoints and packets. Raw pcaps remain private;
// Sample publishes route, link, ICE/relay and WireGuard endpoint readbacks.
type nativeCaptureSession struct {
	l                *live
	r                *record
	handle           *e2e.NativeCapture
	label, directory string
	result           json.RawMessage
}

func (l *live) startNativeCapture(r *record, member, label string) *nativeCaptureSession {
	s, err := l.hook.MemberTransportStack(l.name(member), l.nodeContainer())
	if err != nil {
		r.block("Native capture selected member/exit unavailable: " + label)
		return nil
	}
	handle, err := s.StartNativeCapture(l.ctx)
	if err != nil {
		r.block("Native capture start unavailable: " + label)
		return nil
	}
	return l.captureSession(r, handle, label)
}

func (l *live) captureSession(r *record, handle *e2e.NativeCapture, label string) *nativeCaptureSession {
	c := &nativeCaptureSession{l: l, r: r, handle: handle, label: label, directory: filepath.Join(l.root, ".agent/tmp/f15-native", l.runID, label+"-"+uuid.NewString())}
	l.t.Cleanup(func() { c.stop() })
	return c
}

func (c *nativeCaptureSession) sample(label string) {
	if c == nil || c.result != nil {
		return
	}
	if err := c.handle.Sample(c.l.ctx, label); err != nil {
		c.r.block("Native capture readback unavailable: " + c.label + "/" + label)
	}
}

func (c *nativeCaptureSession) stop() json.RawMessage {
	if c == nil {
		return nil
	}
	if c.result != nil {
		return c.result
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	evidence, err := c.handle.Stop(ctx, c.directory)
	c.result, _ = json.Marshal(evidence)
	// Sidecar has no path or payload. Exact pcaps stay in this ignored directory.
	if e := os.WriteFile(filepath.Join(c.directory, "capture.json"), c.result, 0600); e != nil {
		c.r.block("Native capture private receipt unavailable: " + c.label)
	}
	c.r.jsonCheck("native capture completeness "+c.label, "all processes stopped, complete exports, explicit statistics and zero capture drops", evidence, "StartNativeCapture; Sample; Stop with independent monotonic command bounds; private pcaps", err == nil)
	return c.result
}

func knownPMTURouteReadback(raw json.RawMessage) bool {
	var evidence e2e.NativeCaptureEvidence
	if json.Unmarshal(raw, &evidence) != nil {
		return false
	}
	seen := map[string]bool{}
	for _, sample := range evidence.Samples {
		n := sample.Points["node"]
		routes := n.Routes["peer_route"]
		if !n.Commands["peer_route"].OK || len(routes) != 1 {
			continue
		}
		mtu1280 := false
		for _, metric := range routes[0].Metrics {
			if metric["mtu"] == float64(1280) {
				mtu1280 = true
			}
		}
		switch sample.Label {
		case "before-bulk", "after-bulk":
			seen[sample.Label] = mtu1280
		case "before-route-constraint", "after-route-removal":
			seen[sample.Label] = !mtu1280
		}
	}
	return seen["before-route-constraint"] && seen["before-bulk"] && seen["after-bulk"] && seen["after-route-removal"]
}
