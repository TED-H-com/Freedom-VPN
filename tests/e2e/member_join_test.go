//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReadProjectedBindingMatchesTheDevice(t *testing.T) {
	device := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	member := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	raw := []byte(`{"signed":{"device_bindings":[{` +
		`"device_id":"11111111-1111-1111-1111-111111111111",` +
		`"member_id":"22222222-2222-2222-2222-222222222222",` +
		`"binding_revision":"3",` +
		`"valid_from":"2026-10-29T11:00:00Z",` +
		`"valid_until":"2026-10-29T13:00:00Z"` +
		`}]}}`)
	got, ok := readProjectedBinding(raw, device)
	if !ok || got.Revision.String() != "3" || got.Member != member || got.Device != device {
		t.Fatalf("binding: %+v %v", got, ok)
	}
	if !got.From.Equal(time.Date(2026, 10, 29, 11, 0, 0, 0, time.UTC)) || !got.Until.Equal(time.Date(2026, 10, 29, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("window: %s %s", got.From, got.Until)
	}
	other := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	if _, ok = readProjectedBinding(raw, other); ok {
		t.Fatal("other device")
	}
	if _, ok = readProjectedBinding(raw, uuid.Nil); ok {
		t.Fatal("nil device")
	}
}
