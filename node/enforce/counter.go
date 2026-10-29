package enforce

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Counter names are stable for the life of a device. They are never deleted
// and recreated by a ruleset render (§10.3, §10.7).
const (
	counterV6      = "ctv6"
	counterUnknown = "ctunku"
	counterUnkDown = "ctunkd"
	counterDenied  = "ctdeny"
	counterHost    = "cthost"
	counterBadDNS  = "ctbadns"
	counterInvalid = "ctinv"
	counterInput   = "ctin"
	counterBase    = "ctbase"
	counterNoDown  = "ctnodn"
)

func usageCounter(direction string, device uuid.UUID) string {
	switch direction {
	case dirUpload:
		return "u_" + hexID(device)
	case dirDownload:
		return "d_" + hexID(device)
	case dirDNSUpload:
		return "nu_" + hexID(device)
	case dirDNSDownload:
		return "nd_" + hexID(device)
	default:
		return ""
	}
}

func hexID(id uuid.UUID) string {
	return strings.ReplaceAll(id.String(), "-", "")
}

func parseDeviceCounter(name string) (device uuid.UUID, direction, namespace string, ok bool) {
	var prefix, dir, ns string
	switch {
	case strings.HasPrefix(name, "u_"):
		prefix, dir, ns = "u_", dirUpload, MeteringVersion
	case strings.HasPrefix(name, "d_"):
		prefix, dir, ns = "d_", dirDownload, MeteringVersion
	case strings.HasPrefix(name, "nu_"):
		prefix, dir, ns = "nu_", dirDNSUpload, DNSNamespace
	case strings.HasPrefix(name, "nd_"):
		prefix, dir, ns = "nd_", dirDNSDownload, DNSNamespace
	default:
		return uuid.Nil, "", "", false
	}
	id, err := uuid.Parse(name[len(prefix):])
	if err != nil {
		return uuid.Nil, "", "", false
	}
	return id, dir, ns, true
}

func diagnosticNamespace(name string) bool {
	switch name {
	case counterV6, counterUnknown, counterUnkDown, counterDenied, counterHost,
		counterBadDNS, counterInvalid, counterInput, counterBase, counterNoDown:
		return true
	default:
		return false
	}
}

// DetectResets compares absolute readings. A drop or a missing counter is a
// meter-epoch change, never a negative delta (§10.5, §10.7).
func DetectResets(previous, current map[string]CounterValue) []Reset {
	if len(previous) == 0 {
		return nil
	}
	var out []Reset
	for name, prev := range previous {
		cur, ok := current[name]
		if !ok {
			out = append(out, Reset{
				Name: name, Previous: prev.Bytes, Current: 0,
				Reason: ReasonMeterEpoch,
				Detail: "counter disappeared",
			})
			continue
		}
		if cur.Bytes < prev.Bytes || cur.Packets < prev.Packets {
			out = append(out, Reset{
				Name: name, Previous: prev.Bytes, Current: cur.Bytes,
				Reason: ReasonMeterEpoch,
				Detail: fmt.Sprintf("counter went backwards (packets %d -> %d, bytes %d -> %d)", prev.Packets, cur.Packets, prev.Bytes, cur.Bytes),
			})
		}
	}
	sortResets(out)
	return out
}

func sortResets(out []Reset) {
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j].Name < out[j-1].Name {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
}
