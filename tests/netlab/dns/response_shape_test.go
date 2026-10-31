//go:build netlab

package publishlab_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	nodedns "github.com/TED-H-com/Freedom-VPN/node/dns"
	"golang.org/x/net/dns/dnsmessage"
)

var shapeTypes = []dnsmessage.Type{
	dnsmessage.TypeA, dnsmessage.TypeAAAA, dnsmessage.TypeCNAME,
	dnsmessage.TypeSVCB, dnsmessage.TypeHTTPS, dnsmessage.TypeTXT,
	dnsmessage.TypeMX, dnsmessage.TypeNS, dnsmessage.TypeALL,
}

var shapeNames = []string{"blocked.example.com", "chain.example.com", "hop.example.com", "child.wild.example.com"}

// TestAT20_ResponseShape guards the dated BACKEND.md table against a pin bump.
// It exercises the real node applier, then changes only blocking_mode for the
// two comparison matrices. All names and addresses are TEST ONLY fixtures.
func TestAT20_ResponseShape(t *testing.T) {
	startedAt := time.Now().UTC()
	if os.Getenv("FVPN_NETLAB") != "1" {
		t.Skip("set FVPN_NETLAB=1 to run the pinned AdGuard response-shape lab")
	}
	prefix := os.Getenv("FVPN_LAB_PREFIX")
	if !prefixPattern.MatchString(prefix) {
		t.Fatal("set a valid FVPN_LAB_PREFIX")
	}
	prefix += "-shape"
	state := t.TempDir()
	t.Cleanup(func() { cleanupLab(prefix, state) })
	fixture := filepath.Join(moduleRoot(t), "tests/netlab/dns/testdata/response-shape")
	for _, path := range []string{"docs/dns/BACKEND.md", "deploy/dev/adguard/docker-compose.yml"} {
		if !bytes.Contains(readShapeFile(t, filepath.Join(moduleRoot(t), path)), []byte(aghImage)) {
			t.Fatalf("%s must use the conformance image pin", path)
		}
	}
	for _, dir := range []string{"resolver1/conf", "resolver1/work"} {
		if err := os.MkdirAll(filepath.Join(state, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	initial := readShapeFile(t, filepath.Join(fixture, "AdGuardHome.yaml"))
	writeShapeFile(t, filepath.Join(state, "resolver1/conf/AdGuardHome.yaml"), initial)
	// --internal prevents accidental traffic to third-party DNS or update hosts.
	docker(t, "network", "create", "--internal", "--subnet", "192.0.2.0/24", prefix+"-net")
	docker(t, "run", "-d", "--name", prefix+"-upstream", "--network", prefix+"-net", "--ip", "192.0.2.2",
		"-v", fixture+":/etc/coredns:ro", coreImage, "-conf", "/etc/coredns/Corefile")
	user := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
	waitIntegrationSlot(t)
	docker(t, "run", "-d", "--name", prefix+"-resolver1", "--network", prefix+"-net", "--ip", "192.0.2.3", "--user", user,
		"-v", filepath.Join(state, "resolver1/conf")+":/opt/adguardhome/conf",
		"-v", filepath.Join(state, "resolver1/work")+":/opt/adguardhome/work",
		aghImage, "--no-check-update", "--no-permcheck", "-c", "/opt/adguardhome/conf/AdGuardHome.yaml", "-w", "/opt/adguardhome/work")
	assertUnprivileged(t, prefix+"-resolver1")
	base := "http://" + shapePort(t, prefix+"-resolver1", "8080/tcp")
	waitReady(t, base, labCreds{})
	info := getJSON(t, base+"/control/status", labCreds{})
	if info["version"] != "v0.107.59" {
		t.Fatalf("unexpected resolver version: %v", info["version"])
	}
	imageID := strings.TrimSpace(string(docker(t, "inspect", "-f", "{{.Image}}", prefix+"-resolver1")))

	// Retain the actual request bodies without headers, credentials or host paths.
	var mu sync.Mutex
	var writes []shapeWrite
	targetURL, _ := url.Parse(base)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	recorder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil {
				http.Error(w, "read request", http.StatusBadRequest)
				return
			}
			mu.Lock()
			writes = append(writes, shapeWrite{r.Method, r.URL.Path, json.RawMessage(raw)})
			mu.Unlock()
			r.Body = io.NopCloser(bytes.NewReader(raw))
		}
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(recorder.Close)
	ap, err := nodedns.New([]nodedns.Target{{ID: "shape", AdminURL: recorder.URL, DNSAddr: shapePort(t, prefix+"-resolver1", "5353/udp")}})
	if err != nil {
		t.Fatal(err)
	}
	var observations []shapeObservation
	var configs []map[string]any
	var documents []json.RawMessage
	input := dnspublish.Input{ProfileCode: "response-shape", BlockResponse: "nxdomain", BlockedTTLSeconds: 60,
		Upstreams: []string{"192.0.2.2:5353"}, BootstrapDNS: []string{"192.0.2.2:5353"}}
	apply := func(generation string) {
		compiled, err := dnspublish.Compile(input)
		if err != nil {
			t.Fatal(err)
		}
		ev, err := ap.Apply(context.Background(), generation, compiled.Document)
		if err != nil || !ev.Verified {
			t.Fatalf("production apply: verified=%t err=%v", ev.Verified, err)
		}
		documents = append(documents, json.RawMessage(compiled.Bytes))
	}
	probe := func(mode, container string, ttl int, check bool) {
		for _, proto := range []string{"udp", "tcp"} {
			addr := shapePort(t, container, "5353/"+proto)
			for _, name := range shapeNames {
				for _, qt := range shapeTypes {
					if mode != "authority" {
						// Measure fresh replies; upstream AA is stripped on cache hits.
						shapePost(t, base+"/control/cache_clear", map[string]any{})
					}
					obs := queryShape(t, addr, proto, name, qt)
					obs.Mode, obs.ConfiguredTTL = mode, ttl
					observations = append(observations, obs)
					if check {
						want := expectedShape(mode, name, qt, uint32(ttl))
						if !reflect.DeepEqual(obs.responseShape, want) {
							t.Errorf("%s/%d %s %s %s: got %+v want %+v", mode, ttl, proto, name, qt, obs.responseShape, want)
						}
					} else if qt != dnsmessage.TypeALL && (obs.RCode != 0 || !slices.Contains(obs.AnswerTypes, qt.String())) {
						t.Errorf("positive control %s %s %s: %+v", mode, name, qt, obs.responseShape)
					}
				}
			}
		}
	}
	// Evidence is optional; the conformance assertions always run. Save even on
	// failure so a pin change can be diagnosed without accepting a new golden.
	t.Cleanup(func() {
		if dir := os.Getenv("FVPN_DNS_SHAPE_EVIDENCE"); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Error(err)
				return
			}
			for _, name := range []string{"AdGuardHome.yaml", "Corefile", "db.example.com"} {
				writeShapeFile(t, filepath.Join(dir, name), readShapeFile(t, filepath.Join(fixture, name)))
			}
			mu.Lock()
			defer mu.Unlock()
			writeShapeJSON(t, filepath.Join(dir, "requests.json"), writes)
			writeShapeJSON(t, filepath.Join(dir, "readback.json"), configs)
			writeShapeJSON(t, filepath.Join(dir, "documents.json"), documents)
			var raw bytes.Buffer
			for _, obs := range observations {
				if err := json.NewEncoder(&raw).Encode(obs); err != nil {
					t.Fatal(err)
				}
			}
			writeShapeFile(t, filepath.Join(dir, "probes.jsonl"), raw.Bytes())
			writeShapeJSON(t, filepath.Join(dir, "images.json"), map[string]string{
				"adguard": aghImage, "coredns": coreImage, "version": info["version"].(string),
				"adguard_container_image_id": imageID, "probe_platform": runtime.GOOS + "/" + runtime.GOARCH,
				"started_at": startedAt.Format(time.RFC3339), "finished_at": time.Now().UTC().Format(time.RFC3339),
			})
		}
	})

	apply("1")
	probe("authority", prefix+"-upstream", 60, false)
	probe("unfiltered", prefix+"-resolver1", 60, false)
	input.Rules = []dnspublish.Rule{
		{RuleID: "01900000-0000-7000-8000-000000000001", Domain: "blocked.example.com", Action: "block", Reason: "admin_rule"},
		{RuleID: "01900000-0000-7000-8000-000000000002", Domain: "wild.example.com", IncludeSubdomains: true, Action: "block", Reason: "admin_rule"},
	}
	apply("2")
	for _, mode := range []string{"nxdomain", "default", "null_ip"} {
		if mode != "nxdomain" {
			// Keep every field of the actual production body; only mode differs.
			var body map[string]any
			mu.Lock()
			for _, req := range writes {
				if req.Path == "/control/dns_config" {
					if err := json.Unmarshal(req.Body, &body); err != nil {
						t.Fatal(err)
					}
				}
			}
			mu.Unlock()
			body["blocking_mode"] = mode
			shapePost(t, recorder.URL+"/control/dns_config", body)
			shapePost(t, recorder.URL+"/control/cache_clear", map[string]any{})
		}
		readback := getJSON(t, base+"/control/dns_info", labCreds{})
		if readback["blocking_mode"] != mode || readback["blocked_response_ttl"] != float64(60) {
			t.Fatal("mode/TTL read-back mismatch")
		}
		configs = append(configs, readback)
		probe(mode, prefix+"-resolver1", 60, true)
	}
	// Disambiguate a fixed negative-cache constant from the configured TTL and
	// reproduce the members acceptance lane's explicitly chosen TEST ONLY TTL.
	input.BlockedTTLSeconds = 5
	apply("3")
	configs = append(configs, getJSON(t, base+"/control/dns_info", labCreds{}))
	probe("nxdomain", prefix+"-resolver1", 5, true)
	assertPrivacy(t, base, labCreds{})
	assertNoQueryFiles(t, state)
}

