package watchdog

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestD156_PendingAdoptionKeepsOriginalBounds(t *testing.T) {
	for _, kind := range []string{"pending", "stale", "expired", "future", "other_boot", "unknown_boot", "zero_generation", "zero_deadline", "serving_claim", "baseline_claim", "unverified", "missing_rules"} {
		t.Run(kind, func(t *testing.T) {
			w, x, b, _ := watchRig(t)
			now, _ := w.mono()
			b.Serving, b.BaselineOK, b.Reason = false, false, "boot_adopting"
			switch kind {
			case "stale":
				b.Mono = now - 30*time.Second - time.Nanosecond
			case "expired":
				b.PolicyExpires = now
			case "future":
				b.Mono = now + time.Nanosecond
			case "other_boot":
				b.BootID = uuid.New()
			case "unknown_boot":
				w.BootID = uuid.Nil
			case "zero_generation":
				b.Generation = 0
			case "zero_deadline":
				b.PolicyExpires = 0
			case "serving_claim":
				b.Serving = true
			case "baseline_claim":
				b.BaselineOK = true
			case "unverified":
				b.Reason = "boot_unverified"
			case "missing_rules":
				x.present = 0
			}
			if err := WriteHeartbeat(w.Config.HeartbeatPath, b); err != nil {
				t.Fatal(err)
			}
			if err := w.Once(t.Context()); err != nil {
				t.Fatal(err)
			}
			want := 1
			if kind == "pending" {
				want = 0
			}
			if x.calls != want {
				t.Fatalf("%s: baselines=%d want=%d", kind, x.calls, want)
			}
		})
	}
}
