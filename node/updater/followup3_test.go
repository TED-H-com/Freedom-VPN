package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

type notedSource struct {
	work Work
	err  error
}

func (s notedSource) Current(context.Context) (Work, error) { return s.work, s.err }

func TestUPD_SignedRevocationRescueInstallsIntent(t *testing.T) {
	w := newWorld(t)
	r1 := w.makeRelease("r1", 1, []int{1})
	first := w.work(r1, "1", api.PublicationIntentPublished)
	first.PolicyGeneration = decstr.MustParse("1")
	first.PolicyKnown = true
	w.setWork(first)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(r1.id)

	r2 := w.makeRelease("r2", 1, []int{1})
	work := w.work(r2, "2", api.PublicationIntentPublished)
	work.PolicyGeneration = decstr.MustParse("2")
	work.PolicyKnown = true
	revoked := []uuid.UUID{r1.id}
	work.SignedRevoked = &revoked
	work.Revoked = []uuid.UUID{r1.id}
	work.Serving = []uuid.UUID{r2.id}
	var during bool
	w.eng.Hooks.AfterPhase = func(name string) error {
		if name != "downloading" {
			return nil
		}
		on, _ := w.eng.Journal.BlocksForwarding(context.Background())
		p, err := w.eng.Journal.Pointers(context.Background())
		during = err == nil && on && p.CurrentRelease == r1.id.String()
		return nil
	}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	if !during {
		t.Fatal("forwarding was open while the revoked release was still current")
	}
	w.mustCurrent(r2.id)
	if on, why := w.eng.Journal.BlocksForwarding(context.Background()); on {
		t.Fatalf("forwarding stayed blocked after R2: %s", why)
	}
	if w.oneResult().State != api.UpdateResultStateSucceeded {
		t.Fatalf("state %s", w.oneResult().State)
	}
	if w.hitsOf("/m/"+r2.id.String()) == 0 || w.hitsOf("/a/"+r2.id.String()+"/amd64") == 0 {
		t.Fatal("R2 was not fetched")
	}

	again := w.work(r2, "3", api.PublicationIntentPublished)
	again.PolicyGeneration = decstr.MustParse("3")
	again.PolicyKnown = true
	again.Serving = []uuid.UUID{r2.id}
	w.eng.Hooks.AfterPhase = nil
	w.setWork(again)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	if w.oneResult().State != api.UpdateResultStateAlreadyCurrent {
		t.Fatalf("retry %s", w.oneResult().State)
	}
	w.mustCurrent(r2.id)
}

func TestUPD_NoJobKeepsRevokedReleaseBlocked(t *testing.T) {
	w := newWorld(t)
	r1 := w.makeRelease("r1", 1, []int{1})
	first := w.work(r1, "1", api.PublicationIntentPublished)
	first.PolicyGeneration = decstr.MustParse("1")
	first.PolicyKnown = true
	w.setWork(first)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}

	revoked := []uuid.UUID{r1.id}
	w.eng.Source = notedSource{
		work: Work{
			PolicyGeneration: decstr.MustParse("2"),
			PolicyKnown:      true,
			SignedRevoked:    &revoked,
			Revoked:          []uuid.UUID{r1.id},
		},
		err: ErrNoWork,
	}
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	w.mustCurrent(r1.id)
	on, why := w.eng.Journal.BlocksForwarding(context.Background())
	if !on || why != "update recovery_required" {
		t.Fatalf("blocked %v why %q", on, why)
	}

	r2 := w.makeRelease("r2", 1, []int{1})
	work := w.work(r2, "3", api.PublicationIntentPublished)
	work.PolicyGeneration = decstr.MustParse("3")
	work.PolicyKnown = true
	work.Serving = []uuid.UUID{r2.id}
	w.eng.Source = w.src
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(r2.id)
	if on, why := w.eng.Journal.BlocksForwarding(context.Background()); on {
		t.Fatalf("later job did not resume serving: %s", why)
	}
}

