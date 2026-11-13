//go:build integration

package backup

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestAT30_BackupLocaleGuards(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	names, err := drillNamesFromEnv(os.Getenv("FVPN_BACKUP_DRILL_PREFIX"), cryptorand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanCancel()
		if err := removeDrillResources(cleanCtx, names.Prefix); err != nil {
			t.Error(err)
		}
	})
	// Do not skip missing Docker/locales: pitr-drill is the required CI home
	// for en_US.UTF-8, independent of the ubuntu-latest host's locale catalog.
	runCmd(ctx, t, root, nil, "docker", "build", "-t", names.Image, filepath.Join(root, "deploy/backup/postgres"))
	out := runCmd(ctx, t, root, nil, "docker", "run", "--rm", "--name", names.Prefix+"-locale-info", "--network", "none",
		"--entrypoint", "bash", names.Image, "-c", `
locale
locale -a
bash --version
awk -W version
`)
	t.Logf("built image locale/tool evidence:\n%s", out)
	for i, tc := range []struct{ locale, mode string }{
		{"C", "all"}, {"C.UTF-8", "all"}, {"en_US.UTF-8", "all"},
		{"en_US.UTF-8", "lang"}, {"en_US.UTF-8", "collate"}, {"en_US.UTF-8", "ctype"},
	} {
		t.Run(tc.locale+"/"+tc.mode, func(t *testing.T) {
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--name", fmt.Sprintf("%s-locale-%d", names.Prefix, i), "--network", "none",
				"--mount", "type=bind,src="+filepath.Join(root, "deploy/backup")+",dst=/scripts,readonly",
				"--mount", "type=bind,src="+filepath.Join(root, "tests/backup/testdata")+",dst=/fixtures,readonly",
				"--entrypoint", "bash", names.Image, "/fixtures/locale-guards.sh", "/scripts", "/guard-work", tc.locale, tc.mode)
			output, err := cmd.CombinedOutput()
			if err != nil || localeWarning(output) {
				t.Fatalf("required locale guard matrix: %v\n%s", err, output)
			}
			t.Logf("%s", output)
		})
	}
}
