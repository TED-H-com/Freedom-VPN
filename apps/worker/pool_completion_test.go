package worker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// peakCompleted returns nil only when a run on a pool of exactly its stated
// peak returned a receipt and no error before its deadline. ctxErr is the run
// context's Err(), read immediately after the run returned.
func peakCompleted(receipt string, err, ctxErr error) error {
	if err != nil {
		return err
	}
	if receipt == "" {
		return errors.New("empty receipt")
	}
	if ctxErr != nil {
		return errors.New("completed after its deadline")
	}
	return nil
}

func TestAPI01_PeakCompletedRejectsLateSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	<-ctx.Done()
	if err := peakCompleted("receipt", nil, ctx.Err()); err == nil || !strings.Contains(err.Error(), "after its deadline") {
		t.Fatalf("late success: %v", err)
	}
	for _, tc := range []struct {
		name, receipt string
		err           error
		want          string
	}{
		{name: "live success", receipt: "receipt"},
		{name: "error", err: errors.New("x"), want: "x"},
		{name: "empty receipt", want: "empty receipt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := peakCompleted(tc.receipt, tc.err, nil)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}
