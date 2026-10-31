package counterread

import (
	"context"
	"errors"
	"testing"
)

func TestAT02_ReadFailureCategories(t *testing.T) {
	for _, v := range []struct{ stderr, want string }{{"Operation not permitted", "permission"}, {"Permission denied", "permission"}, {"Resource temporarily unavailable", "transient_io"}, {"Input/output error", "transient_io"}, {"No such file or directory", "nft_read"}} {
		if got := Category(t.Context(), errors.New("exit 1"), v.stderr); got != v.want {
			t.Fatalf("%s: %s", v.stderr, got)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if Category(ctx, errors.New("killed"), "") != "timeout" {
		t.Fatal("deadline lost")
	}
	if Category(t.Context(), nil, "") != "none" {
		t.Fatal("successful read misclassified")
	}
}
