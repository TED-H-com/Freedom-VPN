package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAT31_TelemetryUnavailableIsNotReportedAsZero(t *testing.T) {
	root := t.TempDir()
	for name, value := range map[string]string{
		"sys/fs/cgroup/cpu.stat":                    "usage_usec 123456\nuser_usec 100000\n",
		"sys/fs/cgroup/memory.current":              "1048576\n",
		"proc/sys/net/netfilter/nf_conntrack_count": "7\n",
		"proc/1/fd/0":                               "", "proc/1/fd/1": "", "proc/2/fd/0": "",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := readNodeSample(root)
	if err != nil || got.CPUUsec != 123456 || got.MemoryBytes != 1048576 || got.Conntrack != 7 || got.OpenFDs != 3 || got.Processes != 2 {
		t.Fatalf("sample=%+v err=%v", got, err)
	}
	if err := os.Remove(filepath.Join(root, "proc/sys/net/netfilter/nf_conntrack_count")); err != nil {
		t.Fatal(err)
	}
	if _, err := readNodeSample(root); err == nil {
		t.Fatal("missing conntrack evidence became zero")
	}
}
