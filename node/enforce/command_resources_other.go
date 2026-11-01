//go:build !linux

package enforce

import (
	"context"
	"os"
)

func commandProcessState(_ *os.ProcessState, _ *CommandLifecycle) {}
func commandResources(_ context.Context) CommandResources {
	return CommandResources{SkipReason: "unsupported_platform"}
}
