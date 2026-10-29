package updater

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

type builtRel struct {
	id       uuid.UUID
	manifest api.ReleaseManifest
	env      []byte
	sum      string
	tree     string
	amd      []byte
	arm      []byte
	amdSHA   string
}

type fileSpec struct {
	body []byte
	chop bool
	bad  []byte
}

type world struct {
	t             *testing.T
	root          string
	layout        Layout
	node          Node
	src           *memSource
	gate          *fakeGate
	svc           *fakeSvc
	meter         *fakeMeter
	health        *fakeHealth
	mig           *fakeMig
	eng           *Engine
	key           release.SigningKey
	trust         *envelope.TrustStore
	srv           *httpServer
	files         map[string]fileSpec
	hits          map[string]int
	mu            sync.Mutex
	onGet         func(r *http.Request)
	now           time.Time
	rollbackAllow []uuid.UUID
	releaseSeq    string
}

type httpServer struct {
	URL     string
	Client  *http.Client
	CertPEM []byte
	close   func()
}

func (w *world) serve(rw http.ResponseWriter, r *http.Request) {
	if w.onGet != nil {
		w.onGet(r)
	}
	w.mu.Lock()
	w.hits[r.URL.Path]++
	n := w.hits[r.URL.Path]
	spec, ok := w.files[r.URL.Path]
	w.mu.Unlock()
	if !ok {
		http.NotFound(rw, r)
		return
	}
	if spec.chop && n == 1 && r.Header.Get("Range") == "" {
		hj, ok := rw.(http.Hijacker)
		if !ok {
			http.Error(rw, "no hijack", 500)
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			return
		}
		half := spec.body[:len(spec.body)/2]
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Length: " + strconv.Itoa(len(spec.body)) + "\r\nConnection: close\r\n\r\n")
		_, _ = buf.Write(half)
		_ = buf.Flush()
		_ = conn.Close()
		return
	}
	body := spec.body
	if spec.chop && n > 1 {
		body = spec.bad
	}
	rw.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = rw.Write(body)
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{
		t: t, root: t.TempDir(), now: fixedNow(),
		src: &memSource{}, gate: &fakeGate{}, svc: &fakeSvc{},
		meter: newMeter(), health: &fakeHealth{fail: map[string]bool{}}, mig: &fakeMig{},
		files: map[string]fileSpec{}, hits: map[string]int{},
	}
	w.layout = testLayout(w.root)
	if err := w.layout.Ensure(); err != nil {
		t.Fatal(err)
	}
	w.node = Node{ID: uuid.Must(uuid.NewV7()), Incarnation: uuid.Must(uuid.NewV7()), BootID: uuid.Must(uuid.NewV7())}
	key, err := release.GenerateKey("release-test-" + w.node.ID.String()[:8])
	if err != nil {
		t.Fatal(err)
	}
	w.key = key
	ts, err := key.TrustStore(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	w.trust = ts
	srv := newTestServer(t, http.HandlerFunc(w.serve))
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	w.srv = &httpServer{URL: srv.URL, Client: tlsClient(srv), CertPEM: certPEM, close: srv.Close}
	j := openJournal(t, w.layout.Journal)
	w.eng = &Engine{
		Layout: w.layout, Node: w.node, Platform: testPlatform(),
		ReleaseTrust: ts, Source: w.src, HTTP: w.srv.Client,
		Meter: w.meter, Services: w.svc, Health: w.health, Gate: w.gate, Migrator: w.mig,
		Journal: j, Download: DownloadConfig{MaxAttempts: 4, Initial: time.Millisecond, MaxWait: 5 * time.Millisecond, AttemptTimeout: 5 * time.Second, ContentRetries: 1},
		Now: func() time.Time { return w.now }, Sleep: func(time.Duration) {}, Jitter: func() float64 { return 0 },
		AllowLoopback: true,
	}
	return w
}

func (w *world) makeRelease(marker string, schemaTo int, from []int) builtRel {
	w.t.Helper()
	id := uuid.Must(uuid.NewV7())
	base := filepath.Join(w.root, "src", id.String())
	amdTree := filepath.Join(base, "amd")
	armTree := filepath.Join(base, "arm")
	writeMarker(w.t, amdTree, marker+"-amd64")
	writeMarker(w.t, armTree, marker+"-arm64")
	amdTar := filepath.Join(base, "amd64.tar.zst")
	armTar := filepath.Join(base, "arm64.tar.zst")
	if err := release.PackageDir(amdTree, amdTar, time.Unix(0, 0).UTC()); err != nil {
		w.t.Fatal(err)
	}
	if err := release.PackageDir(armTree, armTar, time.Unix(0, 0).UTC()); err != nil {
		w.t.Fatal(err)
	}
	sbom := filepath.Join(base, "sbom.json")
	if err := os.WriteFile(sbom, []byte("{}\n"), 0o644); err != nil {
		w.t.Fatal(err)
	}
	if schemaTo < 1 {
		schemaTo = 1
	}
	if from == nil {
		from = []int{1}
	}
	seq := w.releaseSeq
	if seq == "" {
		seq = "4"
	}
	in := release.BuildInput{
		ReleaseID: id.String(), ProductVersion: "1.2.0", ReleaseSequence: seq,
		CreatedAt:    "2026-10-28T14:00:00Z",
		SourceCommit: "1111111111111111111111111111111111111111", BuildID: "build-test",
		Components: api.ManifestComponents{NodeAgent: "1.2.0", NodeUpdater: "1.2.0", Netbird: "0.76.2", Enforcement: "1.2.0"},
		Compatibility: api.ManifestCompatibility{
			ControlApiMajor: 1, PolicySchemaVersions: []int{1}, UsageEventSchemaVersions: []int{1},
			MinimumUpdaterVersion: "1.0.0", StateSchemaFrom: from, StateSchemaTo: schemaTo,
			RequiresServiceRestart: true,
			MigrationId:            presence.NewNullNullable[string](),
		},
		Rollback: api.ManifestRollback{
			AllowedReleaseIds: append([]uuid.UUID{}, w.rollbackAllow...), StateReadableByPrevious: true,
			RequiredFreeBytes: decstr.MustParse("4096"), HealthProfile: "vpn-node-standard-v1",
		},
		Evidence: release.EvidenceInput{
			SBOMURL: "https://releases.example.invalid/sbom.json", SBOMPath: sbom,
			NoticesURL: "https://releases.example.invalid/NOTICE.txt",
			SourceURL:  "https://releases.example.invalid/source.tar.zst",
		},
		Artifacts: []release.ArtifactInput{
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "amd64", Path: amdTar,
				URL: w.srv.URL + "/a/" + id.String() + "/amd64", InstalledSizeBytes: "1000000"},
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "arm64", Path: armTar,
				URL: w.srv.URL + "/a/" + id.String() + "/arm64", InstalledSizeBytes: "1000000"},
		},
	}
	signed, manifest, err := release.BuildManifest(in, w.now)
	if err != nil {
		w.t.Fatal(err)
	}
	_, canon, err := release.Sign(signed, w.key)
	if err != nil {
		w.t.Fatal(err)
	}
	sum, err := release.EnvelopeSHA256(canon)
	if err != nil {
		w.t.Fatal(err)
	}
	amd := fileBytes(w.t, amdTar)
	arm := fileBytes(w.t, armTar)
	w.mu.Lock()
	w.files["/m/"+id.String()] = fileSpec{body: canon}
	w.files["/a/"+id.String()+"/amd64"] = fileSpec{body: amd}
	w.files["/a/"+id.String()+"/arm64"] = fileSpec{body: arm}
	w.mu.Unlock()
	h := sha256.Sum256(amd)
	return builtRel{id: id, manifest: manifest, env: canon, sum: sum, tree: amdTree, amd: amd, arm: arm, amdSHA: hexEncode(h[:])}
}

