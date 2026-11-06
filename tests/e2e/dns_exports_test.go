//go:build e2e

package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

func TestAT20_DNSFailureRetainsAfterCounters(t *testing.T) {
	device := uuid.New()
	before := dnsTestCounters(device)
	after := dnsTestCounters(device)
	for name, value := range after {
		if strings.HasPrefix(name, "nu_") || strings.HasPrefix(name, "nd_") {
			value.Bytes += 80
			value.Packets++
			after[name] = value
		}
	}
	for _, afterFails := range []bool{false, true} {
		t.Run(fmt.Sprint(afterFails), func(t *testing.T) {
			var calls []string
			read := func() ([]byte, error) {
				if len(calls) == 0 {
					calls = append(calls, "before")
					return dnsTestSnapshot(t, before), nil
				}
				calls = append(calls, "after")
				if afterFails {
					return []byte("TEST-SECRET-STDERR"), errors.New("TEST-SECRET-ERROR")
				}
				return dnsTestSnapshot(t, after), nil
			}
			query := func() ([]byte, error) {
				calls = append(calls, "query")
				return []byte("fvpn-e2e-member-session: dns read class=timeout\n"), errors.New("TEST-SECRET-QUERY")
			}
			got, err := observeDNSQuery(newDNSPathObservation(false), device, read, query)
			if !reflect.DeepEqual(calls, []string{"before", "query", "after"}) {
				t.Fatalf("command order = %v", calls)
			}
			if err == nil || err.Error() != "designated DNS query: read class=timeout" || got.QueryStatus != "failed" || got.QueryStage != "read" || got.QueryClass != "timeout" {
				t.Fatalf("query failure lost: %+v, %v", got, err)
			}
			if afterFails {
				if got.AfterCounters != "read_error" || got.CounterDeltas != "unavailable" || got.DNSUpload != 0 || got.DNSDownload != 0 {
					t.Fatalf("unread counters presented as available: %+v", got)
				}
			} else if got.CounterDeltas != "available" || got.DNSUpload != 80 || got.DNSDownload != 80 || got.InternetUpload != 0 || got.InternetDownload != 0 {
				t.Fatalf("measured deltas lost: %+v", got)
			}
			assertDNSSafe(t, got, err)
		})
	}
}

func TestAT20_DNSCounterAvailability(t *testing.T) {
	device := uuid.New()
	for _, prefix := range []string{"nu_", "nd_", "u_", "d_"} {
		for _, fault := range []string{"before_missing", "after_missing", "bytes_reset", "packets_reset"} {
			t.Run(prefix+fault, func(t *testing.T) {
				before, after := dnsTestCounters(device), dnsTestCounters(device)
				name := prefix + strings.ReplaceAll(device.String(), "-", "")
				beforeStatus, afterStatus, deltaStatus := "available", "available", "unavailable"
				switch fault {
				case "before_missing":
					delete(before, name)
					beforeStatus = "missing_counter"
				case "after_missing":
					delete(after, name)
					afterStatus = "missing_counter"
				case "bytes_reset", "packets_reset":
					v := after[name]
					if fault == "bytes_reset" {
						v.Bytes--
					} else {
						v.Packets--
					}
					after[name] = v
					deltaStatus = "counter_reset"
				}
				calls := 0
				read := func() ([]byte, error) {
					calls++
					if calls == 1 {
						return dnsTestSnapshot(t, before), nil
					}
					return dnsTestSnapshot(t, after), nil
				}
				got, err := observeDNSQuery(newDNSPathObservation(false), device, read, func() ([]byte, error) { return []byte("rcode=0 answers=1\n"), nil })
				if err == nil || got.BeforeCounters != beforeStatus || got.AfterCounters != afterStatus || got.CounterDeltas != deltaStatus || got.QueryStatus != "succeeded" || calls != 2 {
					t.Fatalf("invalid counter evidence accepted: %+v, %v; reads=%d", got, err, calls)
				}
				if got.DNSUpload != 0 || got.DNSDownload != 0 || got.InternetUpload != 0 || got.InternetDownload != 0 {
					t.Fatalf("partial/wrapped deltas published: %+v", got)
				}
			})
		}
	}
}

