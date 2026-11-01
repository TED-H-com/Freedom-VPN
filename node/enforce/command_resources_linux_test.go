//go:build linux

package enforce

import (
	"context"
	"fmt"
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
	r := readCommandResources(t.Context(), read)
	if !r.CgroupV2Available || !r.VisibleAncestorsComplete || r.HostAncestorsAvailable || len(r.Cgroups) != 3 || r.Cgroups[0].Max.Value != "max 100000" || r.Cgroups[1].Max.Value != "50000 100000" || r.Cgroups[0].Weight.Available || !r.Cgroups[2].Pressure.Available || r.Bracket.After.NS < r.Bracket.Before.NS {
		t.Fatalf("missing vs hidden vs unlimited: %+v", r)
	}
	delete(files, "/sys/fs/cgroup/lab/leaf/cgroup.controllers")
	r = readCommandResources(t.Context(), read)
	if r.CgroupV2Available || r.VisibleAncestorsComplete || len(r.Cgroups) != 0 {
		t.Fatal("invented cgroup on failed lookup")
	}
}

func TestReviewR18_ResourceReadsStopOnCancellation(t *testing.T) {
	for _, cancelAt := range []int{0, 1, 2, 3, 5} {
		t.Run(fmt.Sprint(cancelAt), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if cancelAt == 0 {
				cancel()
			}
			reads := 0
			r := readCommandResources(ctx, func(path string) ([]byte, error) {
				reads++
				if reads == cancelAt {
					cancel()
				}
				switch path {
				case "/proc/self/cgroup":
					return []byte("0::/lab/leaf\n"), nil
				case "/proc/self/mountinfo":
					return []byte("20 19 0:1 / /sys/fs/cgroup rw - cgroup2 cgroup rw\n"), nil
				default:
					return []byte("100"), nil
				}
			})
			if reads != cancelAt || r.SkipReason != "context_canceled" || r.Bracket.Before.Available || r.Bracket.After.Available || r.CgroupV2Available || len(r.Cgroups) != 0 {
				t.Fatalf("canceled sample retained or reads continued: reads=%d sample=%+v", reads, r)
			}
		})
	}
}