func hexEncode(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = hexdigits[c>>4]
		out[i*2+1] = hexdigits[c&0x0f]
	}
	return string(out)
}

// keep records an installed release without moving current or last_good,
// so rollback can still choose it. Retention has not run yet.
func (w *world) keep(rel builtRel) {
	w.t.Helper()
	dir := w.layout.releaseDir(rel.id.String())
	copyTree(w.t, rel.tree, dir)
	sum := treeOrDie(w.t, dir)
	if err := w.eng.Journal.PutRelease(context.Background(), updatejournal.Release{
		ReleaseID: rel.id.String(), ManifestSHA256: rel.sum, ManifestEnvelope: rel.env,
		ArtifactSHA256: rel.amdSHA, TreeHash: sum, StateSchemaTo: rel.manifest.Compatibility.StateSchemaTo,
	}); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) seed(rel builtRel) {
	w.t.Helper()
	dir := w.layout.releaseDir(rel.id.String())
	copyTree(w.t, rel.tree, dir)
	sum := treeOrDie(w.t, dir)
	if err := w.eng.Journal.PutRelease(context.Background(), updatejournal.Release{
		ReleaseID: rel.id.String(), ManifestSHA256: rel.sum, ManifestEnvelope: rel.env,
		ArtifactSHA256: rel.amdSHA, TreeHash: sum, StateSchemaTo: rel.manifest.Compatibility.StateSchemaTo,
	}); err != nil {
		w.t.Fatal(err)
	}
	if err := w.eng.Journal.SetPointers(context.Background(), updatejournal.Pointers{CurrentRelease: rel.id.String(), LastGood: rel.id.String()}); err != nil {
		w.t.Fatal(err)
	}
	if err := w.eng.Journal.SetStateSchema(context.Background(), 1); err != nil {
		w.t.Fatal(err)
	}
	link, err := w.layout.relativeRelease(rel.id.String())
	if err != nil {
		w.t.Fatal(err)
	}
	if err := fsutil.SwitchSymlink(w.layout.Current, link); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) work(rel builtRel, gen string, intent api.PublicationIntent) Work {
	if intent == "" {
		intent = api.PublicationIntentPublished
	}
	return Work{
		Intent: api.UpdateIntent{
			Attempt: 1, AuthorizationValidUntil: w.now.Add(time.Hour),
			Generation: decstr.MustParse(gen), IncarnationId: w.node.Incarnation,
			IssuedAt: w.now.Add(-time.Minute), JobId: uuid.Must(uuid.NewV7()),
			ManifestSha256: rel.sum, ManifestUrl: w.srv.URL + "/m/" + rel.id.String(),
			NodeId: w.node.ID, PublicationIntent: intent, ReleaseId: rel.id,
			BatchId:  presence.NewNullNullable[uuid.UUID](),
			TargetId: presence.NewNullNullable[uuid.UUID](),
		},
		Serving: []uuid.UUID{rel.id}, Revoked: []uuid.UUID{},
		PolicyGeneration: decstr.MustParse("3"),
	}
}

