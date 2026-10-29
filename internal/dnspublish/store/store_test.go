package store

import (
	"context"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
)

func testUpstreams() Config {
	// TEST ONLY documentation prefix. Not a resolver this process dials.
	return Config{Upstreams: []string{"192.0.2.53:53"}}
}

func Test12_5_EmptyUpstreamsRejected(t *testing.T) {
	if _, err := NewPublisher(nil, Config{}); err == nil {
		t.Fatal("empty upstream list was accepted")
	}
}

func Test12_5_IgnoreOtherPolicyTypes(t *testing.T) {
	h, err := NewHandler(nil, testUpstreams())
	if err != nil {
		t.Fatal(err)
	}
	got, err := h(context.Background(), db.ClaimedEvent{
		OutboxEvent: db.OutboxEvent{EventType: dnspublish.EventDNSPublish, Payload: []byte(`{"type":"plan"}`)},
	})
	if err != nil || got != "dns-publish:ignored:plan" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestPublisherRejectsUnconfigured(t *testing.T) {
	h, err := NewHandler(nil, testUpstreams())
	if err != nil {
		t.Fatal(err)
	}
	_, err = h(context.Background(), db.ClaimedEvent{
		OutboxEvent: db.OutboxEvent{EventType: dnspublish.EventDNSPublish, Payload: []byte(`{"type":"dns_profile"}`)},
	})
	if err == nil {
		t.Fatal("expected an unconfigured publisher to fail closed")
	}
	_, err = h(context.Background(), db.ClaimedEvent{
		OutboxEvent: db.OutboxEvent{EventType: "node.heartbeat", Payload: []byte(`{}`)},
	})
	if err == nil {
		t.Fatal("expected an unexpected event to be rejected")
	}
}
