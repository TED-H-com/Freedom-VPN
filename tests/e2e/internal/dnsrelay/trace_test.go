package dnsrelay

import (
	"encoding/json"
	"golang.org/x/net/dns/dnsmessage"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAT20_TraceRetainsMetadataWithoutQueryOrAddress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	trace := Trace{Role: "primary", Path: path}
	q := dnsmessage.Message{Questions: []dnsmessage.Question{{Name: dnsmessage.MustNewName("cache.example.invalid."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}}}
	request, _ := q.Pack()
	a := dnsmessage.Message{Header: dnsmessage.Header{Response: true}, Answers: []dnsmessage.Resource{{Header: dnsmessage.ResourceHeader{Name: q.Questions[0].Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 10}, Body: &dnsmessage.AResource{A: [4]byte{10, 203, 0, 30}}}}}
	response, _ := a.Pack()
	trace.Record(request, response, &net.UDPAddr{}, "udp", time.Now().UTC(), nil)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var e Event
	if json.Unmarshal(raw, &e) != nil || e.Case != "cache" || e.Role != "primary" || !e.Success || len(e.TTLs) != 1 || e.TTLs[0] != 10 || len(e.Classes) != 1 || e.Classes[0] != "private_ipv4" {
		t.Fatalf("lost trace: %s", raw)
	}
	if strings.Contains(string(raw), "example.invalid") || strings.Contains(string(raw), "10.203.0.30") {
		t.Fatal("query/address leak")
	}
	q.Questions[0].Name = dnsmessage.MustNewName("unlisted.example.invalid.")
	request, _ = q.Pack()
	trace.Record(request, response, &net.UDPAddr{}, "udp", time.Now().UTC(), nil)
	after, _ := os.ReadFile(path)
	if string(raw) != string(after) {
		t.Fatal("arbitrary names retained")
	}
}
