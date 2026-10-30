package main

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
)

// TEST ONLY: deterministic, lab-local source. Mixed case and a duplicate make
// normalization observable without recording member DNS queries.
func writeFeed(state string) error {
	var b strings.Builder
	b.WriteString("# TEST ONLY deterministic acceptance feed\n")
	for i := 1; i <= 10; i++ {
		fmt.Fprintf(&b, "0.0.0.0 Feed-%02d.Example.Invalid\n", i)
	}
	b.WriteString("0.0.0.0 feed-01.example.invalid\n")
	raw := []byte(b.String())
	if err := writeSecret(filepath.Join(state, "releases", "feeds", "acceptance.txt"), raw); err != nil {
		return err
	}
	return writeJSON(filepath.Join(state, "feed.json"), map[string]any{
		"content_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "rule_count": 10,
		"source_url": "https://releases.example.invalid/feeds/acceptance.txt", "size_bytes": len(raw),
	})
}
