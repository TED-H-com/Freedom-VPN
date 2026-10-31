//go:build netlab

package netlab

import (
	"strings"
	"testing"
)

func startReloadTrace(t *testing.T) {
	t.Helper()
	mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "start", "/scratch/at10")
	t.Cleanup(func() {
		// Also retain an aborted Apply, not only successful measurement runs.
		if !scratchExists("/scratch/at10-trace.done") {
			finishReloadTrace(t)
		}
	})
	_, _, _ = dockerExec("rm", "-f", "/scratch/at10-trace.done")
}
func finishReloadTrace(t *testing.T) {
	t.Helper()
	if scratchExists("/scratch/stream-a1.log.pid") {
		mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "finish-stream", "/scratch/stream-a1.log")
	}
	mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "stop", "/scratch/at10")
	mustExec(t, "touch", "/scratch/at10-trace.done")
	for _, name := range []string{"at10-origin.txt", "at10-origin.err", "at10-node.txt", "at10-node.err", "at10-responder.txt", "at10-responder.err", "at10-apply.jsonl", "stream-a1.log", "stream-a1.log.jsonl"} {
		if scratchExists("/scratch/" + name) {
			evidenceName := name
			if strings.HasSuffix(name, ".log") {
				evidenceName = strings.TrimSuffix(name, ".log") + ".txt"
			}
			save(t, evidenceName, readScratch(t, "/scratch/"+name))
		}
	}
}