func (w *world) setWork(work Work) { w.src.set(work) }

func (w *world) run() error { return w.eng.Run(context.Background()) }

func (w *world) hitsOf(path string) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.hits[path]
}

func (w *world) mustCurrent(id uuid.UUID) {
	w.t.Helper()
	p, err := w.eng.Journal.Pointers(context.Background())
	if err != nil {
		w.t.Fatal(err)
	}
	link, err := w.layout.currentID()
	if err != nil {
		w.t.Fatal(err)
	}
	if p.CurrentRelease != id.String() || link != id.String() {
		w.t.Fatalf("current journal %s symlink %s, want %s", p.CurrentRelease, link, id)
	}
}

func (w *world) outcomes() []string {
	w.t.Helper()
	box, err := w.eng.Journal.Outbox(context.Background())
	if err != nil {
		w.t.Fatal(err)
	}
	var out []string
	for _, e := range box {
		out = append(out, e.State)
	}
	hist, err := w.eng.Journal.History(context.Background(), "")
	if err != nil {
		w.t.Fatal(err)
	}
	for _, h := range hist {
		out = append(out, "ack:"+h.State)
	}
	return out
}

func (w *world) oneResult() api.UpdateResult {
	w.t.Helper()
	box, err := w.eng.Journal.Outbox(context.Background())
	if err != nil || len(box) == 0 {
		w.t.Fatalf("outbox %v %v", box, err)
	}
	var res api.UpdateResult
	if err := json.Unmarshal(box[len(box)-1].Payload, &res); err != nil {
		w.t.Fatal(err)
	}
	sum, err := canonjson.HashHex(box[len(box)-1].Payload, "payload_hash")
	if err != nil || sum != res.PayloadHash || sum != box[len(box)-1].PayloadHash {
		w.t.Fatalf("payload hash %s stored %s field %s err %v", sum, box[len(box)-1].PayloadHash, res.PayloadHash, err)
	}
	return res
}

