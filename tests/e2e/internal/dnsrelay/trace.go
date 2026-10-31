package dnsrelay

// TEST ONLY metadata for a closed set of synthetic fixture cases. No full
// query names, addresses, packet bodies or resolver history are retained.
import (
	"encoding/json"
	"golang.org/x/net/dns/dnsmessage"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Started         time.Time `json:"started"`
	Completed       time.Time `json:"completed"`
	Role            string    `json:"role"`
	Source          string    `json:"source_role"`
	Case            string    `json:"fixture_case"`
	Type            uint16    `json:"rr_type"`
	Protocol        string    `json:"protocol"`
	RCode           int       `json:"rcode"`
	TTLs            []uint32  `json:"ttls"`
	Classes         []string  `json:"address_classes"`
	AuthoritySerial uint32    `json:"authority_serial"`
	Success         bool      `json:"success"`
}
type Trace struct {
	Role, Path string
	mu         sync.Mutex
}

func FixtureCase(name string) string {
	name = strings.TrimSuffix(strings.ToLower(name), ".")
	for _, label := range []string{"cache", "absent", "rebind", "rr", "alias", "service", "feed-01", "blocked-cname", "blocked-alias", "sub.feed-01", "notfeed-01"} {
		if name == label+".example.invalid" {
			return strings.ReplaceAll(label, ".", "_")
		}
	}
	if name == "example.invalid" {
		return "authority_soa"
	}
	return ""
}
func (t *Trace) Record(request, response []byte, peer net.Addr, proto string, started time.Time, err error) {
	var q, a dnsmessage.Message
	if q.Unpack(request) != nil || len(q.Questions) != 1 {
		return
	}
	fixture := FixtureCase(q.Questions[0].Name.String())
	if fixture == "" {
		return
	}
	e := Event{Started: started, Completed: time.Now().UTC(), Role: t.Role, Source: "native", Case: fixture, Type: uint16(q.Questions[0].Type), Protocol: proto, RCode: -1, TTLs: []uint32{}, Classes: []string{}}
	if t.Role == "authority" {
		host, _, _ := net.SplitHostPort(peer.String())
		e.Source = "other_fixture"
		if strings.HasSuffix(host, ".20") {
			e.Source = "primary"
		}
		if strings.HasSuffix(host, ".22") {
			e.Source = "secondary"
		}
	}
	if err == nil && a.Unpack(response) == nil {
		e.Success = true
		e.RCode = int(a.RCode)
		for _, v := range a.Answers {
			e.TTLs = append(e.TTLs, v.Header.TTL)
			var ip netip.Addr
			switch b := v.Body.(type) {
			case *dnsmessage.AResource:
				ip = netip.AddrFrom4(b.A)
			case *dnsmessage.AAAAResource:
				ip = netip.AddrFrom16(b.AAAA)
			case *dnsmessage.SOAResource:
				e.AuthoritySerial = b.Serial
			}
			if ip.IsValid() {
				c := "other"
				if ip.IsPrivate() {
					c = "private"
				}
				if ip.Is4() {
					c += "_ipv4"
				} else {
					c += "_ipv6"
				}
				e.Classes = append(e.Classes, c)
			}
		}
		for _, v := range a.Authorities {
			if b, ok := v.Body.(*dnsmessage.SOAResource); ok {
				e.AuthoritySerial = b.Serial
			}
		}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	f, openErr := os.OpenFile(t.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if openErr != nil {
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(e)
}
