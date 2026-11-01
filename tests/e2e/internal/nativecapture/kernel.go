package nativecapture

import (
	"net"
	"strconv"
	"strings"
)

// KernelDiagnostics contains only counters and loopback UDP tuple metadata.
// Commands retain read success independently; an unavailable read is not zero.
type KernelDiagnostics struct {
	SNMP6             map[string]string `json:"snmp6"`
	Nstat             map[string]string `json:"nstat"`
	LoopbackConntrack []ConntrackTuple  `json:"loopback_conntrack"`
}
type ConntrackTuple struct {
	Source          string `json:"source"`
	Destination     string `json:"destination"`
	SourcePort      string `json:"source_port"`
	DestinationPort string `json:"destination_port"`
}

func NumericCounters(raw []byte) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		if _, err := strconv.ParseUint(f[1], 10, 64); err != nil {
			continue
		}
		// Kernel counter names contain only letters, digits and underscore.
		valid := true
		for _, c := range f[0] {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
				valid = false
			}
		}
		if valid {
			out[f[0]] = f[1]
		}
	}
	return out
}
func LoopbackConntrack(raw []byte) []ConntrackTuple {
	out := []ConntrackTuple{}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "udp ") {
			continue
		}
		var tuple ConntrackTuple
		loopback := false
		for _, field := range strings.Fields(line) {
			k, v, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			switch k {
			case "src":
				tuple = ConntrackTuple{}
				loopback = false
				if ip := net.ParseIP(v); ip != nil {
					tuple.Source = Token(v)
					loopback = ip.IsLoopback()
				}
			case "dst":
				if ip := net.ParseIP(v); ip != nil {
					tuple.Destination = Token(v)
					loopback = loopback || ip.IsLoopback()
				}
			case "sport":
				if _, e := strconv.ParseUint(v, 10, 16); e == nil {
					tuple.SourcePort = v
				}
			case "dport":
				if _, e := strconv.ParseUint(v, 10, 16); e == nil {
					tuple.DestinationPort = v
				}
				if loopback && tuple.Source != "" && tuple.Destination != "" && tuple.SourcePort != "" && tuple.DestinationPort != "" {
					out = append(out, tuple)
				}
			}
		}
	}
	return out
}
