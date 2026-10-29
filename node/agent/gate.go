package agent

import (
	"os"
	"path/filepath"
)

// ForwardingBlocked reports the updater latch (§13.5). The file
// forwarding.blocked lives in the updater journal directory. The agent also
// treats `fvpn-updater recover` exit status 2 as a block. Either one keeps
// member forwarding off.
func ForwardingBlocked(journalDir string) bool {
	if journalDir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(journalDir, "forwarding.blocked"))
	return err == nil
}
