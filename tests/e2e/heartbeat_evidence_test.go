package e2e

import (
	"fmt"
	"testing"
	"time"
)

func TestInstallerServingReadbackRejectsRetainedHeartbeat(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		delta   time.Duration
		serving int
		want    bool
	}{{-time.Minute, 1, false}, {time.Second, 0, false}, {time.Second, 1, true}} {
		raw := []byte(fmt.Sprintf("v1\nserving %d\nwall_unix_ns %d\n", tc.serving, now.Add(tc.delta).UnixNano()))
		if servingHeartbeatSince(raw, now) != tc.want {
			t.Fatalf("heartbeat evidence %+v", tc)
		}
	}
	if servingHeartbeatSince([]byte("serving 1\nwall_unix_ns malformed"), now) {
		t.Fatal("malformed heartbeat accepted")
	}
}
