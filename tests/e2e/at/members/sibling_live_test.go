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

type siblingCapture struct {
	sender, observer *exec.Cmd
	send, observe    *bufio.Reader
	finished         bool
}

func (l *live) startSibling(r *record) *siblingCapture {
	if !l.installProbe() {
		r.block("D-135 continuous sibling helper unavailable")
		return nil
	}
	c := &siblingCapture{}
	for _, role := range []string{"member", "node"} {
		if _, e := l.inside(role, "rm", "-f", "/tmp/fvpn-members-sibling-stop"); e != nil {
			return nil
		}
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), 6*time.Minute)
	l.t.Cleanup(cancel)
	start := func(role, mode string) (*exec.Cmd, *bufio.Reader, error) {
		cmd := exec.CommandContext(ctx, "docker", "exec", l.name(role), "/tmp/at-members-probe", mode, "bounded")
		out, e := cmd.StdoutPipe()
		if e != nil {
			return nil, nil, e
		}
		if e = cmd.Start(); e != nil {
			return nil, nil, e
		}
		reader := bufio.NewReader(out)
		line, e := reader.ReadString('\n')
		if e != nil || line != "ready\n" {
			_ = cmd.Wait()
			return nil, nil, fmt.Errorf("capture readiness unavailable")
		}
		return cmd, reader, nil
	}
	var err error
	c.observer, c.observe, err = start("node", "sibling-observe")
	if err != nil {
		r.block("D-135 node recorder unavailable")
		return nil
	}
	l.t.Cleanup(func() {
		if !c.finished {
			l.finishSibling(r, c)
		}
	})
	c.sender, c.send, err = start("member", "sibling-send")
	if err != nil {
		r.block("D-135 sibling sender unavailable")
		return c
	}
	// Capture an independent positive interval before revocation.
	time.Sleep(time.Second)
	return c
}

func (l *live) finishSibling(r *record, c *siblingCapture) {
	if c == nil || c.finished {
		return
	}
	c.finished = true
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	var sent, observed json.RawMessage
	stop := func(role string, cmd *exec.Cmd, reader *bufio.Reader, out *json.RawMessage) bool {
		if cmd == nil {
			return false
		}
		e := exec.CommandContext(ctx, "docker", "exec", l.name(role), "touch", "/tmp/fvpn-members-sibling-stop").Run()
		de := json.NewDecoder(reader).Decode(out)
		we := cmd.Wait()
		return e == nil && de == nil && we == nil
	}
	sendOK := stop("member", c.sender, c.send, &sent)
	obsOK := stop("node", c.observer, c.observe, &observed)
	l.siblingSent, l.siblingObserved = sent, observed
	r.jsonCheck("D-135 continuous admitted sibling capture", "every synthetic attempt and node ingress/egress retained; no capture loss", map[string]any{"sender": sent, "node": observed}, "20 ms UDP/40002 stream through native route; AF_PACKET metadata only, CLOCK_MONOTONIC; no forwarding repair", sendOK && obsOK)
}
