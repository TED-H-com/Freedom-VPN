package enforce

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

const (
	// TableName is the only nftables table this package writes.
	TableName = "fvpn"
	// TableFamily is the inet family so one table covers IPv4 and IPv6.
	TableFamily = "inet"
	// MeteringVersion is the usage namespace for upload and download counters
	// (§10.1). Other counters must not be summed into this namespace.
	MeteringVersion = "forward_admission_v1"
	// DNSNamespace counts member packets to and from the designated resolver.
	// They are not Internet-forwarding usage (§10.1).
	DNSNamespace = "dns_service_v1"
	// DiagnosticNamespace counts drops and baseline hits. Not usage.
	DiagnosticNamespace = "diagnostic_v1"

	// HookPriorityFilter sits after conntrack (-200) and before the standard
	// filter hook (0), so every forwarded packet is checked with conntrack
	// state available and before POSTROUTING SNAT (§8.2, §10.2).
	HookPriorityFilter int64 = -150
	// HookPrioritySNAT is the standard srcnat priority.
	HookPrioritySNAT int64 = 100

	// HTBMajor is the root qdisc handle major. skb->priority is set to
	// (HTBMajor<<16)|minor so HTB selects that leaf without a classifier.
	HTBMajor uint32 = 1
	// HTBDefaultMinor is the leaf for host and unclassified packets.
	// Minor 0 is reserved: priority equal to the qdisc handle is HTB direct
	// (unshaped) traffic.
	HTBDefaultMinor uint16 = 0xFFFE

	// ProbeMaxSeconds is the longest synthetic-probe authorization (§13.9).
	// The kernel element timeout is derived from ExpiresAt and cannot be
	// longer than this.
	ProbeMaxSeconds = 300
	// ProbeMaxDestinations is the most exact sockets one probe may name.
	ProbeMaxDestinations = 4

	dirUpload      = "upload"
	dirDownload    = "download"
	dirDNSUpload   = "dns_upload"
	dirDNSDownload = "dns_download"
)

// IPv6Mode selects the single v1 forwarding mode (§8.3). Only block is
// implemented. Any other value is rejected so ::/0 cannot slip through.
type IPv6Mode string

const (
	// IPv6Block drops member IPv6 forwarding. IPv4 service remains.
	IPv6Block IPv6Mode = "block"
)

// Datapath names the interfaces the rendered rules bind to. Names must
// already exist; this package does not create links.
type Datapath struct {
	// MemberIngress faces the routing namespace. Upload arrives on it.
	// Download leaves through it. Lab name: fv-fh.
	MemberIngress string `json:"member_ingress"`
	// Uplink faces the Internet side. Upload leaves through it after SNAT.
	// Lab name: fv-ih.
	Uplink string `json:"uplink"`
	// SNAT is the source written on accepted member upload (§10.2).
	SNAT netip.Addr `json:"snat"`
}

// Binding is one device authorization (§10.2). Allowed addresses are host
// routes (/32 or /128); a wider prefix would admit a range we cannot attribute
// to a single device.
type Binding struct {
	MemberID        uuid.UUID      `json:"member_id"`
	DeviceID        uuid.UUID      `json:"device_id"`
	PeerID          string         `json:"peer_id"`
	AllowedV4       []netip.Prefix `json:"allowed_v4"`
	AllowedV6       []netip.Prefix `json:"allowed_v6,omitempty"`
	BindingRevision uint64         `json:"binding_revision"`
	ValidFrom       time.Time      `json:"valid_from"`
	ValidUntil      time.Time      `json:"valid_until"`
}

// MemberLease is the per-member service window (§16.3). The forwarding
// deadline is the earliest of the entitlement end, the node lease end, and
// signed_server_time plus the offline lease. TimeoutCeilSeconds, when set,
// caps the kernel element timeout so a later render cannot extend it.
type MemberLease struct {
	MemberID            uuid.UUID `json:"member_id"`
	EffectiveServiceEnd time.Time `json:"effective_service_end"`
	NodeLeaseEnd        time.Time `json:"node_lease_end"`
	SignedServerTime    time.Time `json:"signed_server_time"`
	MaxOfflineSeconds   uint64    `json:"max_offline_seconds"`
	TimeoutCeilSeconds  uint64    `json:"timeout_ceil_seconds,omitempty"`
}

// Rate is an HTB rate and ceil in bits per second. Rate is the guaranteed
// share. Ceil is the cap. Child guarantees must sum to at most the parent
// guarantee (§11.4).
type Rate struct {
	RateBPS uint64 `json:"rate_bps"`
	CeilBPS uint64 `json:"ceil_bps"`
}

// MemberQoS is one member class. Every device of that member shares it, per
// direction (§11.3).
type MemberQoS struct {
	MemberID uuid.UUID `json:"member_id"`
	Up       Rate      `json:"up"`
	Down     Rate      `json:"down"`
}

