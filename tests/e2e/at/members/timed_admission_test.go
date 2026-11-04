//go:build e2e

package members

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// D-160: admit the entire case before its first timestamp/deadline read. The
// case envelope covers overlapping persistent probes and all nested timers, so
// an inner tool wrapper cannot sleep while another measurement is still live.
// The optional foreground hook owns admission and the no-wait tool marker.
func (l *live) admitTimedCase(id string) func() {
	hook := os.Getenv("FVPN_AT_MEMBERS_SECTION_HOOK")
	if hook == "" {
		return func() {}
	}
	cmd := exec.CommandContext(l.ctx, "python3", "-B", hook, "start", id)
	if err := cmd.Run(); err != nil {
		l.t.Fatal("D-160 case admission unavailable before clocks: ", err)
	}
	end := l.timedSection("case-" + id)
	return func() {
		end()
		// Admission itself is not checked or awaited at the end of a measurement.
		cmd := exec.Command("python3", "-B", hook, "end", id)
		if err := cmd.Run(); err != nil {
			l.t.Error("D-160 case envelope close: ", err)
		}
	}
}

// Nested measurements publish their own boundaries as well as the enclosing
// case. Existing producer/command/sample timestamps remain the precise bounds.
func (l *live) timedSection(label string) func() {
	path := os.Getenv("FVPN_AT_MEMBERS_SECTIONS")
	if path == "" {
		return func() {}
	}
	_, file, line, _ := runtime.Caller(1)
	start := time.Now().UTC()
	write := func(event string) {
		raw, _ := json.Marshal(map[string]any{"event": event, "label": label, "started_at": start, "at": time.Now().UTC(), "file": filepath.Base(file), "line": line})
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			l.t.Error("D-160 section journal unavailable")
			return
		}
		_, err = f.Write(append(raw, '\n'))
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			l.t.Error("D-160 section journal write failed")
		}
	}
	write("start")
	return func() { write("end") }
}
