//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net"
	"os/exec"
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
}

// ObserveDesignatedDNS queries the product-issued designated resolver through
// the selected exit. It uses the admitted native member and never edits nft.
func (s *Stack) ObserveDesignatedDNS(ctx context.Context, device uuid.UUID, tcp bool) (DNSPathObservation, error) {
	result := DNSPathObservation{Transport: "udp"}
	if tcp {
		result.Transport = "tcp"
	}
	read := func() (map[string]enforce.CounterValue, error) {
		raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.node, "nft", "-j", "list", "table", "inet", "fvpn").Output()
		if err != nil {
			return nil, err
		}
		state, err := enforce.ParseState(raw)
		return state.Counters, err
	}
	bin, err := s.memberSessionBinary(ctx)
	if err != nil {
		return result, err
	}
	if err = dockerCopy(ctx, bin, s.lab.member+":"+memberSessionBin); err != nil {
		return result, err
	}
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.node, "nft", "list", "set", "inet", "fvpn", "resolver_v4").Output()
	if err != nil {
		return result, err
	}
	resolver := firstIPv4(string(raw))
	if resolver == "" {
		return result, fmt.Errorf("designated IPv4 resolver absent")
	}
	before, err := read()
	if err != nil {
		return result, err
	}
	args := []string{"exec", s.lab.member, memberSessionBin, "-dns", resolver + ":53", "-name", "lab.example.invalid"}
	if tcp {
		args = append(args, "-dns-tcp")
	}
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	result.Reply = strings.TrimSpace(string(out))
	if err != nil {
		return result, fmt.Errorf("designated DNS query: %s", clipText(result.Reply, 160))
	}
	after, err := read()
	if err != nil {
		return result, err
	}
	suffix := strings.ReplaceAll(device.String(), "-", "")
	delta := func(prefix string) uint64 {
		a, b := after[prefix+suffix].Bytes, before[prefix+suffix].Bytes
		if a < b {
			return 0
		}
		return a - b
	}
	result.DNSUpload, result.DNSDownload = delta("nu_"), delta("nd_")
	result.InternetUpload, result.InternetDownload = delta("u_"), delta("d_")
	return result, nil
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