func TestUPD_RevokedIntentStaysBlockedUntilRescue(t *testing.T) {
	w := newWorld(t)
	r1 := w.makeRelease("r1", 1, []int{1})
	first := w.work(r1, "1", api.PublicationIntentPublished)
	first.PolicyGeneration = decstr.MustParse("1")
	first.PolicyKnown = true
	w.setWork(first)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}

	r2 := w.makeRelease("r2", 1, []int{1})
	bad := w.work(r2, "2", api.PublicationIntentPublished)
	bad.PolicyGeneration = decstr.MustParse("2")
	bad.PolicyKnown = true
	revoked := []uuid.UUID{r1.id, r2.id}
	bad.SignedRevoked = &revoked
	bad.Revoked = append([]uuid.UUID(nil), revoked...)
	bad.Serving = []uuid.UUID{r2.id}
	w.setWork(bad)
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	if w.hitsOf("/m/"+r2.id.String()) != 0 {
		t.Fatal("revoked intent was fetched")
	}
	w.mustCurrent(r1.id)

	r3 := w.makeRelease("r3", 1, []int{1})
	good := w.work(r3, "3", api.PublicationIntentPublished)
	good.PolicyGeneration = decstr.MustParse("3")
	good.PolicyKnown = true
	good.Revoked = nil
	good.Serving = []uuid.UUID{r3.id}
	w.setWork(good)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(r3.id)
	if on, why := w.eng.Journal.BlocksForwarding(context.Background()); on {
		t.Fatalf("rescue left forwarding blocked: %s", why)
	}
}

func TestUPD_ETagMismatchRestartsPartial(t *testing.T) {
	body := bytes.Repeat([]byte("abc"), 2000)
	sum := sha256.Sum256(body)
	var attempt int
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		half := len(body) / 2
		if attempt == 1 {
			hj, _ := w.(http.Hijacker)
			conn, buf, _ := hj.Hijack()
			_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nETag: \"etag-v1\"\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n")
			_, _ = buf.Write(bytes.Repeat([]byte{0xff}, half))
			_ = buf.Flush()
			_ = conn.Close()
			return
		}
		if r.Header.Get("Range") != "" && attempt == 2 {
			w.Header().Set("ETag", `"etag-v2"`)
			w.Header().Set("Content-Range", "bytes "+strconv.Itoa(half)+"-"+strconv.Itoa(len(body)-1)+"/"+strconv.Itoa(len(body)))
			w.Header().Set("Content-Length", strconv.Itoa(len(body)-half))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(body[half:])
			return
		}
		w.Header().Set("ETag", `"etag-v2"`)
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = w.Write(body)
	}))
	srv.EnableHTTP2 = false
	srv.StartTLS()
	t.Cleanup(srv.Close)
	eng := &Engine{
		HTTP: srv.Client(), AllowLoopback: true,
		Download: DownloadConfig{MaxAttempts: 4, Initial: time.Millisecond, MaxWait: time.Millisecond, AttemptTimeout: 5 * time.Second, ContentRetries: 0},
		Now:      func() time.Time { return fixedNow() }, Sleep: func(time.Duration) {}, Jitter: func() float64 { return 0 },
	}
	dest := t.TempDir() + "/artifact"
	err := eng.download(context.Background(), srv.URL+"/art", dest, hexEncode(sum[:]), hashRaw, int64(len(body)), int64(len(body)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fileBytes(t, dest), body) {
		t.Fatal("partial was spliced across etags")
	}
}

func TestUPD_Range200IsFullBody(t *testing.T) {
	body := bytes.Repeat([]byte("xyz"), 2000)
	sum := sha256.Sum256(body)
	var attempt int
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if r.Header.Get("Range") != "" {
			w.Header().Set("ETag", `"etag-same"`)
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}
		hj, _ := w.(http.Hijacker)
		conn, buf, _ := hj.Hijack()
		half := body[:len(body)/2]
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nETag: \"etag-same\"\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n")
		_, _ = buf.Write(half)
		_ = buf.Flush()
		_ = conn.Close()
	}))
	srv.EnableHTTP2 = false
	srv.StartTLS()
	t.Cleanup(srv.Close)
	eng := &Engine{
		HTTP: srv.Client(), AllowLoopback: true,
		Download: DownloadConfig{MaxAttempts: 4, Initial: time.Millisecond, MaxWait: time.Millisecond, AttemptTimeout: 5 * time.Second, ContentRetries: 0},
		Now:      func() time.Time { return fixedNow() }, Sleep: func(time.Duration) {}, Jitter: func() float64 { return 0 },
	}
	dest := t.TempDir() + "/artifact"
	err := eng.download(context.Background(), srv.URL+"/art", dest, hexEncode(sum[:]), hashRaw, int64(len(body)), int64(len(body)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fileBytes(t, dest), body) {
		t.Fatal("200 body was discarded")
	}
	if attempt < 2 {
		t.Fatalf("attempts %d", attempt)
	}
}

