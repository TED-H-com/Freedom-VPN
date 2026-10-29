package agent

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
)

// Barrier is the usage meter the forwarding gate and the hand-off call
// (§10.7, §13.6 steps 8 and 11). Health is the meter's health. The reason
// meter_path_invalid is METER_PATH_INVALID. A full outbox, a disk floor and
// a persist failure stop forwarding under their own reasons.
type Barrier interface {
	Health(ctx context.Context) (usagestore.Health, error)
	Barrier(ctx context.Context) error
	FinalSample(ctx context.Context) error
	ResumeSampling()
}
