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
	Relay    string `json:"relayServerAddress"`
	IP       string `json:"netbirdIp"`
	Status   string `json:"status"`
	Type     string `json:"connectionType"`
	Endpoint struct {
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
	Relay               Endpoint           `json:"relay_endpoint"`
	UsesKernelInterface bool               `json:"uses_kernel_interface"`
	Start               Clock              `json:"start"`
	End                 Clock              `json:"end"`
	Commands            map[string]Command `json:"commands"`
	Links               []Link             `json:"links"`
	Routes              map[string][]Route `json:"routes"`
	PeerPresent         bool               `json:"peer_present"`
	PeerConnected       bool               `json:"peer_connected"`
	ConnectionType      string             `json:"connection_type"`
	Overlay             string             `json:"overlay"`
	Local               Endpoint           `json:"local"`
	Remote              Endpoint           `json:"remote"`
	WireGuard           Endpoint           `json:"wireguard_endpoint"`
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
	if target != "" {
		rawStatus := run("peer_status", "netbird", "status", "--json")
		var status Status
		_ = json.Unmarshal(rawStatus, &status)
		out.UsesKernelInterface = status.UsesKernelInterface
		p, e := Select(rawStatus, target)
		if e == nil {
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
			var key string
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
					if strings.Split(prefix, "/")[0] == strings.Split(target, "/")[0] {
						key = f[0]
					}
				}
			}
			endpoints := run("wg_endpoints", "wg", "show", tunnel, "endpoints")
			for _, line := range strings.Split(string(endpoints), "\n") {
				f := strings.Fields(line)
				if len(f) == 2 && key != "" && f[0] == key {
					out.WireGuard = EndpointView(f[1])
					host, _, _ := net.SplitHostPort(f[1])
					route("wireguard_route", host)
				}
			}

		}
	}
	out.End, err = Now()
	return out, err
}