func TestUPD_SuccessfulSwitchPreservesIdentity(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	idPath, usagePath := touchPersistent(t, w.layout)
	idBefore, usageBefore := fileBytes(t, idPath), fileBytes(t, usagePath)
	work := w.work(next, "7", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(next.id)
	stops, starts := w.svc.counts()
	// Stop runs at the stop stage and again before migrate. Both calls are
	// idempotent (§13.6 step 9): a crash between them must still find the
	// services stopped. Start runs at switch-complete and again for the check.
	if stops < 1 || starts < 1 {
		t.Fatalf("stops %d starts %d", stops, starts)
	}
	assertSameBytes(t, idPath, idBefore)
	assertSameBytes(t, usagePath, usageBefore)
	res := w.oneResult()
	if res.State != api.UpdateResultStateSucceeded || res.PublicationIntent != api.PublicationIntentPublished {
		t.Fatalf("result %+v", res.State)
	}
	if !res.MemberForwardingEnabled || requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("forwarding %v recovery %v", res.MemberForwardingEnabled, requiredValue(t, res.RecoveryRequired))
	}
	if res.PayloadHash == "" || res.ResultId == uuid.Nil {
		t.Fatal("missing result id or hash")
	}
	blocked, _ := w.eng.Journal.BlocksForwarding(context.Background())
	if blocked {
		t.Fatal("forwarding still blocked after success")
	}
	if w.hitsOf("/a/"+next.id.String()+"/arm64") != 0 {
		t.Fatal("downloaded the wrong architecture")
	}
	if w.hitsOf("/a/"+next.id.String()+"/amd64") == 0 {
		t.Fatal("did not download amd64")
	}
	// Late duplicate ACK is a no-op; a different hash conflicts (§13.8).
	box, _ := w.eng.Journal.Outbox(context.Background())
	dup, err := w.eng.NoteAck(context.Background(), box[0].ResultID, box[0].PayloadHash, false, []byte(`{"accepted":true}`))
	if err != nil || dup {
		t.Fatalf("first ack dup %v err %v", dup, err)
	}
	dup, err = w.eng.NoteAck(context.Background(), box[0].ResultID, box[0].PayloadHash, false, []byte(`{"accepted":true}`))
	if err != nil || !dup {
		t.Fatalf("late ack dup %v err %v", dup, err)
	}
	if _, err := w.eng.NoteAck(context.Background(), box[0].ResultID, strings.Repeat("ab", 32), false, []byte(`{}`)); err == nil {
		t.Fatal("conflicting ack was accepted")
	}
}

func TestUPD_AlreadyCurrentDoesNotRestart(t *testing.T) {
	w := newWorld(t)
	cur := w.makeRelease("cur", 1, []int{1})
	w.seed(cur)
	w.setWork(w.work(cur, "3", api.PublicationIntentPublished))
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	stops, starts := w.svc.counts()
	if stops != 0 || starts != 0 {
		t.Fatalf("restarted stops %d starts %d", stops, starts)
	}
	res := w.oneResult()
	if res.State != api.UpdateResultStateAlreadyCurrent {
		t.Fatalf("state %s", res.State)
	}
}

