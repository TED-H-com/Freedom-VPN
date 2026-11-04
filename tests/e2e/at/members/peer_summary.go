package members

import (
	"encoding/json"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// Narrow input types and an independent output allowlist keep native identities,
// keys, endpoints, DNS names and arbitrary future fields out of the evidence.
// NetBird v0.82.0 client/status PeerStateDetailOutput uses numeric bytes/ns.
type nativePeerSummary struct {
	Available               bool     `json:"available"`
	Found                   bool     `json:"found"`
	Status                  string   `json:"connection_status"`
	Transport               string   `json:"transport"`
	HandshakeAgeNS          *string  `json:"last_wireguard_handshake_age_ns"`
	Received                *string  `json:"received_bytes"`
	Sent                    *string  `json:"sent_bytes"`
	LatencyNS               *string  `json:"latency_ns"`
	Routes                  []string `json:"route_prefixes"`
	IncludesSelectedNetwork bool     `json:"routes_include_selected_network"`
}

func nativePeerView(raw []byte, target, network string, at time.Time) nativePeerSummary {
	out := nativePeerSummary{Status: "unavailable", Transport: "unknown", Routes: []string{}}
	var s struct {
		Peers struct {
			Details []struct {
				IP        string    `json:"netbirdIp"`
				Status    string    `json:"status"`
				Type      string    `json:"connectionType"`
				Handshake time.Time `json:"lastWireguardHandshake"`
				Received  *uint64   `json:"transferReceived"`
				Sent      *uint64   `json:"transferSent"`
				Latency   *int64    `json:"latency"`
				Networks  []string  `json:"networks"`
			} `json:"details"`
		} `json:"peers"`
	}
	if json.Unmarshal(raw, &s) != nil {
		return out
	}
	out.Available = true
	addr, err := netip.ParseAddr(strings.Split(target, "/")[0])
	if err != nil {
		out.Status = "target_unavailable"
		return out
	}
	out.Status = "missing"
	for _, p := range s.Peers.Details {
		candidate, e := netip.ParseAddr(strings.Split(p.IP, "/")[0])
		if e != nil || candidate != addr {
			continue
		}
		out.Found = true
		out.Status = "unknown"
		switch strings.ToLower(p.Status) {
		case "connected", "connecting", "disconnected", "idle":
			out.Status = strings.ToLower(p.Status)
		}
		switch strings.ToLower(p.Type) {
		case "p2p":
			out.Transport = "direct"
		case "relayed":
			out.Transport = "relayed"
		}
		if !p.Handshake.IsZero() && !p.Handshake.After(at) {
			age := strconv.FormatInt(at.Sub(p.Handshake).Nanoseconds(), 10)
			out.HandshakeAgeNS = &age
		}
		if p.Received != nil {
			v := strconv.FormatUint(*p.Received, 10)
			out.Received = &v
		}
		if p.Sent != nil {
			v := strconv.FormatUint(*p.Sent, 10)
			out.Sent = &v
		}
		if p.Latency != nil && *p.Latency >= 0 {
			v := strconv.FormatInt(*p.Latency, 10)
			out.LatencyNS = &v
		}
		selected, _ := netip.ParsePrefix(network)
		for _, value := range p.Networks {
			prefix, e := netip.ParsePrefix(value)
			if e != nil {
				continue
			}
			out.Routes = append(out.Routes, prefix.Masked().String())
			out.IncludesSelectedNetwork = out.IncludesSelectedNetwork || prefix.Masked() == selected.Masked()
		}
		return out
	}
	return out
}
