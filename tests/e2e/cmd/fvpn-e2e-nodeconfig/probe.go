package main

// TEST ONLY observer and traffic generator. Neither writes nftables state,
// challenges or attestations. All output is aggregate, without peer identities.
import (
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth"
	"github.com/google/uuid"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

func probeClient() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("probe id required")
	}
	id, err := uuid.Parse(os.Args[2])
	if err != nil {
		return err
	}
	raw, err := os.ReadFile("/var/lib/vpn-node/e2e/probe.secret")
	if err != nil {
		return err
	}
	key := []byte(strings.TrimSpace(string(raw)))
	addr, err := net.ResolveUDPAddr("udp4", "releases.example.invalid:4443")
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	packet := probeauth.Seal(key, id, false)
	attempts, success := 0, 0
	var response [256]byte
	until := time.Now().Add(180 * time.Second)
	for time.Now().Before(until) {
		if _, err := os.Stat("/var/lib/vpn-node/e2e/probe-stop"); err == nil {
			break
		}
		_ = conn.SetDeadline(time.Now().Add(5 * time.Millisecond))
		attempts++
		if _, err = conn.Write(packet); err == nil {
			n, e := conn.Read(response[:])
			if e == nil {
				got, ok := probeauth.Open(key, response[:n], true)
				if ok && got == id {
					success++
				}
			}
		}
		time.Sleep(time.Millisecond)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{"attempts": attempts, "authenticated_udp_replies": success})
}

func probeWatch() error {
	type evidence struct {
		Samples             int     `json:"samples"`
		GrantSamples        int     `json:"grant_samples"`
		SourceElements      int     `json:"source_elements"`
		DestinationElements int     `json:"destination_elements"`
		MaxTimeoutSeconds   float64 `json:"max_timeout_seconds"`
		Up                  uint64  `json:"probe_up_bytes"`
		Down                uint64  `json:"probe_down_bytes"`
		MemberBefore        uint64  `json:"member_bytes_before"`
		MemberAfter         uint64  `json:"member_bytes_after"`
		ElementsAfter       int     `json:"elements_after"`
	}
	out := evidence{}
	until := time.Now().Add(180 * time.Second)
	for time.Now().Before(until) {
		raw, err := exec.Command("nft", "-j", "list", "table", "inet", "fvpn").Output()
		if err == nil {
			st, err := enforce.ParseState(raw)
			if err == nil {
				out.Samples++
				var member uint64
				for name, c := range st.Counters {
					if strings.HasPrefix(name, "u_") || strings.HasPrefix(name, "d_") {
						member += c.Bytes
					}
				}
				if out.Samples == 1 {
					out.MemberBefore = member
				}
				out.MemberAfter = member
				out.Up = st.Counters["probe_up"].Bytes
				out.Down = st.Counters["probe_down"].Bytes
				var doc struct {
					Items []struct {
						Set struct {
							Name string            `json:"name"`
							Elem []json.RawMessage `json:"elem"`
						} `json:"set"`
					} `json:"nftables"`
				}
				_ = json.Unmarshal(raw, &doc)
				sources, dests := 0, 0
				for _, item := range doc.Items {
					if item.Set.Name != "probe_v4" && item.Set.Name != "probe_dst_v4" {
						continue
					}
					if item.Set.Name == "probe_v4" {
						sources = len(item.Set.Elem)
					} else {
						dests = len(item.Set.Elem)
					}
					for _, elem := range item.Set.Elem {
						var e struct {
							Elem struct {
								Timeout float64 `json:"timeout"`
							} `json:"elem"`
						}
						_ = json.Unmarshal(elem, &e)
						if e.Elem.Timeout > out.MaxTimeoutSeconds {
							out.MaxTimeoutSeconds = e.Elem.Timeout
						}
					}
				}
				out.ElementsAfter = sources + dests
				if sources > 0 && dests > 0 {
					out.GrantSamples++
					out.SourceElements = sources
					out.DestinationElements = dests
				}
			}
		}
		if _, err := os.Stat("/var/lib/vpn-node/e2e/probe-stop"); err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
