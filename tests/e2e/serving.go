//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// ServingBaseline identifies verified, signed tree-built releases and the
// non-empty API-activated support policy. It does not claim node readiness.
type ServingBaseline struct {
	Boot, Good, Bad uuid.UUID
	SupportRevision uuid.UUID
	Digests         map[string]string
}

// PrepareServingBaseline is shared by go test harnesses (-tags e2e). First run
// fvpn-e2e-prepare/up.sh for the harness prefix, then pass an authenticated admin
// cookie client and its CSRF token. FVPN_E2E_STATE selects the prepared stack.
// Call before delivering an enrollment token. candidates adds two real bundles
// for update tests. The helper never writes database rows or readiness values.
type BaselineOptions struct {
	Candidates            bool
	Rollback              ReleaseOptions
	IncludeBootInRollback bool
}

func PrepareServingBaseline(ctx context.Context, client *http.Client, csrf string, candidates bool) (ServingBaseline, error) {
	return PrepareServingBaselineWithOptions(ctx, client, csrf, BaselineOptions{Candidates: candidates})
}

func PrepareServingBaselineWithOptions(ctx context.Context, client *http.Client, csrf string, options BaselineOptions) (ServingBaseline, error) {
	out := ServingBaseline{Digests: map[string]string{}}
	if client == nil || csrf == "" {
		return out, fmt.Errorf("admin client and CSRF required")
	}
	l, err := openLab()
	if err != nil {
		return out, err
	}
	defer l.transport.CloseIdleConnections()
	admin := &session{http: client, csrf: csrf, role: "admin"}
	commit, err := exec.CommandContext(ctx, "git", "-C", l.root, "rev-parse", "HEAD").Output()
	if err != nil {
		return out, err
	}
	specs := []struct{ version, sequence, build string }{{"0.1.0", "1", "e2e-boot"}}
	if options.Candidates {
		specs = append(specs, struct{ version, sequence, build string }{"1.0.1", "2", "e2e-good"}, struct{ version, sequence, build string }{"1.0.2", "3", "e2e-bad"})
	}
	ids := []uuid.UUID{}
	seen := map[string]bool{}
	for i, spec := range specs {
		dir, err := os.MkdirTemp(filepath.Join(l.root, ".agent", "tmp"), "e2e-release-")
		if err != nil {
			return out, err
		}
		rollback := options.Rollback
		if options.IncludeBootInRollback && i > 0 {
			rollback.AllowedReleaseIDs = RollbackAllowlist(out.Boot, rollback.AllowedReleaseIDs...)
		}
		id, raw, err := l.buildRelease(ctx, dir, spec.version, spec.sequence, strings.TrimSpace(string(commit)), spec.build, rollback)
		if err == nil {
			err = l.uploadRelease(ctx, id, dir, raw)
		}
		_ = os.RemoveAll(dir)
		if err != nil {
			return out, err
		}
		var env struct{ Signed json.RawMessage }
		var manifest api.ReleaseManifest
		if err = json.Unmarshal(raw, &env); err != nil {
			return out, err
		}
		if err = json.Unmarshal(env.Signed, &manifest); err != nil {
			return out, err
		}
		for _, artifact := range manifest.Artifacts {
			if seen[artifact.Sha256] {
				return out, fmt.Errorf("release artifacts must be distinct")
			}
			seen[artifact.Sha256] = true
			out.Digests["artifact_sha256_"+spec.version+"_"+string(artifact.Architecture)] = artifact.Sha256
		}
		out.Digests["signed_manifest_sha256_"+spec.version], err = canonjson.HashHex(env.Signed)
		if err != nil {
			return out, err
		}
		out.Digests["manifest_sha256_"+spec.version], err = canonjson.HashHex(raw)
		if err != nil {
			return out, err
		}
		reg, err := admin.call(ctx, "POST", l.public+"/api/v1/admin/releases", map[string]any{"manifest": json.RawMessage(raw), "manifest_url": l.releases + "/r/" + id.String() + "/manifest.json"}, map[string]string{"Idempotency-Key": uuid.NewString()})
		if err != nil || (reg.Status != 201 && reg.Status != 202) {
			return out, fmt.Errorf("register %s: %s", spec.version, httpDetail(reg, err))
		}
		if err = l.candidateConfig(id, spec.version); err != nil {
			return out, err
		}
		ids = append(ids, id)
		switch i {
		case 0:
			out.Boot = id
			err = l.setBootRelease(id)
		case 1:
			out.Good = id
		case 2:
			out.Bad = id
		}
		if err != nil {
			return out, err
		}
	}
	wait, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	verified := waitUntil(wait, time.Second, func() bool {
		resp, err := admin.call(wait, "GET", l.public+"/api/v1/admin/releases", nil, nil)
		var page api.ReleaseList
		if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &page) != nil {
			return false
		}
		count := 0
		for _, want := range ids {
			for _, item := range page.Items {
				if item.ReleaseId == want && item.VerificationState == "verified" {
					count++
				}
			}
		}
		return count == len(ids)
	})
	if !verified {
		return out, fmt.Errorf("release verification timed out")
	}
	out.SupportRevision, _, err = l.createPolicy(ctx, admin, "release_support", "e2e-serving", map[string]any{"approved_release_ids": ids, "revoked_release_ids": []string{}, "minimum_serving_version": map[string]any{"serving_release_ids": ids}})
	return out, err
}

