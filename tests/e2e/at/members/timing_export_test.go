//go:build e2e

package members

import (
	"context"
	"encoding/json"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"os"
	"path/filepath"
	"time"
)

func (l *live) readEnforcementTiming() ([]json.RawMessage, error) {
	rows, _, err := l.readEnforcementTimingAt(l.nodeContainer())
	return rows, err
}

type timingStreamReceipt struct {
	e2e.ContainerExport
	Node     string          `json:"node"`
	Records  int             `json:"records"`
	Last     json.RawMessage `json:"last_original_record"`
	Complete bool            `json:"complete_records"`
}

func (l *live) readEnforcementTimingAt(node string) ([]json.RawMessage, timingStreamReceipt, error) {
	receipt := timingStreamReceipt{Node: node}
	f, e := os.CreateTemp(filepath.Join(l.root, ".agent/tmp"), "members-timing-*")
	if e != nil {
		return nil, receipt, e
	}
	name := f.Name()
	_ = f.Close()
	defer os.Remove(name)
	ctx, cancel := context.WithTimeout(l.ctx, 10*time.Second)
	defer cancel()
	if receipt.ContainerExport, e = l.hook.ExportContainerFile(ctx, node, "/var/lib/vpn-node/e2e/enforcement-timing.jsonl", name); e != nil {
		return nil, receipt, e
	}
	f, e = os.Open(name)
	if e != nil {
		return nil, receipt, e
	}
	defer f.Close()
	rows, err := parseTimingStream(f)
	receipt.Records = len(rows)
	receipt.Complete = err == nil
	if len(rows) > 0 {
		receipt.Last = rows[len(rows)-1]
	}
	return rows, receipt, err
}
