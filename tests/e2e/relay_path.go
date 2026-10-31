package e2e

import (
	"bytes"
	"encoding/json"
	"strings"
)

// classifyNetbird counts relayed and direct peers in netbird status output.
// It returns counts only. Peer identifiers and addresses are not retained.
func classifyNetbird(raw []byte) (relayed, direct int) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return 0, 0
	}
	if raw[0] == '{' || raw[0] == '[' {
		var v any
		if json.Unmarshal(raw, &v) == nil {
			walkNetbird(v, &relayed, &direct)
			return relayed, direct
		}
	}
	for _, line := range strings.Split(string(raw), "\n") {
		low := strings.ToLower(line)
		if !strings.Contains(low, "connection type") {
			continue
		}
		switch {
		case strings.Contains(low, "relay"):
			relayed++
		case strings.Contains(low, "p2p"), strings.Contains(low, "direct"):
			direct++
		}
	}
	return relayed, direct
}

func walkNetbird(v any, relayed, direct *int) {
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			walkNetbird(item, relayed, direct)
		}
	case map[string]any:
		if peer, rel, dir := classifyPeer(t); peer {
			if rel {
				*relayed++
			}
			if dir {
				*direct++
			}
			return
		}
		for _, item := range t {
			walkNetbird(item, relayed, direct)
		}
	}
}

func classifyPeer(m map[string]any) (peer, relayed, direct bool) {
	_, hasRelayed := m["relayed"]
	_, hasType := m["connectionType"]
	_, hasType2 := m["connection_type"]
	_, hasDirect := m["direct"]
	if !hasRelayed && !hasType && !hasType2 && !hasDirect {
		return false, false, false
	}
	if status, ok := m["status"].(string); !ok || !strings.EqualFold(status, "Connected") {
		return true, false, false
	}
	if b, ok := m["relayed"].(bool); ok {
		if b {
			return true, true, false
		}
		return true, false, true
	}
	if b, ok := m["direct"].(bool); ok && b {
		return true, false, true
	}
	kind := strings.ToLower(stringField(m, "connectionType"))
	if kind == "" {
		kind = strings.ToLower(stringField(m, "connection_type"))
	}
	switch {
	case strings.Contains(kind, "relay"):
		return true, true, false
	case strings.Contains(kind, "p2p"), strings.Contains(kind, "direct"):
		return true, false, true
	default:
		return true, false, false
	}
}

func stringField(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}
