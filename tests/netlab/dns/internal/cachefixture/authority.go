// Package cachefixture serves only TEST ONLY positive-cache conformance cases.
// It never recurses or records names, addresses, credentials or DNS payloads.
package cachefixture

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

type Event struct {
	Started   time.Time `json:"started"`
	Completed time.Time `json:"completed"`
	Role      string    `json:"source_role"`
	Case      string    `json:"fixture_case"`
	Protocol  string    `json:"protocol"`
	Type      uint16    `json:"rr_type"`
	Class     string    `json:"answer_class"`
	TTLs      []uint32  `json:"answer_ttl_remaining"`
	Serial    uint32    `json:"snapshot_serial"`
	Outcome   string    `json:"outcome"`
}

type Phase struct {
	Case    string `json:"fixture_case"`
	Changed bool   `json:"changed"`
}

type Authority struct {
	mu      sync.Mutex
	changed map[string]bool
	counts  map[string]int
	events  []Event
}

func validCase(s string) bool {
	switch s {
	case "steady", "boundary", "slow", "retry", "late", "failed", "fallback", "inflight", "smoke":
		return true
	}
	return false
}

func (a *Authority) control(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet && r.URL.Path == "/events" {
		_ = json.NewEncoder(w).Encode(a.events)
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/phase" {
		var p Phase
		if json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&p) != nil || !validCase(p.Case) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		a.changed[p.Case] = p.Changed
		_ = json.NewEncoder(w).Encode(p)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (a *Authority) answer(raw []byte, remote, proto string) []byte {
	var q dnsmessage.Message
	if q.Unpack(raw) != nil || len(q.Questions) != 1 {
		return nil
	}
	question := q.Questions[0]
	name := question.Name.String()
	label := strings.TrimSuffix(name, ".example.com.")
	if name == "example.com." {
		label = "smoke"
	}
	if !validCase(label) {
		return nil
	}
	role := "sampler"
	host, _, _ := net.SplitHostPort(remote)
	switch host {
	case "192.0.2.3":
		role = "primary"
	case "192.0.2.4":
		role = "secondary"
	}
	a.mu.Lock()
	serial, class, ip := uint32(1001), "other_ipv4", [4]byte{198, 51, 100, 10}
	if a.changed[label] {
		serial, class, ip = 1002, "changed_ipv4", [4]byte{203, 0, 113, 20}
	}
	key := label + "/" + role
	first := a.counts[key] == 0
	if question.Type == dnsmessage.TypeA {
		a.counts[key]++
	}
	ev := Event{Started: time.Now().UTC(), Role: role, Case: label, Protocol: proto, Type: uint16(question.Type), TTLs: []uint32{}, Serial: serial, Outcome: "pending"}
	index := len(a.events)
	a.events = append(a.events, ev)
	a.mu.Unlock()
	outcome := "answered"
	defer func() {
		a.mu.Lock()
		a.events[index].Completed, a.events[index].Outcome = time.Now().UTC(), outcome
		if outcome == "answered" && question.Type == dnsmessage.TypeA {
			a.events[index].Class, a.events[index].TTLs = class, []uint32{10}
		}
		a.mu.Unlock()
	}()
	if question.Type == dnsmessage.TypeA && role == "primary" {
		if label == "failed" || (first && (label == "retry" || label == "fallback")) {
			outcome = "dropped"
			return nil
		}
		if first && label == "slow" {
			time.Sleep(2500 * time.Millisecond)
		}
		if first && label == "late" {
			time.Sleep(11 * time.Second)
		}
		if first && label == "inflight" {
			time.Sleep(3500 * time.Millisecond)
		}
	}
	response := dnsmessage.Message{Header: dnsmessage.Header{ID: q.ID, Response: true, Authoritative: true, RecursionDesired: q.RecursionDesired}, Questions: q.Questions}
	h := dnsmessage.ResourceHeader{Name: question.Name, Class: dnsmessage.ClassINET, TTL: 10, Type: question.Type}
	switch question.Type {
	case dnsmessage.TypeA:
		response.Answers = []dnsmessage.Resource{{Header: h, Body: &dnsmessage.AResource{A: ip}}}
	case dnsmessage.TypeSOA:
		response.Answers = []dnsmessage.Resource{{Header: h, Body: &dnsmessage.SOAResource{NS: dnsmessage.MustNewName("ns.example.com."), MBox: dnsmessage.MustNewName("hostmaster.example.com."), Serial: serial, Refresh: 60, Retry: 60, Expire: 600, MinTTL: 10}}}
	}
	out, _ := response.Pack()
	return out
}

// Run serves UDP/TCP DNS and metadata control only inside the isolated bridge.
func Run() error {
	a := &Authority{changed: map[string]bool{}, counts: map[string]int{}, events: []Event{}}
	udp, err := net.ListenPacket("udp", ":5353")
	if err != nil {
		return err
	}
	defer udp.Close()
	tcp, err := net.Listen("tcp", ":5353")
	if err != nil {
		return err
	}
	defer tcp.Close()
	go func() {
		for {
			buf := make([]byte, 4096)
			n, remote, err := udp.ReadFrom(buf)
			if err != nil {
				return
			}
			go func() {
				if out := a.answer(buf[:n], remote.String(), "udp"); out != nil {
					_, _ = udp.WriteTo(out, remote)
				}
			}()
		}
	}()
	go func() {
		for {
			conn, err := tcp.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(25 * time.Second))
				var size [2]byte
				if _, err := io.ReadFull(conn, size[:]); err != nil {
					return
				}
				buf := make([]byte, int(binary.BigEndian.Uint16(size[:])))
				if _, err := io.ReadFull(conn, buf); err != nil {
					return
				}
				if out := a.answer(buf, conn.RemoteAddr().String(), "tcp"); out != nil {
					_, _ = conn.Write(append(binary.BigEndian.AppendUint16(nil, uint16(len(out))), out...))
				}
			}()
		}
	}()
	server := &http.Server{Addr: ":8080", Handler: http.HandlerFunc(a.control), ReadHeaderTimeout: 5 * time.Second}
	return server.ListenAndServe()
}
