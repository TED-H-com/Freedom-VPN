//go:build netlab

package netlab

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

func TestD104_AT13_DeadlineContinuity(t *testing.T) {
	skipLab(t)
	applyBaseline(t)
	m := deviceModel(time.Now().UTC(), 104, allBinds()[:1], generousQoS(), time.Hour)
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	out, stderr, err := dockerExecStdin([]string{"python3", "/work/tests/netlab/scripts/deadline.py"}, raw)
	if err != nil {
		t.Fatalf("deadline trace: %v %s %s", err, stderr, out)
	}
	save(t, "trace.json", out)
	var trace struct {
		Deadline int64 `json:"deadline_ns,string"`
		Apply    struct {
			Deadline enforce.DeadlineEvidence `json:"deadline"`
			Evidence enforce.ServingEvidence  `json:"evidence"`
		} `json:"apply"`
		Packets []struct {
			ID      int   `json:"packet_id"`
			Sent    int64 `json:"sent_ns,string"`
			Ingress int64 `json:"ingress_ns,string"`
			Egress  int64 `json:"egress_ns,string"`
			End     int64 `json:"end_ns,string"`
		} `json:"packets"`
	}
	if err := json.Unmarshal([]byte(out), &trace); err != nil {
		t.Fatal(err)
	}
	var last, firstDrop int64
	positive, drops, post := 0, 0, 0
	for i, p := range trace.Packets {
		if p.ID != i || p.Ingress < p.Sent || p.End < p.Ingress {
			t.Fatalf("missing or invalid ingress/read timestamp: %+v", p)
		}
		if p.Sent >= trace.Deadline {
			post++
		}
		if p.Egress > 0 {
			if firstDrop > 0 || p.Egress >= trace.Deadline || p.End < p.Egress {
				t.Fatalf("late forwarding: %+v", p)
			}
			positive++
			last = p.Egress
		} else {
			drops++
			if firstDrop == 0 {
				firstDrop = p.Sent
			}
		}
	}
	budget := enforce.DeadlineGranularity + enforce.DeadlineInstallBound
	if positive < 3 || drops < 3 || post < 3 || last < trace.Deadline-int64(budget) || firstDrop > trace.Deadline+int64(100*time.Millisecond) {
		t.Fatalf("continuity: last=%s first_drop=%s positives=%d drops=%d post=%d", time.Duration(last-trace.Deadline), time.Duration(firstDrop-trace.Deadline), positive, drops, post)
	}
	d := trace.Apply.Deadline
	if !d.GuardReleased || d.InstalledBy-d.RenderStart > d.InstallBound {
		t.Fatalf("unbounded install: %+v", d)
	}
	t.Logf("D104: granularity=%s bound=%s Apply->installed<=%s render->installed<=%s last_positive=%s first_drop=%s positives=%d drops=%d precise_read=%s", d.Granularity, d.InstallBound, d.InstalledBy-d.ApplyStart, d.InstalledBy-d.RenderStart, time.Duration(last-trace.Deadline), time.Duration(firstDrop-trace.Deadline), positive, drops, trace.Apply.Evidence.DeadlineRead.End-trace.Apply.Evidence.DeadlineRead.Start)
}
