//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

// DNSPathObservation contains only counter evidence and DNS response metadata.
// Query names and packet contents are deliberately absent.
type DNSPathObservation struct {
	Transport        string
	Reply            string
	DNSUpload        uint64 `json:"dns_upload_bytes,string"`
	DNSDownload      uint64 `json:"dns_download_bytes,string"`
	InternetUpload   uint64 `json:"internet_upload_bytes,string"`
	InternetDownload uint64 `json:"internet_download_bytes,string"`
	QueryStatus      string `json:"query_status"`
	QueryStage       string `json:"query_failure_stage,omitempty"`
	QueryClass       string `json:"query_failure_class,omitempty"`
	BeforeCounters   string `json:"before_counter_status"`
	AfterCounters    string `json:"after_counter_status"`
	// The four byte deltas above are measured only when this is "available".
	// Both snapshots must contain nu_, nd_, u_, d_ for this device, with
	// nondecreasing byte AND packet counts. Unavailable numeric zeros are defaults.
	CounterDeltas string `json:"counter_delta_status"`
}

// ObserveDesignatedDNS queries the product-issued designated resolver through
// the selected exit. It uses the admitted native member and never edits nft.
func (s *Stack) ObserveDesignatedDNS(ctx context.Context, device uuid.UUID, tcp bool) (DNSPathObservation, error) {
	result := newDNSPathObservation(tcp)
	bin, err := s.memberSessionBinary(ctx)
	if err != nil {
		return result, fmt.Errorf("designated DNS helper build failed")
	}
	if err = dockerCopy(ctx, bin, s.lab.member+":"+memberSessionBin); err != nil {
		return result, fmt.Errorf("designated DNS helper copy failed")
	}
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.node, "nft", "list", "set", "inet", "fvpn", "resolver_v4").Output()
	if err != nil {
		return result, fmt.Errorf("designated DNS resolver read failed")
	}
	resolver := firstIPv4(string(raw))
	if resolver == "" {
		return result, fmt.Errorf("designated IPv4 resolver absent")
	}
	read := func() ([]byte, error) {
		return exec.CommandContext(ctx, "docker", "exec", s.lab.node, "nft", "-j", "list", "table", "inet", "fvpn").Output()
	}
	query := func() ([]byte, error) {
		args := []string{"exec", s.lab.member, memberSessionBin, "-dns", resolver + ":53", "-name", "lab.example.invalid"}
		if tcp {
			args = append(args, "-dns-tcp")
		}
		return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	}
	return observeDNSQuery(result, device, read, query)
}

func newDNSPathObservation(tcp bool) DNSPathObservation {
	result := DNSPathObservation{Transport: "udp", QueryStatus: "not_attempted", BeforeCounters: "not_attempted", AfterCounters: "not_attempted", CounterDeltas: "unavailable"}
	if tcp {
		result.Transport = "tcp"
	}
	return result
}

// The command closures keep the unchanged single-query success sequence
// testable without Docker. The after snapshot is also retained on query failure.
func observeDNSQuery(result DNSPathObservation, device uuid.UUID, read, query func() ([]byte, error)) (DNSPathObservation, error) {
	suffix := strings.ReplaceAll(device.String(), "-", "")
	names := [4]string{"nu_" + suffix, "nd_" + suffix, "u_" + suffix, "d_" + suffix}
	before, status := dnsCounterSnapshot(names, read)
	result.BeforeCounters = status
	if status == "read_error" || status == "parse_error" {
		return result, fmt.Errorf("designated DNS before counters: %s", status)
	}
	out, queryErr := query()
	result.Reply, result.QueryStage, result.QueryClass = safeDNSReply(out)
	result.QueryStatus = "succeeded"
	if queryErr != nil || result.QueryStage != "" {
		result.QueryStatus = "failed"
		if result.QueryStage == "" {
			result.QueryStage, result.QueryClass = "command", "command_error"
		}
		queryErr = fmt.Errorf("designated DNS query: %s class=%s", result.QueryStage, result.QueryClass)
	}
	// A failed query still needs its after snapshot. Preserve the query failure
	// even if this read also fails; snapshot status records that second failure.
	after, status := dnsCounterSnapshot(names, read)
	result.AfterCounters = status
	if result.BeforeCounters == "available" && result.AfterCounters == "available" {
		result.CounterDeltas = "available"
		for _, name := range names {
			if after[name].Bytes < before[name].Bytes || after[name].Packets < before[name].Packets {
				result.CounterDeltas = "counter_reset"
				break
			}
		}
		if result.CounterDeltas == "available" {
			result.DNSUpload = after[names[0]].Bytes - before[names[0]].Bytes
			result.DNSDownload = after[names[1]].Bytes - before[names[1]].Bytes
			result.InternetUpload = after[names[2]].Bytes - before[names[2]].Bytes
			result.InternetDownload = after[names[3]].Bytes - before[names[3]].Bytes
		}
	}
	if queryErr != nil {
		return result, queryErr
	}
	if result.CounterDeltas != "available" {
		return result, fmt.Errorf("designated DNS counters unavailable: before=%s after=%s delta=%s", result.BeforeCounters, result.AfterCounters, result.CounterDeltas)
	}
	return result, nil
}

