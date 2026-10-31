//go:build netlab

package publishlab_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	nodedns "github.com/TED-H-com/Freedom-VPN/node/dns"
	"github.com/TED-H-com/Freedom-VPN/tests/netlab/dns/internal/cachefixture"
	"golang.org/x/net/dns/dnsmessage"
	"gopkg.in/yaml.v3"
)

type cacheReceipt struct {
	Role      string    `json:"role"`
	Started   time.Time `json:"started"`
	Completed time.Time `json:"completed"`
	Outcome   string    `json:"outcome"`
}

type cacheTrace struct {
	Case            string               `json:"fixture_case"`
	Step            string               `json:"step"`
	Started         time.Time            `json:"started"`
	Completed       time.Time            `json:"completed"`
	Responder       string               `json:"responder"`
	Class           string               `json:"answer_class"`
	RCode           dnsmessage.RCode     `json:"rcode"`
	TTL             []uint32             `json:"answer_ttl_remaining"`
	Milliseconds    int64                `json:"milliseconds"`
	CacheState      string               `json:"cache_state"`
	ResolverEvents  []cacheReceipt       `json:"resolver_events"`
	AuthorityEvents []cachefixture.Event `json:"authority_events"`
	AuthoritySerial uint32               `json:"authority_serial_after_query"`
	AuthorityReadAt time.Time            `json:"authority_read_at"`
}

type cacheExchange struct {
	started, completed time.Time
	message            dnsmessage.Message
	err                error
}

func exchangeCache(addr, name string, qt dnsmessage.Type, timeout time.Duration) cacheExchange {
	r := cacheExchange{started: time.Now().UTC()}
	q := dnsmessage.Message{Header: dnsmessage.Header{ID: 42, RecursionDesired: true}, Questions: []dnsmessage.Question{{Name: dnsmessage.MustNewName(name + ".example.com."), Type: qt, Class: dnsmessage.ClassINET}}}
	raw, err := q.Pack()
	if err == nil {
		var conn net.Conn
		conn, err = net.DialTimeout("udp", addr, timeout)
		if err == nil {
			defer conn.Close()
			err = conn.SetDeadline(time.Now().Add(timeout))
			if err == nil {
				_, err = conn.Write(raw)
			}
			if err == nil {
				buf := make([]byte, 4096)
				var n int
				n, err = conn.Read(buf)
				if err == nil {
					err = r.message.Unpack(buf[:n])
				}
				if err == nil && (!r.message.Response || r.message.ID != q.ID || r.message.Truncated || !reflect.DeepEqual(r.message.Questions, q.Questions)) {
					err = fmt.Errorf("invalid fixture response")
				}
			}
		}
	}
	r.completed, r.err = time.Now().UTC(), err
	return r
}