func (l *lab) buildRelease(ctx context.Context, dir, product, sequence, commit, build string, options ...ReleaseOptions) (uuid.UUID, []byte, error) {
	pkg := filepath.Join(dir, "tree")
	binDir := pkg
	installerLayout := len(options) > 0 && options[0].InstallerLayout
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return uuid.Nil, nil, err
	}
	if installerLayout {
		units := filepath.Join(l.root, "deploy", "node", "systemd")
		if err := os.CopyFS(filepath.Join(pkg, "lib", "systemd"), os.DirFS(units)); err != nil {
			return uuid.Nil, nil, err
		}
	}
	metadata, _ := json.Marshal(map[string]string{"product_version": product, "source_commit": commit, "purpose": "TEST ONLY acceptance"})
	if err := os.WriteFile(filepath.Join(pkg, "release.json"), metadata, 0644); err != nil {
		return uuid.Nil, nil, err
	}
	installed := map[string]string{}
	for _, arch := range []string{"amd64", "arm64"} {
		packages := map[string]string{"agent": "./node/agent/cmd/fvpn-agent", "watchdog": "./node/watchdog/cmd/fvpn-watchdog", "updater": "./node/updater/cmd/fvpn-updater"}
		if installerLayout {
			packages["node-install"] = "./packaging/installer/cmd/fvpn-node-install"
		}
		for name, path := range packages {
			cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", filepath.Join(binDir, "fvpn-"+name), path)
			cmd.Dir = l.root
			cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0", "GOTOOLCHAIN=local")
			if err := cmd.Run(); err != nil {
				return uuid.Nil, nil, fmt.Errorf("build %s %s: %w", arch, name, err)
			}
		}
		var size uint64
		if err := filepath.Walk(pkg, func(_ string, info os.FileInfo, err error) error {
			if err == nil && info.Mode().IsRegular() {
				size += uint64(info.Size())
			}
			return err
		}); err != nil {
			return uuid.Nil, nil, err
		}
		installed[arch] = decstr.U64(size).String()
		if err := release.PackageDir(pkg, filepath.Join(dir, arch+".tar.zst"), time.Unix(1_700_000_000, 0).UTC()); err != nil {
			return uuid.Nil, nil, err
		}
	}
	return l.signedRelease(dir, product, sequence, commit, build, installed, options...)
}

// createPolicy uses only the policy API, including resource-version readback.
func (l *lab) createPolicy(ctx context.Context, admin *session, kind, code string, payload any) (uuid.UUID, uuid.UUID, error) {
	resp, err := admin.call(ctx, "POST", l.public+"/api/v1/admin/policies", map[string]any{"type": kind, "code": code, "schema_version": 1, "reason": "TEST ONLY acceptance input", "payload": payload}, map[string]string{"Idempotency-Key": uuid.NewString()})
	var revision struct {
		ID         uuid.UUID `json:"revision_id"`
		Definition uuid.UUID `json:"definition_id"`
	}
	if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &revision) != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("create %s: %s %s", kind, httpDetail(resp, err), apiErrorText(resp.Body))
	}
	list, err := admin.call(ctx, "GET", l.public+"/api/v1/admin/policies?type="+kind, nil, nil)
	var page api.PolicyDefinitionList
	if err != nil || list.Status != 200 || json.Unmarshal(list.Body, &page) != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("list %s: %s", kind, httpDetail(list, err))
	}
	version := ""
	for _, p := range page.Items {
		if p.DefinitionId == revision.Definition {
			version = p.ResourceVersion.String()
		}
	}
	if version == "" {
		return uuid.Nil, uuid.Nil, fmt.Errorf("policy missing from list")
	}
	act, err := admin.call(ctx, "POST", l.public+"/api/v1/admin/policies/"+revision.Definition.String()+"/activate", map[string]any{"revision_id": revision.ID, "reason": "TEST ONLY acceptance approval"}, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": `"` + version + `"`})
	if err != nil || act.Status != 202 {
		return uuid.Nil, uuid.Nil, fmt.Errorf("activate %s: %s %s", kind, httpDetail(act, err), apiErrorText(act.Body))
	}
	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !waitUntil(wait, time.Second, func() bool {
		list, err := admin.call(wait, "GET", l.public+"/api/v1/admin/policies?type="+kind, nil, nil)
		var page api.PolicyDefinitionList
		if err != nil || list.Status != 200 || json.Unmarshal(list.Body, &page) != nil {
			return false
		}
		for _, p := range page.Items {
			if p.DefinitionId == revision.Definition {
				active, e := p.ActiveRevisionId.Get()
				return e == nil && active == revision.ID
			}
		}
		return false
	}) {
		return uuid.Nil, uuid.Nil, fmt.Errorf("active %s revision readback timed out", kind)
	}
	return revision.ID, revision.Definition, nil
}
