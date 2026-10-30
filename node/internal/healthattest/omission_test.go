package healthattest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestD063_ProbeOmissionRoundTripAndBinding(t *testing.T) {
	base, _, _, versions := validPair(t)
	in := Input{TxnID: base.TxnID, ReleaseID: base.ReleaseID, NodeID: base.NodeID,
		IncarnationID: base.IncarnationID, BootID: base.BootID, Components: versions,
		Now: base.IssuedAt, TTL: 45 * time.Second, ProbeOmitted: &ProbeOmission{PeerID: "test-peer", Reason: "node_addresses_unavailable"}}
	ch, err := Issue(in)
	if err != nil {
		t.Fatal(err)
	}
	in.ProbeOmitted.PeerID = "changed"
	if ch.ProbeOmitted.PeerID != "test-peer" {
		t.Fatal("omission aliases caller")
	}
	dir := privateDir(t)
	if err := WriteChallenge(dir, ch); err != nil {
		t.Fatal(err)
	}
	read, _, err := ReadChallenge(dir)
	if err != nil || !ch.Same(read) {
		t.Fatal("consumer's shared reader refused omission", err)
	}
	for _, omit := range []*ProbeOmission{nil, {PeerID: "other", Reason: ch.ProbeOmitted.Reason}, {PeerID: "test-peer", Reason: "node_source"}} {
		changed := ch
		changed.ProbeOmitted = omit
		if changed.Same(ch) {
			t.Fatal("omission substitution kept the same binding")
		}
	}
	if err := MarkConsumed(dir, ch); err != nil {
		t.Fatal(err)
	}
	read, _, err = ReadChallenge(dir)
	if err != nil || !read.Consumed || *read.ProbeOmitted != *ch.ProbeOmitted {
		t.Fatal("consumption lost omission", err)
	}
	// Confirm, rather than assume, the compatibility boundary: the consumer
	// uses strict decoding. Newly declared fields work, unknown ones do not.
	b, _ := json.Marshal(ch)
	b = append(b[:len(b)-1], []byte(`,"unknown_future_member":true}`)...)
	if err := os.WriteFile(filepath.Join(dir, ChallengeFile), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadChallenge(dir); err == nil {
		t.Fatal("unknown challenge fields silently accepted")
	}
	for _, omission := range []*ProbeOmission{{PeerID: "test-peer", Reason: "raw private diagnostic"}, {PeerID: strings.Repeat("x", 129), Reason: "malformed"}, {PeerID: "bad\npeer", Reason: "malformed"}} {
		in.ProbeOmitted = omission
		if _, err := Issue(in); err == nil {
			t.Fatal("unbounded or unknown omission accepted")
		}
	}
	in.ProbeOmitted, in.Probe = ch.ProbeOmitted, &ChallengeProbe{}
	if _, err := Issue(in); err == nil {
		t.Fatal("probe and omission coexist")
	}
}
