package enforce

import (
	"context"
	"time"
)

// ApplyCaptureForTest retains the final render input in memory for an explicit
// TEST ONLY fixture. It can contain selectors: never attach it to an ordinary
// observation/log. The fixture serializes/hashes it only after Apply returns.
// TimeoutReserve is private Model state needed to reproduce this exact render.
// The model's maps/slices belong to the caller; do not mutate them during Apply.
type ApplyCaptureForTest struct {
	Rendered       bool          `json:"rendered"`
	Model          Model         `json:"model"`
	TimeoutReserve time.Duration `json:"timeout_reserve_ns,string"`
}

type applyCaptureKey struct{}

func WithApplyCaptureForTest(ctx context.Context, capture *ApplyCaptureForTest) context.Context {
	return context.WithValue(ctx, applyCaptureKey{}, capture)
}

func captureRenderForTest(ctx context.Context, m Model) {
	if capture, _ := ctx.Value(applyCaptureKey{}).(*ApplyCaptureForTest); capture != nil {
		*capture = ApplyCaptureForTest{true, m, m.timeoutReserve}
	}
}