func TestUPD_BelowMinimumDoesNotRestart(t *testing.T) {
	w := newWorld(t)
	cur := w.makeRelease("cur", 1, []int{1})
	w.seed(cur)
	work := w.work(cur, "3", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{}
	w.setWork(work)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.MinimumVersionViolation {
		t.Fatal(err)
	}
	stops, _ := w.svc.counts()
	if stops != 0 {
		t.Fatalf("restarted: %d stops", stops)
	}
}

func TestUPD_DisabledStaysDisabled(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	work := w.work(next, "8", api.PublicationIntentDisabled)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	res := w.oneResult()
	blocked, _, unblocks := w.gate.snapshot()
	if res.PublicationIntent != api.PublicationIntentDisabled || res.MemberForwardingEnabled || unblocks != 0 || !blocked {
		t.Fatalf("intent %s forwarding %v unblocks %d blocked %v", res.PublicationIntent, res.MemberForwardingEnabled, unblocks, blocked)
	}
}

func TestUPD_BadSignatureDoesNotFetchArtifact(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	other, err := release.GenerateKey("other-release-key")
	if err != nil {
		t.Fatal(err)
	}
	signed, err := canonjson.Canonicalize(next.env)
	if err != nil {
		t.Fatal(err)
	}
	var env envelope.Envelope
	if err := json.Unmarshal(signed, &env); err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(env.Signed, &obj); err != nil {
		t.Fatal(err)
	}
	badEnv, err := envelope.Sign(obj, other.Private(), other.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	bad, err := badEnv.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	// The desired-state hash is of these bytes, so the download matches and
	// the release signature is what fails.
	sum := sha256.Sum256(bad)
	sumHex := hexEncode(sum[:])
	w.mu.Lock()
	w.files["/m/"+next.id.String()] = fileSpec{body: bad}
	w.mu.Unlock()
	work := w.work(next, "9", api.PublicationIntentPublished)
	work.Intent.ManifestSha256 = sumHex
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	err = w.run()
	op := asOp(err)
	if op == nil || (op.Code != apierr.SignatureInvalid && op.Code != apierr.KeyUntrusted) {
		t.Fatal(err)
	}
	if w.hitsOf("/a/"+next.id.String()+"/amd64") != 0 {
		t.Fatal("artifact was fetched after a bad signature")
	}
	w.mustCurrent(old.id)
}

func TestUPD_MissingBaselineArchitecture(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	w.seed(old)
	// Both artifacts are present for the schema, but arm64 is not Ubuntu 24.04.
	next := w.makeRelease("partial", 1, []int{1})
	var env envelope.Envelope
	if err := json.Unmarshal(next.env, &env); err != nil {
		t.Fatal(err)
	}
	raw := string(env.Signed)
	raw = strings.Replace(raw, `"os_versions":["24.04"]`, `"os_versions":["22.04"]`, 1)
	// Replace only the arm64 entry: the first os_versions is amd64 if artifacts
	// are ordered amd64 then arm64. Confirm both remain 24.04 if the replace
	// hit amd64; do a second targeted replace on the arm64 object instead.
	if strings.Count(raw, `"os_versions":["22.04"]`) != 1 {
		t.Fatal("os version replace failed")
	}
	// The first artifact is amd64. Put 24.04 back on it and change arm64.
	raw = strings.Replace(raw, `"os_versions":["22.04"]`, `"os_versions":["24.04"]`, 1)
	raw = strings.Replace(raw, `"architecture":"arm64","format":"tar.zst","installed_size_bytes":"1000000","os_id":"ubuntu","os_versions":["24.04"]`,
		`"architecture":"arm64","format":"tar.zst","installed_size_bytes":"1000000","os_id":"ubuntu","os_versions":["22.04"]`, 1)
	if !strings.Contains(raw, `"os_versions":["22.04"]`) {
		t.Fatalf("arm64 os version was not changed: %s", raw)
	}
	resigned, err := envelope.Sign([]byte(raw), w.key.Private(), w.key.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := resigned.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	// Schema still accepts 22.04. requireBothArch must reject it.
	if err := validateReleaseEnvelope(canon); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canon)
	w.mu.Lock()
	w.files["/m/"+next.id.String()] = fileSpec{body: canon}
	w.mu.Unlock()
	work := w.work(next, "4", api.PublicationIntentPublished)
	work.Intent.ManifestSha256 = hexEncode(sum[:])
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	err = w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.UnsupportedPlatform {
		t.Fatal(err)
	}
	if w.hitsOf("/a/"+next.id.String()+"/amd64") != 0 {
		t.Fatal("artifact fetched for an incomplete release")
	}
}

func TestUPD_HashMismatchIsPermanent(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	bad := bytes.Repeat([]byte("x"), len(next.amd))
	w.mu.Lock()
	w.files["/a/"+next.id.String()+"/amd64"] = fileSpec{body: bad}
	w.mu.Unlock()
	work := w.work(next, "5", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.HashMismatch || asOp(err).Retry {
		t.Fatal(err)
	}
	w.mustCurrent(old.id)
	job, err := w.eng.Journal.GetJob(context.Background(), work.Intent.JobId.String(), 1)
	if err != nil || job.Phase != updatejournal.JobDone {
		t.Fatalf("job %+v %v", job.Phase, err)
	}
}

func TestAT26_ResumeAfterContentChange(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	bad := append([]byte(nil), next.amd...)
	bad[len(bad)/2] ^= 0xff
	w.mu.Lock()
	spec := w.files["/a/"+next.id.String()+"/amd64"]
	spec.chop = true
	spec.bad = bad
	w.files["/a/"+next.id.String()+"/amd64"] = spec
	w.mu.Unlock()
	work := w.work(next, "6", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.HashMismatch {
		t.Fatal(err)
	}
	if w.hitsOf("/a/"+next.id.String()+"/amd64") < 3 {
		t.Fatalf("hits %d, want a partial, a full re-download and a failed verify", w.hitsOf("/a/"+next.id.String()+"/amd64"))
	}
	w.mustCurrent(old.id)
}

func TestUPD_HostileArchive(t *testing.T) {
	for _, kind := range []string{"traversal", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			w := newWorld(t)
			old := w.makeRelease("old", 1, []int{1})
			w.seed(old)
			idPath, usagePath := touchPersistent(t, w.layout)
			beforeID, beforeUsage := fileBytes(t, idPath), fileBytes(t, usagePath)
			next := w.makeRelease("bad", 1, []int{1})
			// A traversal or a symlink that escapes would land in the releases
			// directory, next to the release id, as "escaped".
			escaped := filepath.Join(w.layout.Releases, "escaped")
			replaceAmd(t, w, next, hostileTar(t, kind))
			work := w.work(next, "11", api.PublicationIntentPublished)
			work.Serving = []uuid.UUID{old.id, next.id}
			w.mu.Lock()
			body := w.files["/m/"+next.id.String()].body
			w.mu.Unlock()
			sum := sha256.Sum256(body)
			work.Intent.ManifestSha256 = hexEncode(sum[:])
			w.setWork(work)
			if err := w.run(); err == nil {
				t.Fatal("hostile archive installed")
			}
			if _, statErr := os.Stat(escaped); !os.IsNotExist(statErr) {
				t.Fatalf("archive escaped into %s: %v", escaped, statErr)
			}
			if link := filepath.Join(w.layout.releaseDir(next.id.String()), "escape"); kind == "symlink" {
				if _, statErr := os.Lstat(link); !os.IsNotExist(statErr) {
					t.Fatalf("symlink was left behind: %v", statErr)
				}
			}
			assertSameBytes(t, idPath, beforeID)
			assertSameBytes(t, usagePath, beforeUsage)
			w.mustCurrent(old.id)
		})
	}
}

func hostileTar(t *testing.T, kind string) []byte {
	t.Helper()
	var mem tarMember
	switch kind {
	case "symlink":
		mem = tarMember{name: "escape", typ: tar.TypeSymlink, link: "../escaped"}
	default:
		mem = tarMember{name: "../escaped", body: "nope"}
	}
	b, err := io.ReadAll(packTar(t, []tarMember{mem}))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func replaceAmd(t *testing.T, w *world, rel builtRel, amd []byte) {
	t.Helper()
	base := filepath.Join(w.root, "src", rel.id.String()+"-bad")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	amdPath := filepath.Join(base, "amd64.tar.zst")
	armPath := filepath.Join(base, "arm64.tar.zst")
	if err := os.WriteFile(amdPath, amd, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(armPath, rel.arm, 0o644); err != nil {
		t.Fatal(err)
	}
	sbom := filepath.Join(base, "sbom.json")
	if err := os.WriteFile(sbom, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := release.BuildInput{
		ReleaseID: rel.id.String(), ProductVersion: "1.2.0", ReleaseSequence: "4",
		CreatedAt:    "2026-10-28T14:00:00Z",
		SourceCommit: "1111111111111111111111111111111111111111", BuildID: "build-test",
		Components: api.ManifestComponents{NodeAgent: "1.2.0", NodeUpdater: "1.2.0", Netbird: "0.76.2", Enforcement: "1.2.0"},
		Compatibility: api.ManifestCompatibility{
			ControlApiMajor: 1, PolicySchemaVersions: []int{1}, UsageEventSchemaVersions: []int{1},
			MinimumUpdaterVersion: "1.0.0", StateSchemaFrom: []int{1}, StateSchemaTo: 1, RequiresServiceRestart: true,
			MigrationId: presence.NewNullNullable[string](),
		},
		Rollback: api.ManifestRollback{
			AllowedReleaseIds: []uuid.UUID{}, StateReadableByPrevious: true,
			RequiredFreeBytes: decstr.MustParse("4096"), HealthProfile: "vpn-node-standard-v1",
		},
		Evidence: release.EvidenceInput{
			SBOMURL: "https://releases.example.invalid/sbom.json", SBOMPath: sbom,
			NoticesURL: "https://releases.example.invalid/NOTICE.txt",
			SourceURL:  "https://releases.example.invalid/source.tar.zst",
		},
		Artifacts: []release.ArtifactInput{
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "amd64", Path: amdPath,
				URL: w.srv.URL + "/a/" + rel.id.String() + "/amd64", InstalledSizeBytes: "1000000"},
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "arm64", Path: armPath,
				URL: w.srv.URL + "/a/" + rel.id.String() + "/arm64", InstalledSizeBytes: "1000000"},
		},
	}
	signed, _, err := release.BuildManifest(in, w.now)
	if err != nil {
		t.Fatal(err)
	}
	_, canon, err := release.Sign(signed, w.key)
	if err != nil {
		t.Fatal(err)
	}
	w.mu.Lock()
	w.files["/m/"+rel.id.String()] = fileSpec{body: canon}
	w.files["/a/"+rel.id.String()+"/amd64"] = fileSpec{body: amd}
	w.mu.Unlock()
}

func TestUPD_DiskFullLeavesCurrent(t *testing.T) {
	for _, full := range []string{"download", "extract"} {
		t.Run(full, func(t *testing.T) {
			w := newWorld(t)
			old := w.makeRelease("old", 1, []int{1})
			next := w.makeRelease("new", 1, []int{1})
			w.seed(old)
			if full == "download" {
				w.eng.Faults.DiskFullDownload = true
			} else {
				w.eng.Faults.DiskFullExtract = true
			}
			work := w.work(next, "12", api.PublicationIntentPublished)
			work.Serving = []uuid.UUID{old.id, next.id}
			w.setWork(work)
			err := w.run()
			if asOp(err) == nil || asOp(err).Code != apierr.DiskSpaceLow || !asOp(err).Retry {
				t.Fatal(err)
			}
			w.mustCurrent(old.id)
			job, jerr := w.eng.Journal.GetJob(context.Background(), work.Intent.JobId.String(), 1)
			if jerr != nil || job.Phase == updatejournal.JobDone {
				t.Fatalf("job finished on disk full: %+v %v", job.Phase, jerr)
			}
		})
	}
}

func TestUPD_RollbackRestoresLastGood(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	idPath, usagePath := touchPersistent(t, w.layout)
	beforeID, beforeUsage := fileBytes(t, idPath), fileBytes(t, usagePath)
	w.health.fail[next.id.String()] = true
	work := w.work(next, "13", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(old.id)
	res := w.oneResult()
	if res.State != api.UpdateResultStateRolledBack || requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("state %s recovery %v", res.State, requiredValue(t, res.RecoveryRequired))
	}
	assertSameBytes(t, idPath, beforeID)
	assertSameBytes(t, usagePath, beforeUsage)
	blocked, _ := w.eng.Journal.BlocksForwarding(context.Background())
	if blocked {
		t.Fatal("members stayed blocked after a verified rollback")
	}
}

func TestUPD_RecoveryRequiredWhenLastGoodUnsafe(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	w.health.fail[next.id.String()] = true
	work := w.work(next, "14", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{next.id} // last_good is outside the serving set
	w.setWork(work)
	err := w.run()
	if !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	blocked, why := w.eng.Journal.BlocksForwarding(context.Background())
	if !blocked || why == "" {
		t.Fatal("members were not held")
	}
	res := w.oneResult()
	if !requiredValue(t, res.RecoveryRequired) || res.MemberForwardingEnabled {
		t.Fatalf("recovery %v forwarding %v", requiredValue(t, res.RecoveryRequired), res.MemberForwardingEnabled)
	}
}

func TestUPD_SupersedeEachPhase(t *testing.T) {
	phases := []string{"claimed", "downloading", "verifying", "verified"}
	for _, phase := range phases {
		t.Run(phase, func(t *testing.T) {
			w := newWorld(t)
			old := w.makeRelease("old", 1, []int{1})
			first := w.makeRelease("first", 1, []int{1})
			second := w.makeRelease("second", 1, []int{1})
			w.seed(old)
			w1 := w.work(first, "20", api.PublicationIntentPublished)
			w1.Serving = []uuid.UUID{old.id, first.id, second.id}
			w2 := w.work(second, "21", api.PublicationIntentPublished)
			w2.Serving = w1.Serving
			w.setWork(w1)
			w.eng.Hooks.AfterPhase = func(name string) error {
				if name == phase {
					w.src.set(w2)
				}
				return nil
			}
			if err := w.run(); err != nil {
				t.Fatal(err)
			}
			w.mustCurrent(second.id)
			got := w.outcomes()
			if !containsStr(got, string(api.UpdateResultStateSuperseded)) || !containsStr(got, string(api.UpdateResultStateSucceeded)) {
				t.Fatalf("outcomes %v", got)
			}
		})
	}
}

func TestUPD_SupersedeDuringApplyFinishesCurrent(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	first := w.makeRelease("first", 1, []int{1})
	second := w.makeRelease("second", 1, []int{1})
	w.seed(old)
	w1 := w.work(first, "30", api.PublicationIntentPublished)
	w1.Serving = []uuid.UUID{old.id, first.id, second.id}
	w2 := w.work(second, "31", api.PublicationIntentPublished)
	w2.Serving = w1.Serving
	w.setWork(w1)
	w.eng.Hooks.AfterStage = func(stage string) error {
		if stage == string(updatejournal.StagePrepared) {
			w.src.set(w2)
		}
		return nil
	}
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(second.id)
	// The first transaction reached success; it was not abandoned mid-apply.
	hist := map[string]bool{}
	box, err := w.eng.Journal.Outbox(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range box {
		var res api.UpdateResult
		if json.Unmarshal(e.Payload, &res) != nil {
			t.Fatal("payload")
		}
		hist[res.RequestedReleaseId.String()+" "+string(res.State)] = true
	}
	if !hist[first.id.String()+" "+string(api.UpdateResultStateSucceeded)] {
		t.Fatalf("first release was not finished: %v", hist)
	}
	if !hist[second.id.String()+" "+string(api.UpdateResultStateSucceeded)] {
		t.Fatalf("second release did not run: %v", hist)
	}
}

func TestUPD_AuthorizationExpiredDoesNotClaim(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	work := w.work(next, "2", api.PublicationIntentPublished)
	work.Intent.AuthorizationValidUntil = w.now.Add(-time.Minute)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.AuthorizationExpired {
		t.Fatal(err)
	}
	if _, _, ok, gerr := w.eng.Journal.AcceptedGeneration(context.Background()); gerr != nil || ok {
		t.Fatalf("generation accepted: %v %v", ok, gerr)
	}
}

func TestUPD_IncarnationMismatchDoesNotClaim(t *testing.T) {
	w := newWorld(t)
	next := w.makeRelease("new", 1, []int{1})
	work := w.work(next, "2", api.PublicationIntentPublished)
	work.Intent.IncarnationId = uuid.Must(uuid.NewV7())
	w.setWork(work)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.IncarnationMismatch {
		t.Fatal(err)
	}
	if _, _, ok, _ := w.eng.Journal.AcceptedGeneration(context.Background()); ok {
		t.Fatal("mismatch claimed a generation")
	}
	if w.hitsOf("/m/"+next.id.String()) != 0 {
		t.Fatal("manifest fetched")
	}
}

func TestUPD_BlockFailureKeepsTransaction(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	w.gate.err = errors.New("nft busy")
	work := w.work(next, "9", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.UpdateBusy {
		t.Fatal(err)
	}
	txn, terr := w.eng.Journal.ActiveTxn(context.Background())
	if terr != nil || txn.Stage != updatejournal.StagePrepared {
		t.Fatalf("txn %+v %v", txn.Stage, terr)
	}
	blocked, _ := w.eng.Journal.BlocksForwarding(context.Background())
	if !blocked {
		t.Fatal("journal did not keep members blocked")
	}
}

func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func requiredValue[T any](t *testing.T, value interface{ Get() (T, error) }) T {
	t.Helper()
	v, err := value.Get()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
