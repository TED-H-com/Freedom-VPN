package main

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
)

func TestAPI01_StartupPoolBoundBeforeDatabase(t *testing.T) {
	originalFlags, originalArgs := flag.CommandLine, os.Args
	t.Cleanup(func() { flag.CommandLine = originalFlags; os.Args = originalArgs })
	flag.CommandLine = flag.NewFlagSet("TEST ONLY worker", flag.ContinueOnError)
	os.Args = []string{"fvpn-worker"}
	t.Setenv("FVPN_WORKER_CONCURRENCY", "12")
	// No secret exists: the budget must reject startup before a database read.
	t.Setenv("FVPN_DATABASE_URL_FILE", filepath.Join(t.TempDir(), "absent-database-secret"))
	err := run(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var bound *worker.PoolBudgetError
	if !errors.As(err, &bound) || bound.Maximum != 11 || !strings.Contains(err.Error(), "largest accepted worker_concurrency is 11") {
		t.Fatalf("startup: %v", err)
	}
	t.Log(err)
}
