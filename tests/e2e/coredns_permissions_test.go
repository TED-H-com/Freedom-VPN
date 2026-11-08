package e2e

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAT20_CoreDNSFixtureModesPreserveSecretsAndUmask(t *testing.T) {
	fixtures := make(map[string][]byte)
	for _, name := range []string{"up.sh", "nginx.conf", "coredns/Corefile", "coredns/example.invalid.zone"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "deploy", "devstack", name))
		if err != nil {
			t.Fatal(err)
		}
		fixtures[name] = data
	}
	for _, mask := range []string{"022", "077"} {
		for _, stale := range []bool{false, true} {
			name := "fresh"
			if stale {
				name = "stale"
			}
			t.Run("umask"+mask+"/"+name, func(t *testing.T) {
				root := t.TempDir()
				state := filepath.Join(root, "state")
				write := func(path string, data []byte, mode os.FileMode) {
					t.Helper()
					if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, data, mode); err != nil {
						t.Fatal(err)
					}
					// Fix fixture modes independently of the test runner's umask.
					if err := os.Chmod(path, mode); err != nil {
						t.Fatal(err)
					}
				}
				checkMode := func(path string, want os.FileMode) {
					t.Helper()
					info, err := os.Stat(path)
					if err != nil {
						t.Fatal(err)
					}
					if got := info.Mode().Perm(); got != want {
						t.Errorf("%s mode = %04o, want %04o", path, got, want)
					}
				}
				checkBytes := func(path string, want string) {
					t.Helper()
					got, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					if string(got) != want {
						t.Errorf("%s contents changed unexpectedly", path)
					}
				}
				for name, data := range fixtures {
					write(filepath.Join(root, "deploy", "devstack", name), data, 0644)
				}
				write(filepath.Join(state, "ready"), []byte("ready\n"), 0600)
				composeEnv := "FVPN_E2E_PREFIX=modes\nFVPN_E2E_PORT_BASE=24000\nFVPN_E2E_LAB_PREFIX=172.28.31\nFVPN_E2E_ORIGIN_PREFIX=192.0.4\n"
				write(filepath.Join(state, "render", "compose.env"), []byte(composeEnv), 0600)
				secret := "TEST ONLY - NOT A SECRET\n"
				write(filepath.Join(state, "secrets", "private.txt"), []byte(secret), 0600)
				coredns := filepath.Join(state, "render", "coredns")
				if stale {
					write(filepath.Join(coredns, "Corefile"), []byte("stale Corefile\n"), 0600)
					write(filepath.Join(coredns, "example.invalid.zone"), []byte("stale zone\n"), 0600)
					// Unknown files in the mount must not be widened by recursive chmod.
					write(filepath.Join(coredns, "private.txt"), []byte(secret), 0600)
				}
				stubBin := filepath.Join(root, "bin")
				write(filepath.Join(stubBin, "docker"), []byte(`#!/bin/sh
set -eu
if [ "$#" -ne 6 ] || [ "$1" != build ] || [ "$2" != --iidfile ] ||
   [ "$3" != "$FVPN_E2E_STATE/render/postgres.image" ] || [ "$4" != -t ] ||
   [ "$5" != "fvpn-$FVPN_E2E_PREFIX-postgres:run" ] || [ "$6" != "$PWD/deploy/backup/postgres" ]; then
    echo 'unexpected Docker invocation' >&2
    exit 95
fi
umask > "$FVPN_E2E_STATE/observed-umask"
printf 'render reached\n' > "$FVPN_E2E_STATE/default-mode"
exit 73
`), 0755)
				write(filepath.Join(stubBin, "go"), []byte("#!/bin/sh\necho 'unexpected Go invocation' >&2\nexit 96\n"), 0755)
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, "sh", "-c", `umask "$1"; exec sh "$2"`, "test", mask, filepath.Join(root, "deploy", "devstack", "up.sh"))
				cmd.Env = []string{
					"PATH=" + stubBin + ":/usr/bin:/bin",
					"FVPN_E2E_STATE=" + state,
					"FVPN_E2E_PREFIX=modes",
					"FVPN_E2E_PORT_BASE=24000",
					"E2E_RESET=0",
				}
				output, err := cmd.CombinedOutput()
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 73 {
					t.Fatalf("up.sh did not stop at the first build: %v\n%s", err, output)
				}
				checkBytes(filepath.Join(state, "default-mode"), "render reached\n")
				checkMode(coredns, 0755)
				checkMode(filepath.Join(coredns, "Corefile"), 0644)
				checkMode(filepath.Join(coredns, "example.invalid.zone"), 0644)
				checkBytes(filepath.Join(coredns, "Corefile"), string(fixtures["coredns/Corefile"]))
				zone := strings.NewReplacer("172.28.30.", "172.28.31.", "192.0.3.", "192.0.4.").Replace(string(fixtures["coredns/example.invalid.zone"]))
				checkBytes(filepath.Join(coredns, "example.invalid.zone"), zone)
				for _, dir := range []string{state, filepath.Join(state, "render"), filepath.Join(state, "secrets")} {
					checkMode(dir, 0700)
				}
				checkMode(filepath.Join(state, "render", "compose.env"), 0600)
				checkBytes(filepath.Join(state, "render", "compose.env"), composeEnv)
				checkMode(filepath.Join(state, "secrets", "private.txt"), 0600)
				checkBytes(filepath.Join(state, "secrets", "private.txt"), secret)
				if stale {
					checkMode(filepath.Join(coredns, "private.txt"), 0600)
					checkBytes(filepath.Join(coredns, "private.txt"), secret)
				}
				observed, err := os.ReadFile(filepath.Join(state, "observed-umask"))
				if err != nil {
					t.Fatal(err)
				}
				gotMask, err := strconv.ParseUint(strings.TrimSpace(string(observed)), 8, 32)
				wantMask, _ := strconv.ParseUint(mask, 8, 32)
				if err != nil || gotMask != wantMask {
					t.Errorf("umask = %q (%v), want %s", observed, err, mask)
				}
				checkMode(filepath.Join(state, "default-mode"), 0666&^os.FileMode(wantMask))
			})
		}
	}
}
