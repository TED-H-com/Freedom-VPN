//go:build linux

package main

import (
	"encoding/binary"
	"testing"
)

func TestAT02_SiblingFilterRejectsUnrelatedTraffic(t *testing.T) {
	b := make([]byte, 28)
	b[0] = 0x45
	b[9] = 17
	binary.BigEndian.PutUint16(b[22:24], 40002)
	b = append(b, siblingMagic...)
	b = binary.BigEndian.AppendUint64(b, 9007199254740993)
	if seq, ok := siblingSequence(b); !ok || seq != 9007199254740993 {
		t.Fatal("lost exact sequence")
	}
	for i := 0; i < len(b); i++ {
		if _, ok := siblingSequence(b[:i]); ok {
			t.Fatal("truncated packet accepted")
		}
	}
	b[22] = 0
	if _, ok := siblingSequence(b); ok {
		t.Fatal("unrelated port accepted")
	}
}
