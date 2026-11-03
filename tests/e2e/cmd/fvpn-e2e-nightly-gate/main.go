package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "fvpn-e2e-nightly-gate: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("fvpn-e2e-nightly-gate", flag.ContinueOnError)
	summary := fs.String("summary", "", "Path to acceptance-summary.md (overrides -runs-root/-label)")
	runsRoot := fs.String("runs-root", "docs/e2e/runs", "Root directory for acceptance runs")
	label := fs.String("label", "nightly", "Run label subdirectory")
	listPath := fs.String("list", "", "Optional path to expected-blocked JSON (defaults to embedded tests/e2e/nightly_expected_blocked.json)")
	testExitCode := fs.Int("test-exit-code", 0, "Exit code of make e2e-test")
	testLogPath := fs.String("test-log", "", "Optional path to captured e2e-test log")
	stepSummary := fs.String("step-summary", os.Getenv("GITHUB_STEP_SUMMARY"), "Path to GitHub Actions step summary file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var list e2e.NightlyExpectedBlockedList
	var err error
	if *listPath != "" {
		list, err = e2e.LoadNightlyExpectedBlockedList(*listPath)
	} else {
		list, err = e2e.DefaultNightlyExpectedBlockedList()
	}
	if err != nil {
		return err
	}

	summaryPath := *summary
	if summaryPath == "" {
		summaryPath, err = e2e.ResolveNightlySummaryPath(*runsRoot, *label)
		if err != nil {
			return err
		}
	}

	var testLog string
	if *testLogPath != "" {
		rawLog, readErr := os.ReadFile(*testLogPath)
		if readErr != nil {
			return fmt.Errorf("read test log %q: %w", *testLogPath, readErr)
		}
		testLog = string(rawLog)
	}

	rep, evalErr := e2e.EvaluateNightlyGate(summaryPath, list, e2e.NightlyGateOptions{
		TestExitCode: *testExitCode,
		TestLog:      testLog,
	})
	runDir := filepath.Dir(summaryPath)
	if writeErr := e2e.WriteNightlyGateArtifacts(runDir, rep, *stepSummary); writeErr != nil {
		return writeErr
	}
	_, _ = os.Stdout.WriteString(rep.Markdown())
	return evalErr
}
