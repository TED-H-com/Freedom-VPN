//go:build e2e

package updates_test

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Only ephemeral owned containers are instrumented. Generated updater config,
// original argv, service deadlines and product binaries remain unchanged.
func (s *updatesFollowup) installStopObserver(container string, systemd bool) {
	s.r.t.Helper()
	binary := filepath.Join(s.r.lab.root, ".agent/tmp/f13-stop-observer")
	if _, err := os.Stat(binary); err != nil {
		s.r.t.Fatal("stop observer binary missing")
	}
	if err := exec.CommandContext(s.ctx, "docker", "cp", binary, container+":/usr/local/bin/fvpn-stop-observer").Run(); err != nil {
		s.r.t.Fatal("copy stop observer failed")
	}
	var script string
	if systemd {
		script = `printf '%s\n' '#!/bin/sh' 'exec /usr/local/bin/fvpn-stop-observer /usr/bin/systemctl "$@"' > /usr/local/bin/systemctl
chmod 755 /usr/local/bin/systemctl`
	} else {
		script = `test ! -e /usr/local/bin/agent-service.original.sh
cp /usr/local/bin/agent-service.sh /usr/local/bin/agent-service.original.sh
printf '%s\n' '#!/bin/sh' 'if [ "${1:-}" = --supervisor ]; then exec /usr/local/bin/agent-service.original.sh "$@"; fi' 'exec /usr/local/bin/fvpn-stop-observer /usr/local/bin/agent-service.original.sh "$@"' > /usr/local/bin/agent-service.sh
chmod 755 /usr/local/bin/agent-service.sh`
	}
	if _, err := s.r.lab.docker(container, "sh", "-eu", "-c", script); err != nil {
		s.r.t.Fatal("install TEST ONLY stop observation failed")
	}
}
