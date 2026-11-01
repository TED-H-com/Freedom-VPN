// Command fvpn-e2e-native-observe reads TEST ONLY path metadata in a container.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture"
	"os"
	"time"
)

func main() {
	clock := flag.Bool("clock", false, "clock pair only")
	peer := flag.String("peer", "", "selected peer overlay address")
	origin := flag.String("origin", "", "controlled origin address")
	flag.Parse()
	var out any
	var err error
	if *clock {
		out, err = nativecapture.Now()
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		out, err = nativecapture.Read(ctx, *peer, *origin)
	}
	if err != nil {
		os.Exit(1)
	}
	if json.NewEncoder(os.Stdout).Encode(out) != nil {
		os.Exit(1)
	}
}
