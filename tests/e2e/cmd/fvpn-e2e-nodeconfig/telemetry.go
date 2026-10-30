package main

// TEST ONLY read-only container telemetry. No process arguments, file names,
// connection tuples or packet contents are emitted.
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type nodeSample struct {
	CPUUsec     uint64 `json:"cpu_usage_usec"`
	MemoryBytes uint64 `json:"memory_bytes"`
	OpenFDs     int    `json:"open_fds"`
	Processes   int    `json:"processes"`
	Conntrack   uint64 `json:"conntrack_entries"`
}

func readNodeSample(root string) (nodeSample, error) {
	var out nodeSample
	read := func(path string) (string, error) {
		b, err := os.ReadFile(filepath.Join(root, path))
		return strings.TrimSpace(string(b)), err
	}
	parse := func(path string) (uint64, error) {
		b, err := read(path)
		if err != nil {
			return 0, err
		}
		return strconv.ParseUint(b, 10, 64)
	}
	cpu, err := read("sys/fs/cgroup/cpu.stat")
	if err != nil {
		return out, fmt.Errorf("cgroup v2 CPU unavailable")
	}
	found := false
	for _, line := range strings.Split(cpu, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[0] == "usage_usec" {
			out.CPUUsec, err = strconv.ParseUint(f[1], 10, 64)
			found = err == nil
		}
	}
	if !found {
		return out, fmt.Errorf("cgroup v2 CPU missing")
	}
	out.MemoryBytes, err = parse("sys/fs/cgroup/memory.current")
	if err != nil {
		return out, fmt.Errorf("cgroup v2 memory unavailable")
	}
	out.Conntrack, err = parse("proc/sys/net/netfilter/nf_conntrack_count")
	if err != nil {
		return out, fmt.Errorf("namespace conntrack count unavailable")
	}
	processes, err := os.ReadDir(filepath.Join(root, "proc"))
	if err != nil {
		return out, fmt.Errorf("process inventory unavailable")
	}
	for _, process := range processes {
		if _, err := strconv.Atoi(process.Name()); err != nil || !process.IsDir() {
			continue
		}
		fds, err := os.ReadDir(filepath.Join(root, "proc", process.Name(), "fd"))
		if os.IsNotExist(err) {
			continue
		} // Process exited between the two reads.
		if err != nil {
			return out, fmt.Errorf("process descriptors unavailable")
		}
		out.Processes++
		out.OpenFDs += len(fds)
	}
	return out, nil
}

func telemetry() error {
	out, err := readNodeSample("/")
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
