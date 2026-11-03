package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestF27_GuardLogKeepsPredicatesAndDrops(t *testing.T) {
	original := "add chain inet fvpn fv_deadline_guard { type filter hook forward priority -151; policy accept; }\nflush chain inet fvpn fv_deadline_guard\nadd rule inet fvpn fv_deadline_guard iifname \"fv-fh\" drop\nadd rule inet fvpn fv_deadline_guard oifname \"fv-fh\" drop\n"
	got, err := guardLogScript(original, 1001)
	if err != nil {
		t.Fatal(err)
	}
	log := regexp.MustCompile(` log prefix "f29g1001-(iifname|oifname)" group 29 snaplen 128 queue-threshold 1`)
	if len(log.FindAllString(got, -1)) != 2 || log.ReplaceAllString(got, "") != original {
		t.Fatal("observer changed predicates, ordering or drop verdicts")
	}
	for _, bad := range []string{strings.ReplaceAll(original, " drop", " accept"), original + "add rule inet fvpn fv_deadline_guard iifname \"fv-fh\" drop\n", "delete chain inet fvpn fv_deadline_guard\n"} {
		if _, err := guardLogScript(bad, 1001); err == nil {
			t.Fatal("unexpected guard shape was accepted")
		}
	}
}
