package enforce

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

const deadlineGuard = "fv_deadline_guard"

// MonotonicNow returns Linux CLOCK_MONOTONIC elapsed time, shared with kernel
// element expiry and the agent's timeanchor. Wall-clock corrections do not
// affect it. Suspend/resume is a separate platform acceptance prerequisite.
func MonotonicNow() (time.Duration, error) {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return 0, err
	}
	return time.Duration(ts.Nano()), nil
}

// DeadlineEvidence brackets installation, not a hardware timestamp. InstalledBy
// is sampled after the atomic nft batch returns; the element was installed no
// later than this sample. All times use CLOCK_MONOTONIC, in nanoseconds.
type DeadlineEvidence struct {
	Granularity   time.Duration `json:"granularity_ns,string"`
	InstallBound  time.Duration `json:"install_bound_ns,string"`
	ApplyStart    time.Duration `json:"apply_start_ns,string"`
	RenderStart   time.Duration `json:"render_start_ns,string"`
	InstallStart  time.Duration `json:"install_start_ns,string"`
	InstalledBy   time.Duration `json:"installed_by_ns,string"`
	InstallOK     bool          `json:"install_ok"`
	ReleaseStart  time.Duration `json:"release_start_ns,string"`
	ReleaseEnd    time.Duration `json:"release_end_ns,string"`
	GuardReleased bool          `json:"guard_released"`
}

// guardedNFT atomically installs the candidate behind a separate member-path
// drop chain. Even an arbitrarily delayed commit or death before checking the
// elapsed budget cannot expose an overlong relative timeout. Once the budget
// passes, deleting the guard cannot restart the already-counting-down elements.
func guardedNFT(plan string, m Model) string {
	return plan + fmt.Sprintf("add chain inet fvpn %s { type filter hook forward priority %d; policy accept; }\nflush chain inet fvpn %s\nadd rule inet fvpn %s iifname %q drop\nadd rule inet fvpn %s oifname %q drop\n",
		deadlineGuard, HookPriorityFilter-1, deadlineGuard, deadlineGuard, m.Datapath.MemberIngress, deadlineGuard, m.Datapath.MemberIngress)
}

func releaseDeadlineGuard(ctx context.Context, ex Executor) error {
	return ex.NftBatch(ctx, "delete chain inet fvpn "+deadlineGuard+"\n")
}