func dnsCounterSnapshot(names [4]string, read func() ([]byte, error)) (map[string]enforce.CounterValue, string) {
	raw, err := read()
	if err != nil {
		return nil, "read_error"
	}
	state, err := enforce.ParseState(raw)
	if err != nil {
		return nil, "parse_error"
	}
	for _, name := range names {
		if _, ok := state.Counters[name]; !ok {
			return state.Counters, "missing_counter"
		}
	}
	// ParseState intentionally defaults absent/null counter numbers to zero.
	// Measurement evidence requires both original JSON integer fields to exist.
	var doc struct {
		Nftables []struct {
			Counter *struct {
				Family  string          `json:"family"`
				Table   string          `json:"table"`
				Name    string          `json:"name"`
				Bytes   json.RawMessage `json:"bytes"`
				Packets json.RawMessage `json:"packets"`
			} `json:"counter"`
		} `json:"nftables"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, "parse_error"
	}
	for _, object := range doc.Nftables {
		counter := object.Counter
		if counter == nil || counter.Family != "" && counter.Family != "inet" || counter.Table != "" && counter.Table != "fvpn" {
			continue
		}
		for _, name := range names {
			if counter.Name != name {
				continue
			}
			for _, number := range []json.RawMessage{counter.Bytes, counter.Packets} {
				if _, err := strconv.ParseUint(strings.TrimSpace(string(number)), 10, 64); err != nil {
					return nil, "parse_error"
				}
			}
		}
	}
	return state.Counters, "available"
}

// Accept only the helper's complete, bounded metadata line. In particular,
// mixed output and Docker errors must not copy names, addresses or raw stderr.
func safeDNSReply(raw []byte) (reply, stage, class string) {
	if len(raw) > 160 {
		return "", "command", "unrecognized_output"
	}
	line := strings.TrimSpace(string(raw))
	parts := strings.Split(line, " ")
	if len(parts) == 2 && strings.HasPrefix(parts[0], "rcode=") && strings.HasPrefix(parts[1], "answers=") {
		rcode, e1 := strconv.ParseUint(strings.TrimPrefix(parts[0], "rcode="), 10, 4)
		answers, e2 := strconv.ParseUint(strings.TrimPrefix(parts[1], "answers="), 10, 16)
		if e1 == nil && e2 == nil && line == fmt.Sprintf("rcode=%d answers=%d", rcode, answers) {
			return line, "", ""
		}
	}
	parts = strings.Split(strings.TrimPrefix(line, "fvpn-e2e-member-session: dns "), " class=")
	if strings.HasPrefix(line, "fvpn-e2e-member-session: dns ") && len(parts) == 2 {
		switch parts[0] {
		case "dial", "write", "read":
			switch parts[1] {
			case "timeout", "canceled", "eof", "closed", "refused", "reset", "unreachable", "io_error":
				return "", parts[0], parts[1]
			}
		}
	}
	return "", "command", "unrecognized_output"
}

func firstIPv4(text string) string {
	for _, field := range strings.Fields(strings.ReplaceAll(text, ",", " ")) {
		field = strings.Trim(field, "{};")
		ip := net.ParseIP(field)
		if ip != nil && ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}