// QoS is the node capacity parent plus the member classes (§11.3, §11.4).
// Assignments remembers class minors so a later render does not move a member.
type QoS struct {
	Up          Rate                 `json:"up"`
	Down        Rate                 `json:"down"`
	HostUp      Rate                 `json:"host_up"`
	HostDown    Rate                 `json:"host_down"`
	Members     []MemberQoS          `json:"members"`
	Assignments map[uuid.UUID]uint16 `json:"assignments,omitempty"`
}

// Model is the desired data-plane generation.
type Model struct {
	Generation uint64        `json:"generation"`
	IPv6Mode   IPv6Mode      `json:"ipv6_mode"`
	Now        time.Time     `json:"now"`
	Datapath   Datapath      `json:"datapath"`
	Bindings   []Binding     `json:"bindings"`
	Leases     []MemberLease `json:"leases"`
	// Resolvers are the only destinations that may use TCP/UDP 53 (§8.1).
	Resolvers []netip.Addr `json:"resolvers"`
	// DeniedV4 replaces SpecDeniedV4 when non-nil. A nil slice means the spec
	// list. An empty non-nil slice is a mistake and is rejected.
	DeniedV4 []netip.Prefix `json:"denied_v4,omitempty"`
	// ExtraDeniedV4 is always unioned. Host-detected on-link and reachable
	// prefixes go here, including a private subnet that uses a public prefix
	// (§8.1).
	ExtraDeniedV4 []netip.Prefix `json:"extra_denied_v4,omitempty"`
	// HostV4 is dropped as a destination even when it is not in the deny list.
	// These are the node's own addresses (§8.1).
	HostV4 []netip.Prefix `json:"host_v4,omitempty"`
	QoS    QoS            `json:"qos"`
	// Revoke lists addresses whose conntrack entries are deleted after the
	// ruleset is installed (§8.2).
	Revoke []netip.Addr `json:"revoke,omitempty"`
	// Probe is the short-lived synthetic-probe exception (§13.9). Nil removes
	// it. The caller authenticates the test peer; this package matches SourceV4.
	Probe *ProbeException `json:"probe,omitempty"`
}

// ProbeDestination is one exact IPv4, protocol, and port the synthetic probe
// may reach. AddrV4 is a /32. Proto is "tcp" or "udp".
type ProbeDestination struct {
	AddrV4 netip.Prefix `json:"addr_v4"`
	Proto  string       `json:"proto"`
	Port   uint16       `json:"port"`
}

// ProbeException is the only forward accept allowed while member forwarding
// is blocked (§13.9). ExpiresAt is required. The kernel drops the elements
// when it passes, even if the agent is gone.
type ProbeException struct {
	// PeerID is the NetBird peer id of the dedicated test peer. It is not a
	// member device. Enforcement matches SourceV4; it does not authenticate.
	PeerID string `json:"peer_id"`
	// SourceV4 is the /32 overlay address of that peer.
	SourceV4 netip.Prefix `json:"source_v4"`
	// Destinations are the team-controlled Internet endpoints.
	Destinations []ProbeDestination `json:"destinations,omitempty"`
	// ResolverDNS also allows TCP and UDP destination port 53 to the model's
	// resolvers. On the full ruleset that is @resolver_v4. On the baseline
	// ruleset the same resolver addresses are installed for the exception.
	ResolverDNS bool      `json:"resolver_dns,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
	// TimeoutCeilSeconds caps the kernel timeout the same way as
	// MemberLease.TimeoutCeilSeconds. Zero means no extra cap. Render also
	// caps to the remaining lifetime already installed in State, so a
	// re-render cannot extend a live probe past its original ExpiresAt.
	TimeoutCeilSeconds uint64 `json:"timeout_ceil_seconds,omitempty"`
}

// CounterValue is an absolute nft named-counter reading (§10.5, §10.7).
type CounterValue struct {
	Packets uint64 `json:"packets"`
	Bytes   uint64 `json:"bytes"`
}

// Sample is one device counter joined to the binding that explains it.
type Sample struct {
	DeviceID        uuid.UUID `json:"device_id"`
	MemberID        uuid.UUID `json:"member_id"`
	PeerID          string    `json:"peer_id"`
	BindingRevision uint64    `json:"binding_revision"`
	Direction       string    `json:"direction"`
	Packets         uint64    `json:"packets"`
	Bytes           uint64    `json:"bytes"`
	Namespace       string    `json:"namespace"`
}

// Reset is a counter that went backwards or vanished (§10.7).
type Reset struct {
	Name     string `json:"name"`
	Previous uint64 `json:"previous_bytes"`
	Current  uint64 `json:"current_bytes"`
	Reason   Reason `json:"reason"`
	Detail   string `json:"detail"`
}

func (m Model) denyV4() ([]netip.Prefix, error) {
	base := m.DeniedV4
	if base == nil {
		base = SpecDeniedV4()
	}
	if len(base) == 0 {
		return nil, fmt.Errorf("denied_v4 is empty")
	}
	return mergePrefixes(append(append([]netip.Prefix{}, base...), m.ExtraDeniedV4...)), nil
}
