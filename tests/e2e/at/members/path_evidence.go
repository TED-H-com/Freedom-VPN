package members

import (
	"encoding/json"
	"net"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture"
)

// Independent allowlists: none of these output types can contain a raw address,
// key, hostname, arbitrary tool error or an unrecognised future tool field.
func safePathEndpoint(value string) nativecapture.Endpoint {
	host, port, err := net.SplitHostPort(value)
	if err != nil || net.ParseIP(host) == nil {
		return nativecapture.Endpoint{}
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return nativecapture.Endpoint{}
	}
	return nativecapture.EndpointView(value)
}

type pathRoute struct {
	Family      string `json:"family"`
	Interface   string `json:"interface"`
	Destination string `json:"destination_label"`
	Source      string `json:"source_label"`
	Gateway     string `json:"gateway_label"`
}

func addressLabel(s string) string {
	a, err := netip.ParseAddr(strings.Split(s, "/")[0])
	if err != nil {
		return ""
	}
	return nativecapture.Token(a.WithZone("").String())
}
func addressFamily(s string) string {
	a, e := netip.ParseAddr(strings.Split(s, "/")[0])
	if e != nil {
		return "unavailable"
	}
	if a.Is4() {
		return "ipv4"
	}
	return "ipv6"
}

var interfaceName = regexp.MustCompile(`^(eth[0-9]+|fv[a-zA-Z0-9_-]*|lo)$`)

func safeInterface(s string) string {
	if interfaceName.MatchString(s) {
		return s
	}
	return "unavailable"
}
func parsePathRoutes(raw []byte) []pathRoute {
	var rows []struct{ Dst, Dev, Prefsrc, Src, Gateway string }
	out := []pathRoute{}
	if json.Unmarshal(raw, &rows) != nil {
		return out
	}
	for _, r := range rows {
		source := r.Prefsrc
		if source == "" {
			source = r.Src
		}
		out = append(out, pathRoute{addressFamily(r.Dst), safeInterface(r.Dev), addressLabel(r.Dst), addressLabel(source), addressLabel(r.Gateway)})
	}
	return out
}

type pathNeighbour struct {
	Address   string   `json:"address_label"`
	Interface string   `json:"interface"`
	States    []string `json:"states"`
}

func parsePathNeighbours(raw []byte) []pathNeighbour {
	var rows []struct {
		Dst, Dev string
		State    []string
	}
	out := []pathNeighbour{}
	if json.Unmarshal(raw, &rows) != nil {
		return out
	}
	for _, r := range rows {
		n := pathNeighbour{addressLabel(r.Dst), safeInterface(r.Dev), []string{}}
		for _, s := range r.State {
			switch s {
			case "INCOMPLETE", "REACHABLE", "STALE", "DELAY", "PROBE", "FAILED", "NOARP", "PERMANENT", "NONE":
				n.States = append(n.States, s)
			}
		}
		out = append(out, n)
	}
	return out
}

type wireGuardView struct {
	Found     bool                   `json:"found"`
	Interface string                 `json:"interface"`
	Endpoint  nativecapture.Endpoint `json:"endpoint"`
	Handshake string                 `json:"latest_handshake_unix_seconds"`
	Received  string                 `json:"received_bytes"`
	Sent      string                 `json:"sent_bytes"`
}

func decimalField(s string) string {
	if _, e := strconv.ParseUint(s, 10, 64); e == nil {
		return s
	}
	return ""
}

// Accept wg show all dump, discard interface/private-key lines and match the
// selected peer only in memory. Never serialize either key or allowed-IP data.
func parseWireGuardDump(raw []byte, key, overlay string) wireGuardView {
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) != 9 {
			continue
		}
		match := key != "" && f[1] == key
		if key == "" && overlay != "" {
			for _, p := range strings.Split(f[4], ",") {
				match = match || strings.Split(p, "/")[0] == strings.Split(overlay, "/")[0]
			}
		}
		if match {
			return wireGuardView{true, safeInterface(f[0]), safePathEndpoint(f[3]), decimalField(f[5]), decimalField(f[6]), decimalField(f[7])}
		}
	}
	return wireGuardView{}
}

