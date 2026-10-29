package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRun_ValidationErrors(t *testing.T) {
	var buf bytes.Buffer
	if err := Run(context.Background(), nil, nil, &buf); err == nil {
		t.Fatalf("expected error for missing subcommand")
	}
	if err := Run(context.Background(), nil, []string{"unknown"}, &buf); err == nil || !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("expected unknown subcommand error, got %v", err)
	}
}
