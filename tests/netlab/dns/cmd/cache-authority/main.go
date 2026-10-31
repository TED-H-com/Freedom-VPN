// TEST ONLY owned authority for the pinned-resolver cache conformance probe.
package main

import (
	"os"

	"github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/internal/cachefixture"
)

func main() {
	if cachefixture.Run() != nil {
		os.Exit(1)
	}
}
