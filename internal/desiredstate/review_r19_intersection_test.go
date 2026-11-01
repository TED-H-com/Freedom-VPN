package desiredstate

import (
	"bytes"
	"os"
	"testing"
)

func TestReviewR19_Q2_BindingRevisionIsTheMaxDeviceRevision(t *testing.T) {
	src, err := os.ReadFile("compiler.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(src, []byte("if b.BindingRevision > maxBindingRev")) ||
		!bytes.Contains(src, []byte("BindingRevision:          nodes.ParseU64(maxBindingRev),")) {
		t.Fatal("compiler no longer publishes the maximum included binding revision")
	}
	// The same max, applied to a two-device set. Removing the lower revision
	// leaves the document number unchanged while the set contents change.
	// Removing the higher revision makes the document number go backwards.
	const low, high uint64 = 9, 12
	maxOf := func(vs ...uint64) uint64 {
		var m uint64
		for _, v := range vs {
			if v > m {
				m = v
			}
		}
		return m
	}
	if maxOf(low, high) != high || maxOf(high) != high || maxOf(low) != low {
		t.Fatal("max helper")
	}
	t.Fatalf("TODO(review): DesiredState.binding_revision is maxBindingRev, not a monotonic set generation. A set whose revisions are %d and %d stays at %d after the lower device is removed, and falls to %d after the higher device is removed",
		low, high, maxOf(high), maxOf(low))
}