type tcpControlView struct {
	Family          string                 `json:"family"`
	State           string                 `json:"state"`
	Local           nativecapture.Endpoint `json:"local"`
	Remote          nativecapture.Endpoint `json:"remote"`
	Retransmissions string                 `json:"retransmissions"`
	RTT             string                 `json:"rtt_ms"`
	MSS             string                 `json:"mss"`
	PMTU            string                 `json:"pmtu"`
}

var tcpMetric = regexp.MustCompile(`^(retrans|rtt|mss|pmtu):([0-9]+(?:[./][0-9]+)*)$`)

func parseControlTCP(raw []byte, address, port string) []tcpControlView {
	out := []tcpControlView{}
	current := -1
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			current = -1
			if len(f) < 5 || !strings.Contains(line, `"fvpn-agent"`) {
				continue
			}
			h, p, e := net.SplitHostPort(f[4])
			if e != nil || net.ParseIP(h) == nil || addressLabel(h) != addressLabel(address) || p != port {
				continue
			}
			switch f[0] {
			case "ESTAB", "SYN-SENT", "CLOSE-WAIT", "FIN-WAIT-1", "FIN-WAIT-2", "LAST-ACK", "CLOSING", "TIME-WAIT":
			default:
				continue
			}
			out = append(out, tcpControlView{Family: addressFamily(h), State: f[0], Local: safePathEndpoint(f[3]), Remote: safePathEndpoint(f[4])})
			current = len(out) - 1
		}
		if current < 0 {
			continue
		}
		for _, v := range f {
			m := tcpMetric.FindStringSubmatch(v)
			if m == nil {
				continue
			}
			switch m[1] {
			case "retrans":
				out[current].Retransmissions = m[2]
			case "rtt":
				out[current].RTT = m[2]
			case "mss":
				out[current].MSS = m[2]
			case "pmtu":
				out[current].PMTU = m[2]
			}
		}
	}
	return out
}

type storedObservation struct {
	ReceivedAt  time.Time `json:"received_at"`
	ReportID    string    `json:"report_id"`
	Generation  *string   `json:"generation"`
	Incarnation string    `json:"incarnation"`
}

func observationsInWindow(raw []byte, start, end time.Time) []storedObservation {
	var rows []storedObservation
	out := []storedObservation{}
	if json.Unmarshal(raw, &rows) != nil {
		return out
	}
	for _, r := range rows {
		if !r.ReceivedAt.Before(start) && r.ReceivedAt.Before(end) {
			out = append(out, r)
		}
	}
	return out
}

var logTime = regexp.MustCompile(`(?:^|\s)time=([^\s]+)`)

// Current agent code has no per-submission log event. Preserve coverage and
// explicitly report that absence instead of presenting zero submissions.
func observationLogWindow(before, after []byte, start, end time.Time) map[string]any {
	var first, last *time.Time
	lines := 0
	for _, line := range strings.Split(string(after), "\n") {
		m := logTime.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		at, e := time.Parse(time.RFC3339Nano, strings.Trim(m[1], `"`))
		if e != nil {
			continue
		}
		if first == nil {
			first = &at
		}
		last = &at
		if !at.Before(start) && at.Before(end) {
			lines++
		}
	}
	prefix := len(after) >= len(before) && string(after[:len(before)]) == string(before)
	return map[string]any{"submission_events_available": false, "reason": "agent observe/flush and controlclient do not log individual submission times or outcomes", "start_bytes": strconv.Itoa(len(before)), "end_bytes": strconv.Itoa(len(after)), "start_content_retained_as_prefix": prefix, "overwrite_or_replacement_detected": !prefix, "first_retained_log_time": first, "last_retained_log_time": last, "timestamped_lines_in_window": lines, "window_start": start, "window_end": end, "coverage_limit": "timestamps bound retained log lines, not silent periods; replacement can lose the prefix; a shorter or replaced log cannot prove all submissions"}
}
