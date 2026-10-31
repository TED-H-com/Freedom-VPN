package enforce

import (
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// CommandTiming brackets an Executor call, not a kernel commit. OK describes
// command completion only. Zero timestamps mean absent/unavailable samples.
type CommandTiming struct {
	Start time.Duration `json:"start_ns,string"`
	End   time.Duration `json:"end_ns,string"`
	OK    bool          `json:"ok"`
}

// VerificationTiming retains each verifier invocation, including both reads
// during quarantine recovery. TCRead brackets readServingTC (multiple commands);
// NFTRead and PrecisionRead each bracket one Executor.Output call. SHA256 is
// the corresponding D-106 projection digest, not a digest of these timestamps.
type VerificationTiming struct {
	Start         time.Duration        `json:"start_ns,string"`
	End           time.Duration        `json:"end_ns,string"`
	NFTRead       CommandTiming        `json:"nft_read"`
	TCRead        CommandTiming        `json:"tc_read"`
	PrecisionRead DeadlineReadEvidence `json:"precision_read"`
	SHA256        string               `json:"tc_sha256"`
	Verified      bool                 `json:"verified"`
}

// TimingEvidence is a local diagnostic sibling of D-106, version 1. All *_ns
// values are decimal-string Linux CLOCK_MONOTONIC nanoseconds, comparable only
// within the same boot/time namespace. It contains no selectors or command text.
// Operation is apply or verify. Deadline is populated only for apply; absent
// stages stay zero. Succeeded is the returned error == nil; Verified is the
// serving result, which can remain true after a conntrack-delete failure.
type TimingEvidence struct {
	Version             int                  `json:"version"`
	Clock               string               `json:"clock"`
	Operation           string               `json:"operation"`
	CandidateGeneration decstr.U64           `json:"candidate_generation"`
	Start               time.Duration        `json:"start_ns,string"`
	End                 time.Duration        `json:"end_ns,string"`
	Succeeded           bool                 `json:"succeeded"`
	Verified            bool                 `json:"verified"`
	Deadline            DeadlineEvidence     `json:"deadline"`
	Verifications       []VerificationTiming `json:"verifications"`
}

// TimingEvidence returns a detached snapshot without I/O or resampling. Nil
// means no retained operation. Apply returns it even on rollback/quarantine;
// later live VerifyServing calls produce separate snapshots. This is not an
// ObservationReport field and does not change the D-106 wire schema or digest.
func (ev ServingEvidence) TimingEvidence() *TimingEvidence {
	if ev.timing == nil {
		return nil
	}
	v := *ev.timing
	v.Verifications = append([]VerificationTiming{}, v.Verifications...)
	return &v
}

func newTiming(operation string, generation uint64) *TimingEvidence {
	return &TimingEvidence{Version: 1, Clock: "CLOCK_MONOTONIC", Operation: operation,
		CandidateGeneration: decstr.U64(generation), Verifications: []VerificationTiming{}}
}

func timingSample() time.Duration {
	now, _ := MonotonicNow()
	return now // zero on clock failure; consumers must reject incomplete samples
}
