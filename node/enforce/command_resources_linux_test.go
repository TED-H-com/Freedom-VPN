//go:build linux

package enforce

import (
	"os"
	"strings"
	"testing"
)

func TestD133_CgroupVisibilityAndMissingFields(t *testing.T) {
	const mounts = "20 19 0:1 / /sys/fs/cgroup rw - cgroup2 cgroup rw\n"
	for _, tc := range []struct {
		membership, mounts string
		ok                 bool
	}{
		{"0::/lab/leaf\n", mounts, true},
		{"0::/\n", mounts, true},
		{"0::/lab/../leaf\n", mounts, false},
		{"0::/outside\n", strings.Replace(mounts, " / /sys/", " /lab /sys/", 1), false},
		{"1:cpu:/lab\n", mounts, false},
		{"0::/lab\n", "", false},
	} {
		_, _, ok := cgroupLocation(tc.membership, tc.mounts)
		if ok != tc.ok {
			t.Fatalf("unexpected availability %t for %+v", ok, tc)
		}
	}
	files := map[string]string{
		"/proc/self/cgroup": "0::/lab/leaf\n", "/proc/self/mountinfo": mounts,
		"/sys/fs/cgroup/lab/leaf/cgroup.controllers": "cpu", "/sys/fs/cgroup/lab/leaf/cpu.max": "max 100000\n",
		"/sys/fs/cgroup/lab/cpu.max": "50000 100000\n", "/sys/fs/cgroup/lab/cpu.stat": "nr_throttled 17\n",
		"/sys/fs/cgroup/cpu.pressure": "some avg10=1.00 avg60=0.00 avg300=0.00 total=123\n",
	}
	read := func(p string) ([]byte, error) {
		s, ok := files[p]
		if !ok {
			return nil, os.ErrNotExist
		}
		return []byte(s), nil
	}
	r := readCommandResources(read)
	if !r.CgroupV2Available || !r.VisibleAncestorsComplete || r.HostAncestorsAvailable || len(r.Cgroups) != 3 || r.Cgroups[0].Max.Value != "max 100000" || r.Cgroups[1].Max.Value != "50000 100000" || r.Cgroups[0].Weight.Available || !r.Cgroups[2].Pressure.Available || r.Bracket.After.NS < r.Bracket.Before.NS {
		t.Fatalf("missing vs hidden vs unlimited: %+v", r)
	}
	delete(files, "/sys/fs/cgroup/lab/leaf/cgroup.controllers")
	r = readCommandResources(read)
	if r.CgroupV2Available || r.VisibleAncestorsComplete || len(r.Cgroups) != 0 {
		t.Fatal("invented cgroup on failed lookup")
	}
}
