//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// NativeLogScan inventories the protected transient stream, never its values.
// JoinNativeMember invokes this exported scanner before deleting that stream.
type NativeLogScan struct {
	Present        bool   `json:"present"`
	Bytes          int64  `json:"bytes"`
	Mode           string `json:"mode"`
	DeviceFlowKeys int    `json:"device_flow_keys"`
}

func (s *Stack) ScanNativeJoinLog(ctx context.Context, container string) (NativeLogScan, error) {
	var out NativeLogScan
	if !strings.HasPrefix(container, s.lab.prefix+"-member") {
		return out, fmt.Errorf("member container outside stack")
	}
	// The command emits aggregate counts only. No matched line, URL, code or
	// secret crosses the container boundary for this scan.
	raw, err := exec.CommandContext(ctx, "docker", "exec", container, "sh", "-c", `p=/var/lib/vpn-node/e2e/netbird-up.log
 if [ ! -f "$p" ]; then printf '{"present":false}'; exit 0; fi
 size=$(stat -c %s "$p"); mode=$(stat -c %u:%a "$p")
 count=$(grep -Eoc 'user_code=|verification_uri|device_code=' "$p" || true)
 printf '{"present":true,"bytes":%s,"mode":"%s","device_flow_keys":%s}' "$size" "$mode" "$count"`).Output()
	if err != nil {
		return out, fmt.Errorf("protected native scan failed")
	}
	return out, json.Unmarshal(raw, &out)
}
