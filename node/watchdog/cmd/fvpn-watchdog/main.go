// Command fvpn-watchdog is the node watchdog (§16.3).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run", "once":
		os.Exit(run(os.Args[1], os.Args[2:]))
	case "version":
		fmt.Println("fvpn-watchdog dev")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: fvpn-watchdog run|once -config path")
}

func run(cmd string, args []string) int {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	config := fs.String("config", "", "path to agent.json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *config == "" {
		usage()
		return 2
	}
	cfg, err := watchdog.LoadConfig(*config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "watchdog: config")
		return 1
	}
	boot, err := watchdog.ReadBootID(cfg.BootIDFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "watchdog: boot id")
		return 1
	}
	w := &watchdog.Watch{Config: cfg, Enforce: watchdog.SystemEnforcer{}, BootID: boot}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if cmd == "once" {
		if err := w.Once(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "watchdog: check failed")
			return 1
		}
		return 0
	}
	if err := w.Run(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, "watchdog: stopped")
		return 1
	}
	return 0
}
