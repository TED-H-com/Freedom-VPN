package main

import (
	"os/signal"
	"syscall"
)

// configureTrustOutputSignals is called only at the real process entry point.
// Go otherwise exits on a broken descriptor-1 pipe before writeTrustResult can
// report its fixed diagnostic. Signal policy is process-wide, so keep it out of
// run/runTrust and library code: in-process calls must not change later legacy
// commands. The five original subcommands retain Go's default SIGPIPE behavior.
func configureTrustOutputSignals(args []string) {
	if len(args) != 0 && (args[0] == "trust-update" || args[0] == "trust-verify") {
		signal.Ignore(syscall.SIGPIPE)
	}
}