func TestAT20_DNSCounterReadFailures(t *testing.T) {
	device := uuid.New()
	for _, at := range []int{1, 2} {
		for _, fault := range []string{"read_error", "parse_error"} {
			t.Run(fmt.Sprintf("%d/%s", at, fault), func(t *testing.T) {
				reads, queries := 0, 0
				read := func() ([]byte, error) {
					reads++
					if reads == at {
						if fault == "read_error" {
							return nil, errors.New("TEST-SECRET-READ")
						}
						return []byte("TEST-SECRET-MALFORMED-COUNTERS"), nil
					}
					return dnsTestSnapshot(t, dnsTestCounters(device)), nil
				}
				got, err := observeDNSQuery(newDNSPathObservation(false), device, read, func() ([]byte, error) {
					queries++
					return []byte("rcode=0 answers=1\n"), nil
				})
				if err == nil || got.CounterDeltas != "unavailable" || reads != at || queries != at-1 {
					t.Fatalf("unexpected sequence/evidence: %+v, %v, reads=%d queries=%d", got, err, reads, queries)
				}
				if at == 1 && (got.BeforeCounters != fault || got.AfterCounters != "not_attempted" || got.QueryStatus != "not_attempted") || at == 2 && (got.BeforeCounters != "available" || got.AfterCounters != fault || got.QueryStatus != "succeeded") {
					t.Fatalf("wrong availability: %+v", got)
				}
				assertDNSSafe(t, got, err)
			})
		}
	}
}

func TestAT20_DNSSuccessAndSafeOutput(t *testing.T) {
	device := uuid.New()
	for _, tcp := range []bool{false, true} {
		reads := 0
		got, err := observeDNSQuery(newDNSPathObservation(tcp), device, func() ([]byte, error) {
			reads++
			counters := dnsTestCounters(device)
			if reads == 2 && tcp {
				for name, value := range counters {
					if strings.HasPrefix(name, "nu_") || strings.HasPrefix(name, "nd_") {
						value.Bytes += 80
						value.Packets++
						counters[name] = value
					}
				}
			}
			return dnsTestSnapshot(t, counters), nil
		}, func() ([]byte, error) { return []byte("rcode=0 answers=1\n"), nil })
		wantDNS := uint64(0)
		if tcp {
			wantDNS = 80
		}
		if err != nil || got.Reply != "rcode=0 answers=1" || got.QueryStatus != "succeeded" || got.CounterDeltas != "available" || got.DNSUpload != wantDNS || got.DNSDownload != wantDNS || got.InternetUpload != 0 || got.InternetDownload != 0 {
			t.Fatalf("valid measured deltas lost: %+v, %v", got, err)
		}
		if tcp && got.Transport != "tcp" || !tcp && got.Transport != "udp" {
			t.Fatalf("wrong transport: %+v", got)
		}
	}
	for _, output := range []string{
		"TEST-SECRET-RAW-ERROR",
		"rcode=0 answers=1\nTEST-SECRET-RAW-ERROR",
		"fvpn-e2e-member-session: dns read class=timeout\nTEST-SECRET-RAW-ERROR",
		"fvpn-e2e-member-session: dns TEST-SECRET-STAGE class=timeout",
		"fvpn-e2e-member-session: dns read class=TEST-SECRET-CLASS",
		strings.Repeat("TEST-SECRET", 30),
	} {
		for _, queryErr := range []error{nil, errors.New("TEST-SECRET-QUERY")} {
			got, err := observeDNSQuery(newDNSPathObservation(false), device, func() ([]byte, error) {
				return dnsTestSnapshot(t, dnsTestCounters(device)), nil
			}, func() ([]byte, error) { return []byte(output), queryErr })
			if err == nil || got.QueryStatus != "failed" || got.QueryStage != "command" || got.QueryClass != "unrecognized_output" || got.Reply != "" || got.AfterCounters != "available" {
				t.Fatalf("unrecognized output accepted: %+v, %v", got, err)
			}
			assertDNSSafe(t, got, err)
		}
	}
}

func dnsTestCounters(device uuid.UUID) map[string]enforce.CounterValue {
	values := make(map[string]enforce.CounterValue)
	for _, prefix := range []string{"nu_", "nd_", "u_", "d_"} {
		values[prefix+strings.ReplaceAll(device.String(), "-", "")] = enforce.CounterValue{Bytes: 100, Packets: 2}
	}
	return values
}

func dnsTestSnapshot(t *testing.T, counters map[string]enforce.CounterValue) []byte {
	t.Helper()
	entries := make([]any, 0, len(counters))
	for name, v := range counters {
		entries = append(entries, map[string]any{"counter": map[string]any{"family": "inet", "table": "fvpn", "name": name, "bytes": v.Bytes, "packets": v.Packets}})
	}
	raw, err := json.Marshal(map[string]any{"nftables": entries})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func assertDNSSafe(t *testing.T, observation DNSPathObservation, err error) {
	t.Helper()
	raw, marshalErr := json.Marshal(observation)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if strings.Contains(string(raw), "TEST-SECRET") || err != nil && strings.Contains(err.Error(), "TEST-SECRET") {
		t.Fatal("raw failure content escaped")
	}
}