// TestAT20_PositiveCacheTTL exercises the production applier, independent
// resolver caches and first-fetch delays. No cache clear or reconfiguration is
// allowed between the first measured query and the end of this experiment.
func TestAT20_PositiveCacheTTL(t *testing.T) {
	if os.Getenv("FVPN_NETLAB") != "1" {
		t.Skip("set FVPN_NETLAB=1 for pinned positive-cache conformance")
	}
	prefix := os.Getenv("FVPN_LAB_PREFIX")
	if !prefixPattern.MatchString(prefix) {
		t.Fatal("set a valid FVPN_LAB_PREFIX")
	}
	prefix += "-cache"
	state, root := t.TempDir(), moduleRoot(t)
	t.Cleanup(func() { cleanupLab(prefix, state) })
	for _, path := range []string{"docs/dns/BACKEND.md", "deploy/dev/adguard/docker-compose.yml"} {
		if !bytes.Contains(readShapeFile(t, filepath.Join(root, path)), []byte(aghImage)) {
			t.Fatalf("%s must match conformance pin", path)
		}
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(state, "authority"), "./tests/netlab/dns/cmd/cache-authority")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build authority: %v\n%s", err, out)
	}
	docker(t, "network", "create", "--internal", "--subnet", "192.0.2.0/24", prefix+"-net")
	user := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
	docker(t, "run", "-d", "--name", prefix+"-upstream", "--network", prefix+"-net", "--ip", "192.0.2.2", "--user", user, "-v", filepath.Join(state, "authority")+":/fixture/authority:ro", "--entrypoint", "/fixture/authority", aghImage)
	initial := readShapeFile(t, filepath.Join(root, "tests/netlab/dns/testdata/positive-cache/AdGuardHome.yaml"))
	var targets []nodedns.Target
	var readbacks []map[string]any
	var writes []struct {
		Role string `json:"role"`
		shapeWrite
	}
	var mu sync.Mutex
	roles := []string{"primary", "secondary"}
	for i, role := range roles {
		which := fmt.Sprintf("resolver%d", i+1)
		for _, dir := range []string{"conf", "work"} {
			if err := os.MkdirAll(filepath.Join(state, which, dir), 0700); err != nil {
				t.Fatal(err)
			}
		}
		writeShapeFile(t, filepath.Join(state, which, "conf/AdGuardHome.yaml"), initial)
		waitIntegrationSlot(t)
		docker(t, "run", "-d", "--name", prefix+"-"+which, "--network", prefix+"-net", "--ip", fmt.Sprintf("192.0.2.%d", i+3), "--user", user,
			"-v", filepath.Join(state, which, "conf")+":/opt/adguardhome/conf", "-v", filepath.Join(state, which, "work")+":/opt/adguardhome/work",
			aghImage, "--no-check-update", "--no-permcheck", "-c", "/opt/adguardhome/conf/AdGuardHome.yaml", "-w", "/opt/adguardhome/work")
		assertUnprivileged(t, prefix+"-"+which)
		base := fmt.Sprintf("http://192.0.2.%d:8080", i+3)
		waitReady(t, base, labCreds{})
		if getJSON(t, base+"/control/status", labCreds{})["version"] != "v0.107.59" {
			t.Fatal("wrong resolver version")
		}
		targetURL, _ := url.Parse(base)
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		recorder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				if err != nil {
					http.Error(w, "request read", 400)
					return
				}
				mu.Lock()
				writes = append(writes, struct {
					Role string `json:"role"`
					shapeWrite
				}{role, shapeWrite{r.Method, r.URL.Path, json.RawMessage(raw)}})
				mu.Unlock()
				r.Body = io.NopCloser(bytes.NewReader(raw))
			}
			proxy.ServeHTTP(w, r)
		}))
		t.Cleanup(recorder.Close)
		targets = append(targets, nodedns.Target{ID: role, AdminURL: recorder.URL, DNSAddr: fmt.Sprintf("192.0.2.%d:5353", i+3)})
	}
	compiled, err := dnspublish.Compile(dnspublish.Input{ProfileCode: "positive-cache", BlockResponse: "nxdomain", BlockedTTLSeconds: 60, Upstreams: []string{"192.0.2.2:5353"}, BootstrapDNS: []string{"192.0.2.2:5353"}})
	if err != nil {
		t.Fatal(err)
	}
	ap, err := nodedns.New(targets)
	if err != nil {
		t.Fatal(err)
	}
	if ev, err := ap.Apply(context.Background(), "1", compiled.Document); err != nil || !ev.Verified {
		t.Fatalf("production apply: verified=%t err=%v", ev.Verified, err)
	}
	for i, target := range targets {
		info := getJSON(t, target.AdminURL+"/control/dns_info", labCreds{})
		for key, want := range map[string]any{"cache_size": float64(4194304), "cache_ttl_min": float64(0), "cache_ttl_max": float64(0), "cache_optimistic": false, "upstream_mode": "", "upstream_timeout": float64(10)} {
			if info[key] != want {
				t.Errorf("%s %s=%v want %v", target.ID, key, info[key], want)
			}
		}
		var persisted struct {
			DNS map[string]any `yaml:"dns"`
		}
		if err := yaml.Unmarshal(readShapeFile(t, filepath.Join(state, fmt.Sprintf("resolver%d/conf/AdGuardHome.yaml", i+1))), &persisted); err != nil {
			t.Fatal(err)
		}
		if persisted.DNS["upstream_mode"] != "load_balance" || persisted.DNS["fastest_timeout"] != "1s" {
			t.Fatal("unexpected persisted mode/fastest timeout")
		}
		readbacks = append(readbacks, map[string]any{"role": target.ID, "dns_info": info, "yaml_dns": persisted.DNS})
	}
	var traces []cacheTrace
	events := func() []cachefixture.Event {
		var out []cachefixture.Event
		if err := json.Unmarshal([]byte(getBody(t, "http://192.0.2.2:8080/events", labCreds{})), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	phase := func(name string) {
		shapePost(t, "http://192.0.2.2:8080/phase", cachefixture.Phase{Case: name, Changed: true})
	}
	record := func(name, step, role string, r cacheExchange, attempts []cacheReceipt) cacheTrace {
		if r.err != nil {
			t.Fatalf("%s/%s: query failed: %v", name, step, r.err)
		}
		obs := cacheTrace{Case: name, Step: step, Started: r.started, Completed: r.completed, Responder: role, RCode: r.message.RCode, TTL: []uint32{}, ResolverEvents: attempts, AuthorityEvents: []cachefixture.Event{}}
		obs.ResolverEvents = append(obs.ResolverEvents, cacheReceipt{role, r.started, r.completed, "answered"})
		if len(attempts) > 0 {
			obs.Started = attempts[0].Started
		}
		obs.Milliseconds = obs.Completed.Sub(obs.Started).Milliseconds()
		for _, rr := range r.message.Answers {
			a, ok := rr.Body.(*dnsmessage.AResource)
			if !ok {
				t.Fatal("non-A positive response")
			}
			obs.TTL = append(obs.TTL, rr.Header.TTL)
			switch a.A {
			case [4]byte{198, 51, 100, 10}:
				obs.Class = "other_ipv4"
			case [4]byte{203, 0, 113, 20}:
				obs.Class = "changed_ipv4"
			default:
				t.Fatal("non-fixture answer")
			}
		}
		for _, ev := range events() {
			if ev.Case == name && ev.Type == 1 && !ev.Started.Before(obs.Started) && !ev.Started.After(obs.Completed) {
				obs.AuthorityEvents = append(obs.AuthorityEvents, ev)
			}
		}
		obs.CacheState = "inferred_hit_no_authority_exchange"
		if len(obs.AuthorityEvents) > 0 {
			obs.CacheState = "miss_authority_exchange_observed"
		}
		if obs.Class == "" {
			obs.CacheState = "unknown"
		}
		soa := exchangeCache("192.0.2.2:5353", name, dnsmessage.TypeSOA, time.Second)
		if soa.err != nil || len(soa.message.Answers) != 1 {
			t.Fatal("authority serial sample failed")
		}
		body, ok := soa.message.Answers[0].Body.(*dnsmessage.SOAResource)
		if !ok {
			t.Fatal("missing authority SOA")
		}
		obs.AuthoritySerial, obs.AuthorityReadAt = body.Serial, soa.completed
		traces = append(traces, obs)
		return obs
	}
	probe := func(name, step string, i int) cacheTrace {
		return record(name, step, roles[i], exchangeCache(targets[i].DNSAddr, name, dnsmessage.TypeA, 25*time.Second), nil)
	}
	check := func(obs cacheTrace, class string, ttl uint32, fetches int) {
		t.Helper()
		if obs.RCode != 0 || obs.Class != class || !reflect.DeepEqual(obs.TTL, []uint32{ttl}) || len(obs.AuthorityEvents) != fetches {
			t.Fatalf("unexpected cache observation: %+v; want %s TTL %d fetches %d", obs, class, ttl, fetches)
		}
	}
	t.Cleanup(func() {
		if dir := os.Getenv("FVPN_DNS_CACHE_EVIDENCE"); dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Error(err)
				return
			}
			writeShapeFile(t, filepath.Join(dir, "AdGuardHome.yaml"), initial)
			mu.Lock()
			writeShapeJSON(t, filepath.Join(dir, "requests.json"), writes)
			mu.Unlock()
			writeShapeJSON(t, filepath.Join(dir, "readback.json"), readbacks)
			writeShapeJSON(t, filepath.Join(dir, "document.json"), compiled.Document)
			writeShapeJSON(t, filepath.Join(dir, "traces.json"), traces)
			writeShapeJSON(t, filepath.Join(dir, "authority-events.json"), events())
			writeShapeJSON(t, filepath.Join(dir, "provenance.json"), map[string]any{"image": aghImage, "image_id": string(bytes.TrimSpace(docker(t, "inspect", "-f", "{{.Image}}", prefix+"-resolver1"))), "version": "v0.107.59", "platform": runtime.GOOS + "/" + runtime.GOARCH, "finished_at": time.Now().UTC(), "no_cache_flush_after_initial_apply": true})
		}
	})
	// Start well inside a wall-clock second to expose the pinned cache's whole-
	// second expiry, with generous margins on both sides of the boundary.
	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(1700 * time.Millisecond)))
	first := probe("steady", "initial", 0)
	check(first, "other_ipv4", 10, 1)
	boundary := probe("boundary", "initial", 0)
	check(boundary, "other_ipv4", 10, 1)
	phase("steady")
	phase("boundary")
	for second := 1; second <= 9; second++ {
		time.Sleep(time.Until(first.Completed.Add(time.Duration(second) * time.Second)))
		obs := probe("steady", fmt.Sprintf("inside_%ds", second), 0)
		check(obs, "other_ipv4", uint32(10-(obs.Completed.Unix()-first.Completed.Unix())), 0)
		if obs.AuthoritySerial != 1002 {
			t.Fatal("authority did not change")
		}
		if second == 1 {
			check(probe("steady", "other_resolver_fresh_fetch", 1), "changed_ipv4", 10, 1)
		}
	}
	time.Sleep(time.Until(boundary.Completed.Truncate(time.Second).Add(10050 * time.Millisecond)))
	early := probe("boundary", "whole_second_expiry", 0)
	check(early, "changed_ipv4", 10, 1)
	if elapsed := early.Completed.Sub(boundary.Completed); elapsed >= 10*time.Second || elapsed < 9*time.Second {
		t.Fatalf("expiry boundary not observed inside retained TTL: %s", elapsed)
	}
	time.Sleep(time.Until(first.Completed.Add(10100 * time.Millisecond)))
	check(probe("steady", "after_ttl", 0), "changed_ipv4", 10, 1)

	slow := probe("slow", "first_fetch_delayed_2500ms", 0)
	check(slow, "other_ipv4", 10, 1)
	if slow.Milliseconds < 2500 || slow.Milliseconds > 3500 {
		t.Fatal("slow fixture delay not observed")
	}
	phase("slow")
	for second := 1; second <= 9; second++ {
		time.Sleep(time.Until(slow.Completed.Add(time.Duration(second) * time.Second)))
		obs := probe("slow", fmt.Sprintf("inside_%ds", second), 0)
		check(obs, "other_ipv4", uint32(10-(obs.Completed.Unix()-slow.Completed.Unix())), 0)
	}
	time.Sleep(time.Until(slow.Completed.Add(10100 * time.Millisecond)))
	check(probe("slow", "after_ttl", 0), "changed_ipv4", 10, 1)

	retry := probe("retry", "first_packet_dropped", 0)
	check(retry, "other_ipv4", 10, 2)
	if retry.Milliseconds < 9900 || retry.Milliseconds > 12000 {
		t.Fatal("expected one 10s upstream timeout before retry")
	}
	phase("retry")
	check(probe("retry", "after_authority_change", 0), "other_ipv4", 10, 0)

	// The first response is delayed past the upstream timeout. Flip authority
	// while it is outstanding; the retry must cache the newer response. The old
	// reply arriving a second later must not replace that entry.
	result := make(chan cacheExchange, 1)
	go func() { result <- exchangeCache(targets[0].DNSAddr, "late", dnsmessage.TypeA, 25*time.Second) }()
	time.Sleep(500 * time.Millisecond)
	phase("late")
	late := record("late", "timeout_then_retry", roles[0], <-result, nil)
	check(late, "changed_ipv4", 10, 2)
	time.Sleep(time.Until(late.Started.Add(11300 * time.Millisecond)))
	obs := probe("late", "after_discarded_late_old_reply", 0)
	check(obs, "changed_ipv4", uint32(10-(obs.Completed.Unix()-late.Completed.Unix())), 0)

	failed := probe("failed", "both_upstream_attempts_dropped", 0)
	if failed.RCode != dnsmessage.RCodeServerFailure || len(failed.TTL) != 0 || len(failed.AuthorityEvents) != 2 || failed.Milliseconds < 19900 || failed.Milliseconds > 23000 {
		t.Fatalf("failed fetch: %+v", failed)
	}

	// Controlled client fallback, not an emulation of a native platform stub.
	// Only the client knows about the secondary; AdGuard fallback_dns is empty.
	start := exchangeCache(targets[0].DNSAddr, "fallback", dnsmessage.TypeA, 2500*time.Millisecond)
	if start.err == nil {
		t.Fatal("primary should exceed the client's 2.5s deadline")
	}
	fallback := record("fallback", "client_2500ms_then_secondary", roles[1], exchangeCache(targets[1].DNSAddr, "fallback", dnsmessage.TypeA, time.Second), []cacheReceipt{{roles[0], start.started, start.completed, "client_timeout"}})
	check(fallback, "other_ipv4", 10, 2)
	phase("fallback")
	check(probe("fallback", "primary_fresh_fetch_inside_secondary_ttl", 0), "changed_ipv4", 10, 1)
	check(probe("fallback", "secondary_keeps_old_entry", 1), "other_ipv4", 10, 0)
	// Drain the first primary attempt's retry so exported authority metadata is
	// complete. It is a distinct outstanding query, not an inferred cache miss.
	time.Sleep(time.Until(start.started.Add(10300 * time.Millisecond)))
	// A reply may arrive after the client's deadline but before AdGuard's
	// upstream deadline. Retain the outstanding request across query windows.
	abandoned := exchangeCache(targets[0].DNSAddr, "inflight", dnsmessage.TypeA, 2500*time.Millisecond)
	if abandoned.err == nil {
		t.Fatal("inflight primary should exceed client deadline")
	}
	old := record("inflight", "client_timeout_then_secondary", roles[1], exchangeCache(targets[1].DNSAddr, "inflight", dnsmessage.TypeA, time.Second), []cacheReceipt{{roles[0], abandoned.started, abandoned.completed, "client_timeout"}})
	check(old, "other_ipv4", 10, 2)
	phase("inflight")
	fresh := probe("inflight", "primary_while_first_fetch_outstanding", 0)
	check(fresh, "changed_ipv4", 10, 1)
	time.Sleep(time.Until(abandoned.started.Add(3800 * time.Millisecond)))
	inflight := probe("inflight", "primary_after_first_fetch_completed", 0)
	var oldCompleted time.Time
	for _, ev := range events() {
		if ev.Case == "inflight" && ev.Role == "primary" && ev.Type == 1 && ev.Serial == 1001 && ev.Outcome == "answered" {
			if !ev.Started.Before(fresh.Started) || !ev.Completed.After(fresh.Completed) || !ev.Completed.Before(inflight.Started) {
				t.Fatal("old outstanding fetch did not straddle the fresh response")
			}
			oldCompleted = ev.Completed
		}
	}
	if oldCompleted.IsZero() {
		t.Fatal("missing completion evidence for outstanding fetch")
	}
	// Pinned inherent behavior: the older in-flight reply overwrites the newer
	// entry. A trace restricted to the last query would miss its provenance.
	check(inflight, "other_ipv4", uint32(10-(inflight.Completed.Unix()-oldCompleted.Unix())), 0)
	for _, target := range targets {
		assertPrivacy(t, target.AdminURL, labCreds{})
	}
	assertNoQueryFiles(t, state)
	t.Logf("%d positive-cache/failure traces; pinned defaults, two independent caches, slow/retried/late fetches and controlled client fallback", len(traces))
}
