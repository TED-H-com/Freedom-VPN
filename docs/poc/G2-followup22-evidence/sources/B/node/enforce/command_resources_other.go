//go:build !linux

package enforce

import "os"

func commandProcessState(_ *os.ProcessState, _ *CommandLifecycle) {}
func commandResources() CommandResources                          { return CommandResources{} }
