package isolation

import "testing"

func TestAT10_AT13_NegativesNeedCountedControl(t *testing.T) {
	p := Positive{DeviceID: "TEST ONLY", Counted: true, Before: "9007199254740993", After: "9007199254740994"}
	if NegativeStatus(p, true, true) != StatusPass || NegativeStatus(p, true, false) != StatusFail {
		t.Fatal("counted outcomes lost")
	}
	for _, bad := range []Positive{{}, {Counted: true}, {DeviceID: "TEST ONLY", Counted: true, Before: "9", After: "9"}} {
		if NegativeStatus(bad, true, true) != StatusBlocked || ChecksStatus([]Check{{Status: StatusPass, Positive: bad}}) != StatusBlocked {
			t.Fatal("unproved negative passed")
		}
	}
	if NegativeStatus(p, false, true) != StatusBlocked || ChecksStatus(nil) != StatusBlocked {
		t.Fatal("unexecuted case passed")
	}
}
