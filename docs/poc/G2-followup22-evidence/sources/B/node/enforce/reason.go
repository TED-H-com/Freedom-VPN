package enforce

import "fmt"

// Reason is a stable code the agent can report. The strings are the contract.
type Reason string

const (
	// ReasonMeterPathInvalid means the accounting path is not the one this
	// generation claimed: hook, NAT position, source identity, or a counter
	// that can be skipped (§10.3).
	ReasonMeterPathInvalid Reason = "METER_PATH_INVALID"
	// ReasonIsolationBaselineMissing means member forwarding is not held at
	// the fail-closed boot ruleset (§8.2, §17.1).
	ReasonIsolationBaselineMissing Reason = "ISOLATION_BASELINE_MISSING"
	// ReasonGenerationMismatch means the kernel ruleset is not the requested
	// generation.
	ReasonGenerationMismatch Reason = "GENERATION_MISMATCH"
	// ReasonHookPriority means a base chain hook or priority drifted (§8.2).
	ReasonHookPriority Reason = "HOOK_PRIORITY_MISMATCH"
	// ReasonCounterOrder means an accept can skip the per-device counters,
	// including an established/related accept placed before the lease check
	// (§10.2, §16.3).
	ReasonCounterOrder Reason = "COUNTER_ORDER_INVALID"
	// ReasonFlowtable means a flowtable or offload could bypass the counters
	// (§10.3).
	ReasonFlowtable Reason = "FLOWTABLE_BYPASS"
	// ReasonTCForeignFilter means foreign or unreadable tc state can bypass
	// admission, accounting or shaping on a member-path interface (D-085).
	ReasonTCForeignFilter Reason = "TC_FOREIGN_FILTER"
	// ReasonDeviceClass means a device binding has no counter or class.
	ReasonDeviceClass Reason = "DEVICE_CLASS_MISSING"
	// ReasonMeterEpoch means a named counter went backwards or disappeared.
	// Callers must open a new meter epoch and must not hide the gap (§10.7).
	ReasonMeterEpoch Reason = "METER_EPOCH_CHANGE"
	// ReasonQoS means the HTB parent or a member class is missing or wrong
	// (§11.4, §11.5).
	ReasonQoS Reason = "QOS_CLASS_MISSING"
	// ReasonMark means a rule writes skb mark or conntrack mark. NetBird owns
	// 0x1BD00-0x1BDFF (§11.4).
	ReasonMark Reason = "MARK_SEMANTICS_VIOLATED"
	// ReasonIsolation means the allow/deny policy itself does not match.
	ReasonIsolation Reason = "ISOLATION_POLICY_INVALID"
	// ReasonProbeUnbounded means a synthetic-probe set element has no kernel
	// timeout, so the exception would outlive the health check (§13.9).
	ReasonProbeUnbounded Reason = "PROBE_UNBOUNDED"
)

// CheckError is a verification failure with a specific reason.
type CheckError struct {
	Reason Reason
	Detail string
	tc     *TCReadEvidence // deciding read, retained before cleanup
}

func (e *CheckError) Error() string {
	if e == nil {
		return ""
	}
	if e.Detail == "" {
		return string(e.Reason)
	}
	return string(e.Reason) + ": " + e.Detail
}

func checkf(reason Reason, format string, args ...any) error {
	return &CheckError{Reason: reason, Detail: fmt.Sprintf(format, args...)}
}
