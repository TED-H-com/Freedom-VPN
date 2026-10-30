package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/node/dns"
	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// DNSApplier applies one verified resolver document. *dns.Applier implements
// it. A nil applier does not construct an AdGuard client: a referenced
// document is fetched and verified, then dns_policy fails closed with
// outcome not_run and nothing is written to a resolver.
type DNSApplier interface {
	Apply(ctx context.Context, generation string, doc dnspublish.Document) (dns.Evidence, error)
}

// The production applier is the DNS owner's type. This assignment fails the
// build if that signature changes.
var _ DNSApplier = (*dns.Applier)(nil)

var feedCodeRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type resolverErr struct {
	code   api.ErrorCode
	detail string
	body   []byte
}

func (e *resolverErr) Error() string {
	if e == nil {
		return ""
	}
	return string(e.code)
}

// applyResolver fetches, verifies, and applies dns.resolver_document.
// Absent or null keeps designated resolvers only and reports no evidence.
// Size and digest are checked against the exact bytes before parse or Apply.
func (e *Engine) applyResolver(ctx context.Context) {
	e.dnsDocument = false
	e.dnsFailed = false
	e.dnsCode = ""
	e.dnsDetail = ""
	e.dnsEv = nil
	if e.revoked || !e.enrolled || e.desired == nil || !e.desiredOK {
		return
	}
	ref, err := e.desired.Dns.ResolverDocument.Get()
	if err != nil {
		return
	}
	e.dnsDocument = true
	if !resolverRefOK(ref) {
		e.noteResolver(ref, nil, api.DnsPolicyEvidenceOutcomeFailed, api.ErrorCodeVALIDATIONFAILED, "Resolver document reference is not usable.")
		return
	}
	body, rerr := e.loadResolverBytes(ctx, ref)
	if rerr != nil {
		e.noteResolver(ref, rerr.body, api.DnsPolicyEvidenceOutcomeFailed, rerr.code, rerr.detail)
		return
	}
	doc, err := parseResolver(body)
	if err != nil {
		e.noteResolver(ref, body, api.DnsPolicyEvidenceOutcomeFailed, api.ErrorCodeVALIDATIONFAILED, "Resolver document failed validation.")
		return
	}
	if e.dnsApply == nil {
		e.noteResolver(ref, body, api.DnsPolicyEvidenceOutcomeNotRun, api.ErrorCodeDEPENDENCYUNAVAILABLE, "Resolver policy did not run.")
		return
	}
	ev, aerr := e.dnsApply.Apply(ctx, ref.Generation.String(), doc)
	e.noteDNSApply(ref, ev, body, aerr)
}

func (e *Engine) loadResolverBytes(ctx context.Context, ref api.ResolverDocumentRef) ([]byte, *resolverErr) {
	if e.store != nil {
		cached, ok, err := e.store.ResolverCache(ctx, ref.DocumentId.String())
		if err != nil {
			e.storageOK = false
			return nil, &resolverErr{code: api.ErrorCodeSTOREUNAVAILABLE, detail: "Resolver document could not be stored."}
		}
		if ok && matchResolver(cached, ref) == nil {
			return cached, nil
		}
	}
	if e.client == nil {
		return nil, &resolverErr{code: api.ErrorCodeDEPENDENCYUNAVAILABLE, detail: "Resolver document could not be fetched."}
	}
	body, err := e.client.ResolverDocument(ctx, ref.DocumentId)
	if err != nil {
		var se *controlclient.StatusError
		if errors.As(err, &se) && (se.Status == http.StatusNotFound || se.Code == string(api.ErrorCodeNOTFOUND)) {
			return nil, &resolverErr{code: api.ErrorCodeNOTFOUND, detail: "Resolver document was not found."}
		}
		if strings.Contains(err.Error(), "exceeds 1 MiB") {
			return nil, &resolverErr{code: api.ErrorCodePAYLOADTOOLARGE, detail: "Resolver document exceeds the size limit."}
		}
		return nil, &resolverErr{code: api.ErrorCodeDEPENDENCYUNAVAILABLE, detail: "Resolver document could not be fetched."}
	}
	if merr := matchResolver(body, ref); merr != nil {
		return nil, merr
	}
	if e.store != nil {
		if err := e.store.PutResolverCache(ctx, ref.DocumentId.String(), ref.ContentSha256, body); err != nil {
			e.storageOK = false
			return nil, &resolverErr{code: api.ErrorCodeSTOREUNAVAILABLE, detail: "Resolver document could not be stored.", body: body}
		}
	}
	return body, nil
}

