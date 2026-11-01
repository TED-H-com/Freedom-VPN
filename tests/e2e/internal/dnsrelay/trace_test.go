package dnsrelay

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func TestAT20_TraceRetainsMetadataWithoutQueryOrAddress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	trace := Trace{Role: "primary", Path: path}
	q := dnsmessage.Message{Questions: []dnsmessage.Question{{Name: dnsmessage.MustNewName("cache.example.invalid."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}}}
	request, _ := q.Pack()
	a := dnsmessage.Message{Header: dnsmessage.Header{Response: true}, Answers: []dnsmessage.Resource{{Header: dnsmessage.ResourceHeader{Name: q.Questions[0].Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 10}, Body: &dnsmessage.AResource{A: [4]byte{10, 203, 0, 30}}}}}
	response, _ := a.Pack()
	event := trace.Begin(request, &net.UDPAddr{}, "udp", time.Now().UTC())
	trace.Complete(event, response, nil)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	history, complete := History(raw)
	if !complete || len(history) != 1 || history[0].Completion == nil {
		t.Fatalf("incomplete lifecycle: %s", raw)
	}
	e := *history[0].Completion
	if e.QuerySHA256 != fmt.Sprintf("%x", sha256.Sum256(request)) || e.ReplySHA256 != fmt.Sprintf("%x", sha256.Sum256(response)) || e.ReplyBytes != len(response) {
		t.Fatal("wire digest boundary lost")
	}
	if e.Case != "cache" || e.Role != "primary" || !e.Success || len(e.TTLs) != 1 || e.TTLs[0] != 10 || len(e.Classes) != 1 || e.Classes[0] != "private_ipv4" {
		t.Fatalf("lost trace: %s", raw)
	}
	if strings.Contains(string(raw), "example.invalid") || strings.Contains(string(raw), "10.203.0.30") {
		t.Fatal("query/address leak")
	}
	q.Questions[0].Name = dnsmessage.MustNewName("unlisted.example.invalid.")
	request, _ = q.Pack()
	trace.Complete(trace.Begin(request, &net.UDPAddr{}, "udp", time.Now().UTC()), response, nil)
	after, _ := os.ReadFile(path)
	if string(raw) != string(after) {
		t.Fatal("arbitrary names retained")
	}
}

func TestAT20_OutstandingLifecycleAndMissingReceipts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	trace := Trace{Role: "authority", Path: path}
	q := dnsmessage.Message{Questions: []dnsmessage.Question{{Name: dnsmessage.MustNewName("cache.example.invalid."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}}}
	wire, _ := q.Pack()
	older := trace.Begin(wire, &net.UDPAddr{}, "udp", time.Now().UTC())
	raw, _ := os.ReadFile(path)
	pending, ok := History(raw)
	if !ok || len(pending) != 1 || pending[0].Completion != nil {
		t.Fatal("start not visible before exchange")
	}
	newer := trace.Begin(wire, &net.UDPAddr{}, "udp", time.Now().UTC())
	trace.Complete(newer, nil, os.ErrDeadlineExceeded)
	trace.Complete(older, nil, os.ErrDeadlineExceeded)
	raw, _ = os.ReadFile(path)
	history, ok := History(raw)
	if !ok || len(history) != 2 || history[0].Completion.RequestID != older.RequestID || history[1].Completion.RequestID != newer.RequestID {
		t.Fatal("out of order completion not joined")
	}
	lines := strings.SplitAfter(string(raw), "\n")
	for _, damaged := range []string{strings.Join(lines[1:], ""), lines[0] + strings.Join(lines[2:], ""), string(raw) + "{", string(raw) + lines[3]} {
		if _, ok := History([]byte(damaged)); ok {
			t.Fatal("missing/partial/duplicate lifecycle reported complete")
		}
	}
}

func TestAT20_RestartRetainsStreamsAndPendingRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	q := dnsmessage.Message{Questions: []dnsmessage.Question{{Name: dnsmessage.MustNewName("cache.example.invalid."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}}}
	wire, _ := q.Pack()
	first := Trace{Role: "authority", Path: path}
	pending := first.Begin(wire, &net.UDPAddr{}, "udp", time.Now().UTC())
	second := Trace{Role: "authority", Path: path}
	second.Complete(second.Begin(wire, &net.UDPAddr{}, "udp", time.Now().UTC()), nil, os.ErrDeadlineExceeded)
	raw, _ := os.ReadFile(path)
	history, ok := History(raw)
	if !ok || len(history) != 2 || history[0].Start.RequestID != pending.RequestID || history[0].Completion != nil || history[1].Completion == nil || history[0].Start.StreamID == history[1].Start.StreamID {
		t.Fatal("restart erased an unresolved request or stream boundary")
	}
}

func TestAT20_IDNAPositiveCaseMapping(t *testing.T) {
	if FixtureCase("XN--BCHER-KVA.example.invalid.") != "idna_positive" || FixtureCase("bücher.example.invalid") != "" || FixtureCase("xn--other.example.invalid") != "" {
		t.Fatal("IDNA wire case must be closed and A-label only")
	}
}
