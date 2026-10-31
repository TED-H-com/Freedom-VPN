//go:build e2e

package members

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func (l *live) readEnforcementTiming() ([]json.RawMessage, error) {
	f, e := os.CreateTemp(filepath.Join(l.root, ".agent/tmp"), "members-timing-*")
	if e != nil {
		return nil, e
	}
	name := f.Name()
	_ = f.Close()
	defer os.Remove(name)
	ctx, cancel := context.WithTimeout(l.ctx, 10*time.Second)
	defer cancel()
	if _, e = l.hook.ExportContainerFile(ctx, l.nodeContainer(), "/var/lib/vpn-node/e2e/enforcement-timing.jsonl", name); e != nil {
		return nil, e
	}
	f, e = os.Open(name)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return parseTimingStream(f)
}