type shapeWrite struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Body   json.RawMessage `json:"body"`
}

type responseShape struct {
	RCode          dnsmessage.RCode `json:"rcode"`
	AnswerTypes    []string         `json:"answer_types"`
	AnswerTTLs     []uint32         `json:"answer_ttls"`
	AnswerTargets  []string         `json:"answer_targets"`
	Addresses      []string         `json:"addresses"`
	AuthorityTypes []string         `json:"authority_types"`
	SOATTLs        []uint32         `json:"soa_ttls"`
	SOAMinimums    []uint32         `json:"soa_minimums"`
	AA             bool             `json:"aa"`
	AD             bool             `json:"ad"`
}

type shapeObservation struct {
	Mode          string `json:"mode"`
	ConfiguredTTL int    `json:"configured_ttl"`
	Proto         string `json:"proto"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	responseShape `json:"shape"`
	RequestHex    string `json:"request_hex"`
	ResponseHex   string `json:"response_hex"`
}

func expectedShape(mode, name string, qt dnsmessage.Type, ttl uint32) responseShape {
	want := responseShape{AnswerTypes: []string{}, AnswerTTLs: []uint32{}, AnswerTargets: []string{}, Addresses: []string{}, AuthorityTypes: []string{}, SOATTLs: []uint32{}, SOAMinimums: []uint32{}}
	if name == "chain.example.com" && qt == dnsmessage.TypeCNAME {
		// A CNAME query returns only the first unblocked hop. The next query,
		// hop.example.com CNAME, exposes the blocked target and is filtered.
		want.AnswerTypes, want.AnswerTTLs = []string{qt.String()}, []uint32{300}
		want.AnswerTargets = []string{"hop.example.com."}
		want.AuthorityTypes = []string{dnsmessage.TypeNS.String()}
		want.AA = true
		return want
	}
	if qt == dnsmessage.TypeALL {
		want.RCode = dnsmessage.RCodeNotImplemented // refuse_any, before filtering
		return want
	}
	if mode != "nxdomain" && (qt == dnsmessage.TypeA || qt == dnsmessage.TypeAAAA) {
		want.AnswerTypes = []string{qt.String()}
		want.AnswerTTLs = []uint32{ttl}
		want.Addresses = []string{"0.0.0.0"}
		if qt == dnsmessage.TypeAAAA {
			want.Addresses = []string{"::"}
		}
		return want
	}
	if mode == "null_ip" || (mode == "default" && qt == dnsmessage.TypeHTTPS) {
		return want
	}
	want.SOATTLs, want.SOAMinimums = []uint32{ttl}, []uint32{86400}
	want.AuthorityTypes = []string{dnsmessage.TypeSOA.String()}
	if mode == "nxdomain" && (qt == dnsmessage.TypeA || qt == dnsmessage.TypeAAAA || qt == dnsmessage.TypeHTTPS) {
		want.RCode = dnsmessage.RCodeNameError
	}
	return want
}

func queryShape(t *testing.T, addr, proto, name string, qt dnsmessage.Type) shapeObservation {
	t.Helper()
	qname, err := dnsmessage.NewName(name + ".")
	if err != nil {
		t.Fatal(err)
	}
	req := dnsmessage.Message{Header: dnsmessage.Header{ID: 1, RecursionDesired: true, AuthenticData: true},
		Questions: []dnsmessage.Question{{Name: qname, Type: qt, Class: dnsmessage.ClassINET}}}
	raw, err := req.Pack()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.DialTimeout(proto, addr, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	wire := raw
	if proto == "tcp" {
		wire = append(binary.BigEndian.AppendUint16(nil, uint16(len(raw))), raw...)
	}
	if _, err := conn.Write(wire); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 65535)
	var size int
	if proto == "tcp" {
		if _, err = io.ReadFull(conn, buf[:2]); err != nil {
			t.Fatal(err)
		}
		size = int(binary.BigEndian.Uint16(buf[:2]))
		_, err = io.ReadFull(conn, buf[:size])
	} else {
		size, err = conn.Read(buf)
	}
	if err != nil {
		t.Fatal(err)
	}
	var msg dnsmessage.Message
	if err := msg.Unpack(buf[:size]); err != nil {
		t.Fatal(err)
	}
	if !msg.Response || msg.ID != req.ID || msg.Truncated || !reflect.DeepEqual(msg.Questions, req.Questions) {
		t.Fatal("invalid or truncated probe reply")
	}
	shape := responseShape{RCode: msg.RCode, AA: msg.Authoritative, AD: msg.AuthenticData,
		AnswerTypes: []string{}, AnswerTTLs: []uint32{}, AnswerTargets: []string{}, Addresses: []string{}, AuthorityTypes: []string{}, SOATTLs: []uint32{}, SOAMinimums: []uint32{}}
	for _, rr := range msg.Answers {
		shape.AnswerTypes = append(shape.AnswerTypes, rr.Header.Type.String())
		shape.AnswerTTLs = append(shape.AnswerTTLs, rr.Header.TTL)
		switch a := rr.Body.(type) {
		case *dnsmessage.AResource:
			shape.Addresses = append(shape.Addresses, netip.AddrFrom4(a.A).String())
		case *dnsmessage.AAAAResource:
			shape.Addresses = append(shape.Addresses, netip.AddrFrom16(a.AAAA).String())
		case *dnsmessage.CNAMEResource:
			shape.AnswerTargets = append(shape.AnswerTargets, a.CNAME.String())
		case *dnsmessage.SVCBResource:
			shape.AnswerTargets = append(shape.AnswerTargets, a.Target.String())
		case *dnsmessage.HTTPSResource:
			shape.AnswerTargets = append(shape.AnswerTargets, a.Target.String())
		}
	}
	for _, rr := range msg.Authorities {
		shape.AuthorityTypes = append(shape.AuthorityTypes, rr.Header.Type.String())
		if soa, ok := rr.Body.(*dnsmessage.SOAResource); ok {
			shape.SOATTLs = append(shape.SOATTLs, rr.Header.TTL)
			shape.SOAMinimums = append(shape.SOAMinimums, soa.MinTTL)
		}
	}
	return shapeObservation{Proto: proto, Name: name, Type: qt.String(), responseShape: shape,
		RequestHex: hex.EncodeToString(raw), ResponseHex: hex.EncodeToString(buf[:size])}
}

func shapePort(t *testing.T, container, port string) string {
	t.Helper()
	// No published ports: Linux host access is through this isolated bridge.
	ip := strings.TrimSpace(string(docker(t, "inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", container)))
	number, _, _ := strings.Cut(port, "/")
	return net.JoinHostPort(ip, number)
}

func shapePost(t *testing.T, url string, body any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST: HTTP %d", resp.StatusCode)
	}
}

func readShapeFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func writeShapeFile(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeShapeJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeShapeFile(t, path, append(raw, '\n'))
}

func waitIntegrationSlot(t *testing.T) {
	t.Helper()
	for {
		out, err := exec.Command("pgrep", "-af", "gates.sh|mk/at-").CombinedOutput()
		if err != nil && len(out) != 0 {
			t.Fatalf("integration inventory: %v", err)
		}
		// Inspect launchers, not shells whose command text happens to quote a
		// brief containing the pattern. Count nested launchers conservatively.
		suites := map[string]bool{}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			command := filepath.Base(fields[1])
			if command == "make" || ((command == "bash" || command == "sh") && len(fields) > 2 && strings.HasSuffix(fields[2], "gates.sh")) {
				suites[line] = true
			}
		}
		if len(suites) < 2 {
			return
		}
		t.Logf("waiting: %d other integration launchers active", len(suites))
		time.Sleep(15 * time.Second)
	}
}
