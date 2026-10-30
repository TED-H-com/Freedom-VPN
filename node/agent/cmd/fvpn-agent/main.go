// Command fvpn-agent is the exit-node agent (§5, §16.3).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "enroll":
		os.Exit(enroll(os.Args[2:]))
	case "run":
		os.Exit(run(os.Args[2:], true))
	case "step":
		os.Exit(run(os.Args[2:], false))
	case "version":
		fmt.Println("fvpn-agent dev")
	case "usage":
		os.Exit(usageCommand(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: fvpn-agent enroll -config path -token-file path|-")
	fmt.Fprintln(os.Stderr, "       fvpn-agent run|step -config path")
	fmt.Fprintln(os.Stderr, "       fvpn-agent usage snapshot -config path [-json]")
	fmt.Fprintln(os.Stderr, "       fvpn-agent version")
}

func enroll(args []string) int {
	fs := flag.NewFlagSet("enroll", flag.ContinueOnError)
	config := fs.String("config", "", "path to agent.json")
	tokenFile := fs.String("token-file", "", "enrollment token file, or - for stdin")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *config == "" || *tokenFile == "" {
		usage()
		return 2
	}
	// The token is read from the file or stdin only. The environment is ignored (§6.1).
	token, err := agent.ReadToken(*tokenFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-agent: token file")
		return 1
	}
	eng, err := open(*config)
	if err != nil {
		if agent.ExitStatus(err) == agent.ExitUsage {
			fmt.Fprintln(os.Stderr, "fvpn-agent: configuration error: host_reserve_bps is required and must be greater than zero")
			return agent.ExitUsage
		}
		fmt.Fprintln(os.Stderr, "fvpn-agent:", err.Error())
		return 1
	}
	defer eng.Close()
	if err := eng.Enroll(context.Background(), token); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-agent: enrollment failed")
		return 1
	}
	if err := agent.NeutralizeToken(*tokenFile); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-agent: token file")
		return 1
	}
	return 0
}

func run(args []string, loop bool) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	config := fs.String("config", "", "path to agent.json")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *config == "" {
		usage()
		return 2
	}
	eng, err := open(*config)
	if err != nil {
		if agent.ExitStatus(err) == agent.ExitUsage {
			fmt.Fprintln(os.Stderr, "fvpn-agent: configuration error: host_reserve_bps is required and must be greater than zero")
			return agent.ExitUsage
		}
		fmt.Fprintln(os.Stderr, "fvpn-agent:", err.Error())
		return 1
	}
	defer eng.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if loop {
		if err := eng.Run(ctx); err != nil && ctx.Err() == nil {
			fmt.Fprintln(os.Stderr, "fvpn-agent:", err.Error())
			return 1
		}
		return 0
	}
	if err := eng.Boot(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-agent:", err.Error())
		return 1
	}
	if err := eng.Step(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-agent:", err.Error())
		return 1
	}
	return 0
}

func open(path string) (*agent.Engine, error) {
	cfg, err := agent.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	log := slog.New(agent.RedactHandler{Next: slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})})
	return agent.Start(agent.Options{Config: cfg, Log: log})
}
