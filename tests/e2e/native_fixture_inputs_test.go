package e2e

import "testing"

func TestAT09_NativeInputsDoNotSubstitutePlatformsOrQUIC(t *testing.T) {
	for _, platform := range []string{"linux", "android", "ios", "windows", "macos"} {
		for _, proto := range []string{"tcp", "udp", "quic"} {
			got, err := SelectNativeFixture(NativeFixtureInputs{Underlay: "dual", MTU: 1280, Protocol: proto, Platform: platform})
			if err != nil {
				t.Fatal(err)
			}
			want := platform == "linux" && proto != "quic"
			if got.Available != want || !want && got.Reason == "" {
				t.Fatalf("unavailable workload reported executable: %+v", got)
			}
		}
	}
	for _, in := range []NativeFixtureInputs{{Underlay: "public", MTU: 1280, Protocol: "tcp", Platform: "linux"}, {Underlay: "dual", MTU: 1279, Protocol: "tcp", Platform: "linux"}, {Underlay: "dual", MTU: 1501, Protocol: "tcp", Platform: "linux"}, {Underlay: "dual", MTU: 1280, Protocol: "http3-as-udp", Platform: "linux"}, {Underlay: "dual", MTU: 1280, Protocol: "tcp", Platform: "emulated-ios"}} {
		if _, err := SelectNativeFixture(in); err == nil {
			t.Fatalf("accepted invalid input: %+v", in)
		}
	}
}
