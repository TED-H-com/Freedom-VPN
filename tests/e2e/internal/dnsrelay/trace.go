package dnsrelay

// TEST ONLY metadata for a closed set of synthetic fixture cases. No full
// query names, addresses, packet bodies or resolver history are retained.
import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/net/dns/dnsmessage"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
)

type Event struct {
	QuerySHA256     string    `json:"query_sha256,omitempty"`
	ReplySHA256     string    `json:"reply_sha256,omitempty"`
	WireID          uint16    `json:"wire_id"`
	ReplyBytes      int       `json:"reply_bytes,omitempty"`
	RequestID       string    `json:"request_id"`
	StreamID        string    `json:"stream_id"`
	Sequence        uint64    `json:"sequence,string"`
	Phase           string    `json:"phase"`
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
	stream     string
	sequence   uint64
}

func FixtureCase(name string) string {
	if strings.EqualFold(strings.TrimSuffix(name, "."), "xn--bcher-kva.example.invalid") {
		return "idna_positive"
	}
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

// Begin writes before dispatch, so an abandoned client cannot hide a fetch.
// The returned metadata carries a random identity, a wire ID and opaque digests for joins, never a qname.
func (t *Trace) Begin(request []byte, peer net.Addr, proto string, started time.Time) Event {
	if t == nil {
		return Event{}
	}
	var q dnsmessage.Message
	if q.Unpack(request) != nil || len(q.Questions) != 1 {
		return Event{}
	}
	fixture := FixtureCase(q.Questions[0].Name.String())
	if fixture == "" {
		return Event{}
	}
	e := Event{WireID: q.ID, QuerySHA256: fmt.Sprintf("%x", sha256.Sum256(request)), RequestID: uuid.NewString(), Phase: "started", Started: started, Role: t.Role, Source: "native", Case: fixture, Type: uint16(q.Questions[0].Type), Protocol: proto, RCode: -1, TTLs: []uint32{}, Classes: []string{}}
	if t.Role == "authority" {
		host, _, _ := net.SplitHostPort(peer.String())
		e.Source = "other_fixture"
		if strings.HasSuffix(host, ".40") {
			e.Source = "primary"
		}
		if strings.HasSuffix(host, ".22") {
			e.Source = "secondary"
		}
	}
	return t.append(e)
}

// Complete joins the reply (or transport failure) to its original start.
func (t *Trace) Complete(e Event, response []byte, err error) {
	if t == nil || e.RequestID == "" {
		return
	}
	e.Phase, e.Completed = "completed", time.Now().UTC()
	var a dnsmessage.Message
	if err == nil && a.Unpack(response) == nil {
		e.Success = true
		e.ReplySHA256 = fmt.Sprintf("%x", sha256.Sum256(response))
		e.ReplyBytes = len(response)
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
	t.append(e)
}

func (t *Trace) append(e Event) Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stream == "" {
		t.stream = uuid.NewString()
	}
	t.sequence++
	e.StreamID, e.Sequence = t.stream, t.sequence
	f, openErr := os.OpenFile(t.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if openErr != nil {
		return e
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(e)
	return e
}