// matchResolver checks the signed length and SHA-256 of the exact bytes.
// It does not parse them. A longer body is an overflow; a shorter body is
// a size mismatch. Either one refuses Apply.
func matchResolver(body []byte, ref api.ResolverDocumentRef) *resolverErr {
	if len(body) > canonjson.MaxBytes {
		return &resolverErr{code: api.ErrorCodePAYLOADTOOLARGE, detail: "Resolver document exceeds the size limit.", body: body}
	}
	if len(body) != ref.SizeBytes {
		if len(body) > ref.SizeBytes {
			return &resolverErr{code: api.ErrorCodePAYLOADTOOLARGE, detail: "Resolver document exceeds the signed size.", body: body}
		}
		return &resolverErr{code: api.ErrorCodeSIZEMISMATCH, detail: "Resolver document size differs from the signed reference.", body: body}
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != ref.ContentSha256 {
		return &resolverErr{code: api.ErrorCodeHASHMISMATCH, detail: "Resolver document digest differs from the signed reference.", body: body}
	}
	return nil
}

func parseResolver(body []byte) (dnspublish.Document, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var doc dnspublish.Document
	if err := dec.Decode(&doc); err != nil {
		return dnspublish.Document{}, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return dnspublish.Document{}, errors.New("agent: trailing resolver document")
	}
	if err := dnspublish.Verify(doc); err != nil {
		return dnspublish.Document{}, err
	}
	return doc, nil
}

func resolverRefOK(ref api.ResolverDocumentRef) bool {
	if ref.DocumentId == uuid.Nil || ref.ProfileRevisionId == uuid.Nil {
		return false
	}
	if ref.SizeBytes < 1 || ref.SizeBytes > canonjson.MaxBytes || !shaOK(ref.ContentSha256) {
		return false
	}
	_, err := decstr.Parse(ref.Generation.String())
	return err == nil
}

func (e *Engine) noteResolver(ref api.ResolverDocumentRef, body []byte, outcome api.DnsPolicyEvidenceOutcome, code api.ErrorCode, detail string) {
	e.dnsFailed = true
	e.dnsCode = code
	e.dnsDetail = detail
	e.dnsEv = &api.DnsPolicyEvidence{
		AppliedAt:     presence.NewNullNullable[time.Time](),
		ContentSha256: heldHash(body, ref),
		DocumentId:    ref.DocumentId,
		ErrorCode:     presence.NewNullableWithValue(code),
		Feeds:         []api.DnsFeedEvidence{},
		Generation:    ref.Generation,
		Outcome:       outcome,
	}
}

func (e *Engine) noteDNSApply(ref api.ResolverDocumentRef, ev dns.Evidence, body []byte, err error) {
	outcome := classifyDNS(ev, err)
	code := api.ErrorCode("")
	detail := ""
	switch outcome {
	case api.DnsPolicyEvidenceOutcomeApplied:
	case api.DnsPolicyEvidenceOutcomePartial:
		code = api.ErrorCodeROLLBACKFAILED
		detail = "Resolver policy was only partly applied."
	case api.DnsPolicyEvidenceOutcomeNotRun:
		code = api.ErrorCodeDEPENDENCYUNAVAILABLE
		detail = "Resolver policy did not run."
	default:
		outcome = api.DnsPolicyEvidenceOutcomeFailed
		code = api.ErrorCodeVALIDATIONFAILED
		detail = "Resolver policy was not applied."
	}
	e.dnsFailed = outcome != api.DnsPolicyEvidenceOutcomeApplied
	e.dnsCode = code
	e.dnsDetail = detail
	hash := strings.ToLower(strings.TrimSpace(ev.DocumentHash))
	if !shaOK(hash) {
		hash = heldHash(body, ref)
	}
	gen := ref.Generation
	if g, gerr := decstr.Parse(strings.TrimSpace(ev.PolicyGeneration)); gerr == nil {
		gen = g
	}
	at := presence.NewNullNullable[time.Time]()
	errN := presence.NewNullNullable[api.ErrorCode]()
	if outcome == api.DnsPolicyEvidenceOutcomeApplied && !ev.AppliedAt.IsZero() {
		at = presence.NewNullableWithValue(ev.AppliedAt.UTC())
	}
	if outcome != api.DnsPolicyEvidenceOutcomeApplied && code != "" {
		errN = presence.NewNullableWithValue(code)
	}
	e.dnsEv = &api.DnsPolicyEvidence{
		AppliedAt:     at,
		ContentSha256: hash,
		DocumentId:    ref.DocumentId,
		ErrorCode:     errN,
		Feeds:         mapFeeds(ev.Feeds),
		Generation:    gen,
		Outcome:       outcome,
	}
}

// heldHash is the digest of the bytes this attempt held. When the fetch
// returned nothing, it is the signed digest of the issued reference. The
// outcome is failed or not_run in that case, so the digest is not an apply
// claim. It is never taken from a newer desired state.
func heldHash(body []byte, ref api.ResolverDocumentRef) string {
	if len(body) > 0 && len(body) <= canonjson.MaxBytes {
		sum := sha256.Sum256(body)
		return hex.EncodeToString(sum[:])
	}
	if shaOK(ref.ContentSha256) {
		return ref.ContentSha256
	}
	return strings.Repeat("0", 64)
}

func classifyDNS(ev dns.Evidence, err error) api.DnsPolicyEvidenceOutcome {
	if err == nil && ev.DNSOK() && shaOK(ev.DocumentHash) {
		if _, gerr := decstr.Parse(strings.TrimSpace(ev.PolicyGeneration)); gerr == nil {
			return api.DnsPolicyEvidenceOutcomeApplied
		}
	}
	// node/dns reports a rollback that did not restore every resolver with
	// this sentence. The sentinel is unexported. That is a partial apply.
	if err != nil && strings.Contains(err.Error(), "rollback did not restore") {
		return api.DnsPolicyEvidenceOutcomePartial
	}
	if err == nil {
		return api.DnsPolicyEvidenceOutcomeNotRun
	}
	return api.DnsPolicyEvidenceOutcomeFailed
}

func mapFeeds(in []dns.FeedEvidence) []api.DnsFeedEvidence {
	out := make([]api.DnsFeedEvidence, 0, len(in))
	seen := map[string]struct{}{}
	for _, f := range in {
		code := f.FeedCode
		hash := strings.ToLower(f.ContentSHA256)
		if !feedCodeRE.MatchString(code) || !shaOK(hash) || f.RuleCount < 0 {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, api.DnsFeedEvidence{ContentSha256: hash, FeedCode: code, RuleCount: f.RuleCount})
		if len(out) == 32 {
			break
		}
	}
	return out
}

func shaOK(h string) bool {
	if len(h) != 64 {
		return false
	}
	for i := 0; i < len(h); i++ {
		c := h[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}
