// Command fvpn-updater applies a verified desired state on one node
// (§13.4 to §13.9). It holds no release private key.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: fvpn-updater <run|recover|flush|status|version>")
		return 64
	}
	switch args[0] {
	case "version", "-h", "--help", "help":
		if args[0] == "version" {
			fmt.Fprintln(stdout, updater.Version)
			return 0
		}
		fmt.Fprintln(stderr, "usage: fvpn-updater <run|recover|flush|status|version>")
		return 0
	case "run", "recover", "flush":
		return command(args[0], args[1:], stderr)
	case "status":
		return status(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fvpn-updater: unknown command %q\n", args[0])
		return 64
	}
}

func status(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	journal := fs.String("journal", "", "existing journal database file")
	config := fs.String("config", "", "updater config JSON")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || (*journal != "" && *config != "") {
		return 64
	}
	path := *journal
	if *config != "" {
		cfg, err := updater.LoadConfig(*config)
		if err != nil {
			fmt.Fprintf(stderr, "fvpn-updater: %v\n", err)
			return 1
		}
		path = filepath.Join(cfg.Layout.Journal, updatejournal.FileName)
	}
	if path == "" {
		path = filepath.Join(updater.DefaultLayout().Journal, updatejournal.FileName)
	}
	snapshot, err := updatejournal.Snapshot(context.Background(), path)
	if err != nil {
		fmt.Fprintf(stderr, "fvpn-updater: status unavailable: %v\n", err)
		return 1
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	output := struct {
		updatejournal.SnapshotState
		WireCandidateHealth json.RawMessage `json:"candidate_health,omitempty"`
	}{SnapshotState: snapshot}
	if snapshot.State == "rolled_back" {
		output.WireCandidateHealth = snapshot.CandidateHealth
		if len(output.WireCandidateHealth) == 0 {
			output.WireCandidateHealth = json.RawMessage("null")
		}
	}
	if err := enc.Encode(output); err != nil {
		fmt.Fprintln(stderr, "fvpn-updater: write status failed")
		return 1
	}
	return 0
}

func command(name string, args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", "", "updater config JSON (paths only, no keys)")
	if err := fs.Parse(args); err != nil {
		return 64
	}
	if *cfgPath == "" {
		fmt.Fprintln(stderr, "fvpn-updater: -config is required")
		return 64
	}
	cfg, err := updater.LoadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "fvpn-updater: %v\n", err)
		return 1
	}
	if name == "flush" && cfg.ResultURL == "" {
		fmt.Fprintln(stderr, "fvpn-updater: flush requires result_url")
		return 1
	}
	eng, err := cfg.Engine()
	if err != nil {
		fmt.Fprintf(stderr, "fvpn-updater: %v\n", err)
		return 1
	}
	defer eng.Close()
	eng.Log = slog.New(slog.NewJSONHandler(stderr, nil))
	ctx := context.Background()
	var runErr error
	switch name {
	case "run":
		runErr = eng.Run(ctx)
	case "recover":
		runErr = eng.Recover(ctx)
	default:
		runErr = eng.Flush(ctx)
	}
	if runErr != nil && !errors.Is(runErr, updater.ErrMembersBlocked) {
		fmt.Fprintf(stderr, "fvpn-updater: %v\n", runErr)
	}
	return updater.ExitCode(runErr)
}
