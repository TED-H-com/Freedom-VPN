package agent

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

func TestF29_HostSampleAccountingAndUnavailable(t *testing.T) {
	raw := []byte("cpu 10 20 30 40 50 60 70 80 9 8\nprocs_running 3\nprocs_blocked 12\n")
	s := parseHostSample(raw, HostSample{})
	if !s.OK || s.Total != 360 || s.Idle != 40 || s.IOWait != 50 || s.Running != 3 || s.Blocked != 12 {
		t.Fatalf("guest counted twice or idle includes iowait: %+v", s)
	}
	for _, bad := range []string{
		"", "cpu 1 2\nprocs_running 1\nprocs_blocked 2\n",
		"cpu 1 2 3 4 5 6 7 x\nprocs_running 1\nprocs_blocked 2\n",
		"cpu 18446744073709551615 1 0 0 0 0 0 0\nprocs_running 1\nprocs_blocked 2\n",
		string(raw) + "procs_running 2\n",
	} {
		if parseHostSample([]byte(bad), HostSample{}).OK {
			t.Fatal("unavailable sample presented as valid")
		}
	}
}

func TestF29_ApplyHostContextIsOptInAndDoesNotChangeTiming(t *testing.T) {
	r := newRig(t)
	if r.eng.measureHostBeforeApply() != nil {
		t.Fatal("host read enabled outside TEST ONLY trace")
	}
	r.eng.cfg.TestMode, r.eng.cfg.TestTimingPath = true, filepath.Join(r.dir, "host.jsonl")
	h := r.eng.measureHostBeforeApply()
	res, err := enforce.Apply(t.Context(), nil, enforce.Model{}, nil)
	if err == nil {
		t.Fatal("invalid model accepted")
	}
	h.finish(&enforce.CmdError{Cmd: "nft", Err: errors.New("private detail must not serialize"), Out: "private output"})
	r.eng.measureTiming(res.Evidence, "apply", h)
	rows := measurements(t, r.eng.cfg.TestTimingPath)
	if len(rows) != 1 || rows[0].Host == nil || rows[0].Host.Command != "nft" {
		t.Fatal("failed apply lost host context")
	}
	x := rows[0]
	if !x.Host.Before.OK || !x.Host.After.OK || x.Host.Before.End > x.Timing.Start || x.Host.After.Start < x.Timing.End || x.Timing.Succeeded {
		t.Fatalf("samples must bracket untouched enforcement timing: %+v", x)
	}
}
