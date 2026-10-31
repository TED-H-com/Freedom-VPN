package e2e

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrefixE2EDoesNotMatchE2EExports(t *testing.T) {
	script := prefixScript(t)
	owned := func(prefix, name string) bool {
		t.Helper()
		cmd := exec.Command("sh", script, "owned", prefix, name)
		err := cmd.Run()
		if err == nil {
			return true
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return false
		}
		t.Fatalf("prefix.sh %s %s: %v", prefix, name, err)
		return false
	}
	if owned("e2e", "fvpn-e2e-exports-node") {
		t.Fatal("prefix e2e matched e2e-exports node")
	}
	if owned("e2e", "fvpn-e2e-exports") {
		t.Fatal("prefix e2e matched the e2e-exports network")
	}
	if owned("e2e", "fvpn-e2e-exports-pg") {
		t.Fatal("prefix e2e matched an e2e-exports volume")
	}
	if owned("e2e", "fvpn-e2e-exports-node:run") {
		t.Fatal("prefix e2e matched an e2e-exports image")
	}
	if owned("e2e-exports", "fvpn-e2e-node") {
		t.Fatal("prefix e2e-exports matched the e2e node")
	}
	if owned("e2e-exports", "fvpn-e2e") {
		t.Fatal("prefix e2e-exports matched the e2e network")
	}
	if owned("e2e-exports", "fvpn-e2e-member-exports-u01-p1") {
		t.Fatal("prefix e2e-exports matched an e2e member whose name contains exports")
	}
	for _, name := range []string{
		"fvpn-e2e-node",
		"fvpn-e2e-node:run",
		"fvpn-e2e",
		"fvpn-e2e-origin",
		"fvpn-e2e-pg",
		"fvpn-e2e-pgbackrest-repo",
		"fvpn-e2e-nb",
		"fvpn-e2e-member",
		"fvpn-e2e-member-exports-u01-p1",
		"fvpn-e2e-node2",
		"fvpn-e2e-node2-netns",
		"fvpn-e2e-node2-adguard",
		"fvpn-e2e-node-clone",
		"fvpn-e2e-node-clone-net",
		"fvpn-e2e-node-clone-copy",
		"fvpn-e2e-node-probe",
		"fvpn-e2e-state-reset",
		"fvpn-e2e-restore-pg",
		"fvpn-e2e-restore-verify",
		"fvpn-e2e-postgres",
		"fvpn-e2e-node-netns",
		"fvpn-e2e-installer-node",
		"fvpn-e2e-installer-node:24.04",
	} {
		if !owned("e2e", name) {
			t.Errorf("prefix e2e rejected %s", name)
		}
	}
	for _, name := range []string{
		"fvpn-e2e-exports-node",
		"fvpn-e2e-exports-postgres",
		"fvpn-e2e-exports-origin",
		"fvpn-e2e-exports",
		"fvpn-e2e-exports-pgbackrest-repo",
	} {
		if !owned("e2e-exports", name) {
			t.Errorf("prefix e2e-exports rejected %s", name)
		}
	}
	if owned("e2e", "fvpn-e2e-clone") || owned("e2e", "fvpn-e2e-clone-net") {
		t.Fatal("prefix e2e matched a network whose token is not node-clone")
	}
	if owned("e2e", "fvpn-devstack-node:local") || owned("e2e", "fvpn-pg") {
		t.Fatal("prefix e2e matched a shared image or the shared database")
	}
	if owned("e2e-", "fvpn-e2e-node") || owned("e2e.exports", "fvpn-e2e.exports-node") {
		t.Fatal("malformed prefix was accepted")
	}
}

func TestPrefixFilterKeepsTheImageTag(t *testing.T) {
	script := prefixScript(t)
	cmd := exec.Command("sh", script, "filter", "e2e")
	cmd.Stdin = strings.NewReader(strings.Join([]string{
		"fvpn-e2e-node:run",
		"fvpn-e2e-exports-node:run",
		"fvpn-e2e",
		"fvpn-devstack-node:local",
		"fvpn-pg",
		"fvpn-w4-secops-node:run",
	}, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	want := "fvpn-e2e-node:run\nfvpn-e2e\n"
	if got != want {
		t.Fatalf("filter output:\n%q\nwant:\n%q", got, want)
	}
}

func prefixScript(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "deploy", "devstack", "prefix.sh")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
