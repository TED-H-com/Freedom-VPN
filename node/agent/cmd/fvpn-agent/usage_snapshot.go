package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
)

func usageCommand(args []string) int {
	return usageCommandTo(args, os.Stdout, os.Stderr)
}

func usageCommandTo(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || args[0] != "snapshot" {
		fmt.Fprintln(stderr, "usage: fvpn-agent usage snapshot -config path [-json]")
		return agent.ExitSnapshotArgs
	}
	return usageSnapshotTo(args[1:], stdout, stderr)
}

func usageSnapshotTo(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("usage snapshot", flag.ContinueOnError)
	config := fs.String("config", "", "path to agent.json")
	asJSON := fs.Bool("json", false, "print the stable JSON object")
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return agent.ExitSnapshotArgs
	}
	if fs.NArg() != 0 || *config == "" {
		fmt.Fprintln(stderr, "usage: fvpn-agent usage snapshot -config path [-json]")
		return agent.ExitSnapshotArgs
	}
	cfg, err := agent.LoadConfig(*config)
	if err != nil {
		if agent.ExitStatus(err) == agent.ExitUsage {
			fmt.Fprintln(stderr, "fvpn-agent: configuration error: host_reserve_bps is required and must be greater than zero")
			return agent.ExitUsage
		}
		fmt.Fprintln(stderr, "fvpn-agent: configuration error")
		return agent.ExitSnapshotArgs
	}
	code, out, msg := agent.UsageSnapshot(context.Background(), cfg, *asJSON)
	if code != 0 {
		fmt.Fprintln(stderr, msg)
		return code
	}
	if _, err := stdout.Write(out); err != nil {
		fmt.Fprintln(stderr, "fvpn-agent: usage snapshot failed")
		return 1
	}
	return 0
}