func TestUPD_AllowedOriginExactMatch(t *testing.T) {
	if err := validateOrigins([]string{"https://10.1.2.3:8443/manifest"}); err == nil {
		t.Fatal("path accepted")
	}
	if err := validateOrigins([]string{"https://metadata.google.internal"}); err == nil {
		t.Fatal("metadata origin accepted")
	}
	if err := validateOrigins([]string{"https://10.1.2.3:8443"}); err != nil {
		t.Fatal(err)
	}
	policy := urlPolicy{AllowLoopback: true, AllowedOrigins: []string{"https://10.1.2.3:8443"}}
	if err := checkURLPolicy("https://10.1.2.3:8443/manifest.json", policy); err != nil {
		t.Fatal(err)
	}
	if err := checkURLPolicy("https://10.9.9.9/a", policy); err == nil {
		t.Fatal("different origin accepted")
	}
	if err := checkURLPolicy("https://10.1.2.3/a", urlPolicy{AllowLoopback: true}); err == nil {
		t.Fatal("empty allowlist accepted a private address")
	}
	if err := checkURLPolicy("https://169.254.1.1/a", urlPolicy{AllowedOrigins: []string{"https://169.254.1.1"}}); err == nil {
		t.Fatal("link-local accepted")
	}
	if err := checkURLPolicy("https://metadata.google.internal/a", urlPolicy{AllowedOrigins: []string{"https://metadata.google.internal"}}); err == nil {
		t.Fatal("metadata host accepted")
	}
	if err := checkURLPolicy("https://100.64.0.1/a", urlPolicy{AllowedOrigins: []string{"https://100.64.0.1"}}); err != nil {
		t.Fatal(err)
	}
}

func TestUPD_AllowedOriginPinsDial(t *testing.T) {
	oldLookup, oldDial := lookupIPs, dialContextFn
	t.Cleanup(func() {
		lookupIPs, dialContextFn = oldLookup, oldDial
	})
	lookupIPs = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.1.2.3"), net.ParseIP("192.168.9.9")}, nil
	}
	var dialed string
	dialContextFn = func(_ context.Context, _ string, addr string) (net.Conn, error) {
		dialed = addr
		return nil, errors.New("dial stopped")
	}
	policy := urlPolicy{AllowedOrigins: []string{"https://releases.internal"}}
	ctx := withDialScheme(context.Background(), "https")
	_, err := dialPolicy(ctx, "tcp", "releases.internal:443", policy)
	if dialed != "10.1.2.3:443" {
		t.Fatalf("dialed %q err %v", dialed, err)
	}
	dialed = ""
	_, err = dialPolicy(ctx, "tcp", "releases.internal:443", urlPolicy{})
	if dialed != "" || asOp(err) == nil || asOp(err).Code != apierr.SignatureInvalid {
		t.Fatalf("unguarded dialed %q err %v", dialed, err)
	}

	dialed = ""
	sink := HTTPSink{URL: "https://10.1.2.3/results", AllowedOrigins: []string{"https://10.1.2.3"}}
	_, _, err = sink.Submit(context.Background(), []byte(`{}`))
	if asOp(err) == nil || asOp(err).Code == apierr.SignatureInvalid || dialed != "10.1.2.3:443" {
		t.Fatalf("sink dialed %q err %v", dialed, err)
	}
	dialed = ""
	sink = HTTPSink{URL: "https://172.18.0.2:8443/results", AllowLoopback: true}
	_, _, err = sink.Submit(context.Background(), []byte(`{}`))
	if asOp(err) == nil || asOp(err).Code != apierr.SignatureInvalid || dialed != "" {
		t.Fatalf("sink without allowlist dialed %q err %v", dialed, err)
	}
}
