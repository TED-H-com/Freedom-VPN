// Package nativecapture exports TEST ONLY header/path metadata, never peer keys.
package nativecapture

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

type Clock struct {
	BootID string        `json:"boot_id"`
	Before time.Duration `json:"mono_before_ns,string"`
	Wall   time.Time     `json:"wall"`
	After  time.Duration `json:"mono_after_ns,string"`
}

func Now() (Clock, error) {
	var c Clock
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return c, err
	}
	c.BootID = strings.TrimSpace(string(b))
	c.Before, err = enforce.MonotonicNow()
	if err != nil {
		return c, err
	}
	c.Wall = time.Now().UTC()
	c.After, err = enforce.MonotonicNow()
	return c, err
}
func Token(ip string) string {
	if ip == "" {
		return ""
	}
	ip = strings.Split(ip, "/")[0]
	if addr := net.ParseIP(ip); addr != nil {
		ip = addr.String()
	}
	h := sha256.Sum256([]byte(ip))
	return fmt.Sprintf("addr-%x", h[:8])
}

// Status is deliberately a narrow decoder. Omitted native keys never escape.
type Status struct {
	UsesKernelInterface bool   `json:"usesKernelInterface"`
	IP                  string `json:"netbirdIp"`
	Peers               struct {
		Details []Peer `json:"details"`
	} `json:"peers"`
}
type Peer struct {
	LastHandshake time.Time `json:"lastWireguardHandshake"`
	Relay         string    `json:"relayServerAddress"`
	IP            string    `json:"netbirdIp"`
	Status        string    `json:"status"`
	Type          string    `json:"connectionType"`
	Endpoint      struct {
		Local  string `json:"local"`
		Remote string `json:"remote"`
	} `json:"iceCandidateEndpoint"`
}

func Select(raw []byte, target string) (Peer, error) {
	var s Status
	if json.Unmarshal(raw, &s) != nil {
		return Peer{}, fmt.Errorf("status malformed")
	}
	for _, p := range s.Peers.Details {
		if strings.Split(p.IP, "/")[0] == strings.Split(target, "/")[0] && target != "" {
			return p, nil
		}
	}
	return Peer{}, fmt.Errorf("selected peer absent")
}

type Endpoint struct {
	Address string `json:"address"`
	Port    string `json:"port"`
	Family  string `json:"family"`
}

func EndpointView(value string) Endpoint {
	host, port, err := net.SplitHostPort(value)
	if err != nil || net.ParseIP(host) == nil {
		return Endpoint{}
	}
	family := "ipv6"
	if net.ParseIP(host).To4() != nil {
		family = "ipv4"
	}
	return Endpoint{Token(host), port, family}
}

type Link struct {
	Index int    `json:"ifindex"`
	Name  string `json:"ifname"`
	MTU   int    `json:"mtu"`
	Info  struct {
		Kind string `json:"info_kind"`
	} `json:"linkinfo"`
}
type Route struct {
	Destination string           `json:"dst"`
	Device      string           `json:"dev"`
	Gateway     string           `json:"gateway,omitempty"`
	Source      string           `json:"prefsrc,omitempty"`
	Table       any              `json:"table,omitempty"`
	Metrics     []map[string]any `json:"metrics,omitempty"`
}
type Command struct {
	Start Clock `json:"start"`
	End   Clock `json:"end"`
	OK    bool  `json:"ok"`
}
type Snapshot struct {
	NativeLatestHandshake *time.Time         `json:"native_latest_handshake_at,omitempty"`
	Relay                 Endpoint           `json:"relay_endpoint"`
	UsesKernelInterface   bool               `json:"uses_kernel_interface"`
	Start                 Clock              `json:"start"`
	End                   Clock              `json:"end"`
	Commands              map[string]Command `json:"commands"`
	Links                 []Link             `json:"links"`
	Routes                map[string][]Route `json:"routes"`
	PeerPresent           bool               `json:"peer_present"`
	PeerConnected         bool               `json:"peer_connected"`
	ConnectionType        string             `json:"connection_type"`
	Overlay               string             `json:"overlay"`
	Local                 Endpoint           `json:"local"`
	Remote                Endpoint           `json:"remote"`
	LatestHandshake       string             `json:"latest_handshake_unix_seconds,omitempty"`
	Kernel                *KernelDiagnostics `json:"kernel,omitempty"`
	WireGuardPeerMatched  bool               `json:"wireguard_peer_matched"`
	WireGuard             Endpoint           `json:"wireguard_endpoint"`
}

