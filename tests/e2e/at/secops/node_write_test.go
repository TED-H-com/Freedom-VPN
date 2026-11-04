//go:build e2e && integration

package secops

import (
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
)

// Keep the real response (including an already-intended 412). Acceptance never
// substitutes for the caller's operation and dataplane read-back.
func (p *liveProbe) writeNode(name string, admin *liveSession, method, writePath, readPath string, body any, etag string, intended func([]byte) bool) labstack.NodeWriteResult {
	stack, err := labstack.Open()
	if err != nil {
		p.t.Fatal("node write stack open", err)
	}
	defer stack.Close()
	r := stack.WriteNode(p.t.Context(), labstack.BrowserSession{Client: admin.client, CSRF: admin.csrf}, method, writePath, readPath, body, etag, intended, 202)
	p.observations = append(p.observations, map[string]any{"name": name, "actual_status": r.Reply.Status, "actual_code": r.Reply.Code, "passed": r.Accepted(202), "already_intended": r.AlreadyIntended, "node_write_receipt": r.Receipt, "utc": time.Now().UTC()})
	if !r.Accepted(202) {
		p.t.Errorf("%s: node write status=%d code=%s outcome=%s", name, r.Reply.Status, r.Reply.Code, r.Receipt.Outcome)
	}
	return r
}
