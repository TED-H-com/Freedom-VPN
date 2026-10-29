package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
)

func TestVersionAndUsage(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"version"}, &out, &errb); code != 0 || strings.TrimSpace(out.String()) != updater.Version {
		t.Fatalf("version code %d out %q err %q", code, out.String(), errb.String())
	}
	if code := run(nil, &out, &errb); code != 64 {
		t.Fatalf("usage %d", code)
	}
	if code := run([]string{"nope"}, &out, &errb); code != 64 {
		t.Fatalf("unknown %d", code)
	}
	if code := run([]string{"run"}, &out, &errb); code != 64 {
		t.Fatalf("missing config %d", code)
	}
	if code := run([]string{"run", "-not-a-flag"}, &out, &errb); code != 64 {
		t.Fatalf("flag %d", code)
	}
	if code := run([]string{"help"}, &out, &errb); code != 0 {
		t.Fatalf("help %d", code)
	}
}