func Read(ctx context.Context, target, origin string) (Snapshot, error) {
	out := Snapshot{Commands: map[string]Command{}, Routes: map[string][]Route{}}
	var err error
	out.Start, err = Now()
	if err != nil {
		return out, err
	}
	run := func(name string, args ...string) []byte {
		a, e := Now()
		if e != nil {
			return nil
		}
		raw, e := exec.CommandContext(ctx, args[0], args[1:]...).Output()
		b, ce := Now()
		out.Commands[name] = Command{a, b, e == nil && ce == nil}
		if e != nil || ce != nil {
			return nil
		}
		return raw
	}
	_ = json.Unmarshal(run("links", "ip", "-j", "link"), &out.Links)
	route := func(name, dest string) {
		if dest == "" {
			return
		}
		var rows []Route
		if json.Unmarshal(run(name, "ip", "-j", "route", "get", dest), &rows) != nil {
			return
		}
		for i := range rows {
			rows[i].Destination = Token(rows[i].Destination)
			rows[i].Gateway = Token(rows[i].Gateway)
			rows[i].Source = Token(rows[i].Source)
		}
		out.Routes[name] = rows
	}
	route("origin_route", origin)
	// Full route tables complement route-get's single selected path. Keep only
	// the existing metadata shape; tokenize addresses before public export.
	for _, family := range []string{"4", "6"} {
		name := "route_table" + family
		var rows []Route
		if json.Unmarshal(run(name, "ip", "-j", "-"+family, "route", "show", "table", "all"), &rows) == nil {
			for i := range rows {
				if rows[i].Destination != "default" {
					rows[i].Destination = Token(rows[i].Destination)
				}
				rows[i].Gateway = Token(rows[i].Gateway)
				rows[i].Source = Token(rows[i].Source)
			}
			out.Routes[name] = rows
		}
	}
	if target != "" {
		rawStatus := run("peer_status", "netbird", "status", "--json")
		var status Status
		_ = json.Unmarshal(rawStatus, &status)
		out.UsesKernelInterface = status.UsesKernelInterface
		p, e := Select(rawStatus, target)
		if e == nil {
			// Native daemon telemetry is separate from wg's independent readback.
			// Keep an unavailable userspace CLI sample explicitly unavailable.
			// A missing/zero daemon timestamp stays absent, never a fabricated
			// successful handshake or a replacement for a failed wg command.
			if !p.LastHandshake.IsZero() {
				stamp := p.LastHandshake.UTC()
				out.NativeLatestHandshake = &stamp
			}
			out.PeerPresent = true
			out.PeerConnected = strings.EqualFold(p.Status, "Connected")
			out.ConnectionType = p.Type
			out.Overlay = Token(p.IP)
			out.Local = EndpointView(p.Endpoint.Local)
			out.Remote = EndpointView(p.Endpoint.Remote)
			if relay, e := url.Parse(p.Relay); e == nil && relay.Host != "" {
				port := relay.Port()
				if port == "" {
					port = "443"
				}
				host := relay.Hostname()
				if net.ParseIP(host) == nil && strings.HasSuffix(host, ".example.invalid") {
					a, e := net.DefaultResolver.LookupIPAddr(ctx, host)
					if e == nil && len(a) > 0 {
						host = a[0].IP.String()
					}
				}
				if net.ParseIP(host) != nil {
					out.Relay = EndpointView(net.JoinHostPort(host, port))
					route("relay_route", host)
				}
			}

			host, _, _ := net.SplitHostPort(p.Endpoint.Remote)
			route("peer_route", host)
			route("overlay_route", strings.Split(target, "/")[0])
			// Query public allowed-IP/endpoint views only; never `wg show ... dump`
			// (which includes the private key). Match the selected overlay /32.
			key := selectedPublicKey(rawStatus, target)
			tunnel := "fvnb0"
			if routes := out.Routes["overlay_route"]; len(routes) == 1 {
				tunnel = routes[0].Device
			}
			allowed := run("wg_allowed_ips", "wg", "show", tunnel, "allowed-ips")
			for _, line := range strings.Split(string(allowed), "\n") {
				f := strings.Fields(line)
				if len(f) < 2 {
					continue
				}
				for _, prefix := range strings.Split(strings.Join(f[1:], ""), ",") {
					if key == "" && strings.Split(prefix, "/")[0] == strings.Split(target, "/")[0] {
						key = f[0]
					}
				}
			}
			handshakes := run("wg_latest_handshakes", "wg", "show", tunnel, "latest-handshakes")
			for _, line := range strings.Split(string(handshakes), "\n") {
				f := strings.Fields(line)
				if len(f) == 2 && key != "" && f[0] == key {
					if _, e := strconv.ParseUint(f[1], 10, 64); e == nil {
						out.LatestHandshake = f[1]
					}
				}
			}
			endpoints := run("wg_endpoints", "wg", "show", tunnel, "endpoints")
			for _, line := range strings.Split(string(endpoints), "\n") {
				f := strings.Fields(line)
				if len(f) == 2 && key != "" && f[0] == key {
					out.WireGuardPeerMatched = true
					out.WireGuard = EndpointView(f[1])
					host, _, _ := net.SplitHostPort(f[1])
					route("wireguard_route", host)
				}
			}

		}
	}
	if os.Getenv("FVPN_E2E_KERNEL_OBSERVE") == "1" {
		out.Kernel = &KernelDiagnostics{
			SNMP4:             SNMP4Counters(run("snmp4", "cat", "/proc/net/snmp")),
			SNMP6:             NumericCounters(run("snmp6", "cat", "/proc/net/snmp6")),
			Nstat:             NumericCounters(run("nstat", "nstat", "-asz")),
			LoopbackConntrack: LoopbackConntrack(run("conntrack6", "conntrack", "-L", "-f", "ipv6", "-p", "udp")),
		}
	}
	out.End, err = Now()
	return out, err
}

// A selected exit can replace its exact overlay /32 with broader allowed IPs.
// Use its native status key only in memory for the wg join; never serialize it.
func selectedPublicKey(raw []byte, target string) string {
	var status struct {
		Peers struct {
			Details []struct {
				IP  string `json:"netbirdIp"`
				Key string `json:"publicKey"`
			} `json:"details"`
		} `json:"peers"`
	}
	if json.Unmarshal(raw, &status) != nil || target == "" {
		return ""
	}
	for _, p := range status.Peers.Details {
		if strings.Split(p.IP, "/")[0] == strings.Split(target, "/")[0] {
			return p.Key
		}
	}
	return ""
}
