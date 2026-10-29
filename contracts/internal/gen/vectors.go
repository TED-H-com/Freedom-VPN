package gen

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func canonicalize(b []byte) ([]byte, error) { return canonjson.Canonicalize(b) }

// Reason describes why an invalid example must fail.
type Reason struct {
	// Kind is "schema", "payload_hash" or "signature".
	Kind string `json:"kind"`
	// Pointer and Keyword identify the failing schema location (kind=schema).
	Pointer string `json:"pointer,omitempty"`
	Keyword string `json:"keyword,omitempty"`
	// Field names the hash member (kind=payload_hash).
	Field string `json:"field,omitempty"`
	// Code is KEY_UNTRUSTED or SIGNATURE_INVALID (kind=signature).
	Code string `json:"code,omitempty"`
	// Note explains the case.
	Note string `json:"note"`
}

// Files returns every generated file keyed by path relative to the
// contracts directory.
func Files(openapiYAML []byte) (map[string][]byte, error) {
	out := map[string][]byte{}
	schemas, err := JSONSchemas(openapiYAML)
	if err != nil {
		return nil, err
	}
	for name, b := range schemas {
		out["schemas/"+name] = b
	}
	v, err := vectorsAndExamples()
	if err != nil {
		return nil, err
	}
	for k, b := range v {
		out[k] = b
	}
	return out, nil
}

func pretty(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// prettyDoc re-indents canonical JSON (keys stay sorted).
func prettyDoc(canon []byte) []byte {
	var buf bytes.Buffer
	if err := json.Indent(&buf, canon, "", "  "); err != nil {
		panic(err)
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}

func sign(doc []byte, k TestKey) envelope.Envelope {
	env, err := envelope.Sign(json.RawMessage(doc), k.priv, k.KeyID)
	if err != nil {
		panic(err)
	}
	return env
}

func envJSON(env envelope.Envelope) []byte {
	b, err := env.MarshalCanonical()
	if err != nil {
		panic(err)
	}
	return b
}

func withHash(doc []byte, field string) []byte {
	h, err := canonjson.HashHex(doc, field)
	if err != nil {
		panic(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(doc, &m); err != nil {
		panic(err)
	}
	m[field] = json.RawMessage(`"` + h + `"`)
	b, _ := json.Marshal(m)
	return mustCanon(string(b))
}

func setMember(doc []byte, path []string, value string) []byte {
	var v any
	if err := json.Unmarshal(doc, &v); err != nil {
		panic(err)
	}
	cur := v
	for i, p := range path {
		m := cur.(map[string]any)
		if i == len(path)-1 {
			var nv any
			if err := json.Unmarshal([]byte(value), &nv); err != nil {
				panic(err)
			}
			m[p] = nv
			break
		}
		next, ok := m[p]
		if !ok {
			panic("setMember: missing " + p)
		}
		if arr, ok := next.([]any); ok {
			cur = arr[0]
		} else {
			cur = next
		}
	}
	b, _ := json.Marshal(v)
	return mustCanon(string(b))
}

func signingVector(desc, purpose string, k TestKey, signed []byte, env envelope.Envelope) []byte {
	return pretty(map[string]any{
		"_comment":                TestOnlyLabel + ". Deterministic Ed25519 vector; the seed is public test data.",
		"description":             desc,
		"purpose":                 purpose,
		"key_id":                  k.KeyID,
		"seed_hex":                k.SeedHex,
		"public_key_b64":          k.PubB64,
		"verify_at":               TestNow,
		"signed":                  rawJSON(signed),
		"canonical_signed":        string(signed),
		"canonical_signed_sha256": hexSHA(signed),
		"signature_b64":           env.Signatures[0].SignatureB64,
		"envelope":                rawJSON(envJSON(env)),
		"envelope_sha256":         hexSHA(envJSON(env)),
	})
}

func vectorsAndExamples() (map[string][]byte, error) {
	keys := TestKeys()
	out := map[string][]byte{}
	reasons := map[string]Reason{}

	// ---- test keys
	labels := make([]string, 0, len(keys))
	for l := range keys {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	var keyList []TestKey
	for _, l := range labels {
		keyList = append(keyList, keys[l])
	}
	out["vectors/test-keys.json"] = pretty(map[string]any{
		"_comment":   TestOnlyLabel + ". seed_hex = SHA-256 of the UTF-8 string 'TEST ONLY - NOT A SECRET: fvpn contracts <label> test key'. Keys with trusted=true form the test trust store; never trust them outside tests.",
		"not_before": TestKeyNotBefore,
		"not_after":  TestKeyNotAfter,
		"verify_at":  TestNow,
		"keys":       keyList,
	})

	// ---- release manifest
	manifest := mustCanon(releaseManifest)
	manEnv := sign(manifest, keys["release"])
	manEnvJSON := envJSON(manEnv)
	manifestSHA := hexSHA(manEnvJSON)
	out["vectors/release-manifest-signing.json"] = signingVector(
		"Release manifest envelope (§13.3). manifest_sha256 elsewhere is envelope_sha256: SHA-256 of the canonical JSON of the whole envelope.",
		"release", keys["release"], manifest, manEnv)
	out["examples/valid/ReleaseManifestEnvelope.json"] = prettyDoc(manEnvJSON)
	out["examples/valid/ReleaseRegisterRequest.json"] = prettyDoc(mustCanon(`{"manifest":` + string(manEnvJSON) +
		`,"manifest_url":"https://releases.example.invalid/r42/manifest.json"}`))
	tampered := envelope.Envelope{Signed: setMember(manifest, []string{"product_version"}, `"1.1.1"`), Signatures: manEnv.Signatures}
	out["examples/invalid/ReleaseManifestEnvelope.tampered.json"] = prettyDoc(envJSON(tampered))
	reasons["ReleaseManifestEnvelope.tampered.json"] = Reason{Kind: "signature", Code: envelope.CodeSignatureInvalid,
		Note: "product_version changed after signing; the trusted key's signature no longer verifies."}

	// Semantic-invalid fixtures keep valid signatures to isolate the service rule.
	twoAMD := bytes.Replace(manifest, []byte(`"architecture":"arm64"`), []byte(`"architecture":"amd64"`), 1)
	twoAMDEnv := envJSON(sign(twoAMD, keys["release"]))
	out["examples/semantic-invalid/ReleaseRegisterRequest.two-amd64.json"] = prettyDoc(mustCanon(`{"manifest":` + string(twoAMDEnv) + `,"manifest_url":"https://releases.example.invalid/r42/manifest.json"}`))

	// ---- release trust update: the old release root authorizes a new public key.
	trustUpdate := mustCanon(fmt.Sprintf(`{"kind":"fvpn-release-trust-v1","revision":"9007199254740993","keys":[{"id":%q,"purpose":"release","public":%q,"state":"active","not_before":%q,"not_after":%q}]}`,
		keys["untrusted"].KeyID, keys["untrusted"].PubB64, TestKeyNotBefore, TestKeyNotAfter))
	trustEnv := sign(trustUpdate, keys["release"])
	out["vectors/release-trust-update-signing.json"] = signingVector("ReleaseUpdate authorized by the OLD release key (§13.4); the previously untrusted test public key becomes authorized only after verification.", "release", keys["release"], trustUpdate, trustEnv)
	out["examples/trust/valid/release-update.json"] = prettyDoc(envJSON(trustEnv))
	for _, tc := range []struct {
		name  string
		path  []string
		value string
	}{
		{"wrong-kind", []string{"kind"}, `"fvpn-unknown-trust-v1"`},
		{"numeric-revision", []string{"revision"}, `2`},
		{"wrong-purpose", []string{"keys", "purpose"}, `"desired-state"`},
		{"private-material", []string{"keys", "private"}, `"TEST ONLY - NOT A SECRET"`},
		{"revoked-without-time", []string{"keys", "state"}, `"revoked"`},
		{"bad-public-key", []string{"keys", "public"}, `"AA=="`},
	} {
		out["examples/trust/invalid/"+tc.name+".json"] = prettyDoc(envJSON(sign(setMember(trustUpdate, tc.path, tc.value), keys["release"])))
	}

	// ---- desired state
	ds := mustCanon(strings.ReplaceAll(desiredState, "%MANIFEST_SHA256%", manifestSHA))
	dsEnv := sign(ds, keys["desired-state"])
	out["vectors/desired-state-signing.json"] = signingVector(
		"Signed desired state (§14.7) including the update intent (§13.10.2).",
		"desired-state", keys["desired-state"], ds, dsEnv)
	out["examples/valid/DesiredStateEnvelope.json"] = prettyDoc(envJSON(dsEnv))
	out["examples/invalid/DesiredStateEnvelope.untrusted-key.json"] = prettyDoc(envJSON(sign(ds, keys["untrusted"])))
	reasons["DesiredStateEnvelope.untrusted-key.json"] = Reason{Kind: "signature", Code: envelope.CodeKeyUntrusted,
		Note: "Valid Ed25519 signature by a key that is not in the trust store."}
	noSig := envelope.Envelope{Signed: ds, Signatures: []envelope.Signature{}}
	out["examples/invalid/DesiredStateEnvelope.no-signatures.json"] = prettyDoc(envJSON(noSig))
	reasons["DesiredStateEnvelope.no-signatures.json"] = Reason{Kind: "schema", Pointer: "/signatures", Keyword: "minItems",
		Note: "An unsigned envelope is a downgrade and is rejected (§16.2)."}

	// D-034 adds a signed revocation set without rewriting the original vector.
	revoked := `["00000000-0000-4000-8000-000000000042"]`
	dsRevoked := setMember(ds, []string{"revoked_release_ids"}, revoked)
	dsRevokedEnv := sign(dsRevoked, keys["desired-state"])
	out["vectors/desired-state-revocations-signing.json"] = signingVector(
		"Signed desired state with explicit revoked releases (§13.4, D-034).",
		"desired-state", keys["desired-state"], dsRevoked, dsRevokedEnv)
	out["examples/valid/DesiredStateEnvelope.revocations.json"] = prettyDoc(envJSON(dsRevokedEnv))
	out["examples/valid/DesiredStateEnvelope.empty-revocations.json"] = prettyDoc(envJSON(sign(setMember(ds, []string{"revoked_release_ids"}, `[]`), keys["desired-state"])))
	for _, tc := range []struct{ name, value, pointer, keyword string }{
		{"duplicate-revocations", `["00000000-0000-4000-8000-000000000042","00000000-0000-4000-8000-000000000042"]`, "/signed/revoked_release_ids", "uniqueItems"},
		{"invalid-revocation-id", `["release-demo"]`, "/signed/revoked_release_ids/0", "format"},
		{"null-revocations", `null`, "/signed/revoked_release_ids", "type"},
	} {
		name := "DesiredStateEnvelope." + tc.name + ".json"
		out["examples/invalid/"+name] = prettyDoc(envJSON(sign(setMember(ds, []string{"revoked_release_ids"}, tc.value), keys["desired-state"])))
		reasons[name] = Reason{Kind: "schema", Pointer: tc.pointer, Keyword: tc.keyword, Note: "Revocations must be a unique UUID array when present."}
	}
	revocationTamper := envelope.Envelope{Signed: setMember(dsRevoked, []string{"revoked_release_ids"}, `[]`), Signatures: dsRevokedEnv.Signatures}
	out["examples/invalid/DesiredStateEnvelope.revocations-tampered.json"] = prettyDoc(envJSON(revocationTamper))
	reasons["DesiredStateEnvelope.revocations-tampered.json"] = Reason{Kind: "signature", Code: envelope.CodeSignatureInvalid, Note: "Removing a signed revocation invalidates the signature."}

	// ---- serving lease
	lease := mustCanon(servingLease)
	leaseEnv := sign(lease, keys["lease"])
	out["vectors/serving-lease-signing.json"] = signingVector("Serving lease (§16.3).", "lease", keys["lease"], lease, leaseEnv)
	out["examples/valid/ServingLeaseEnvelope.json"] = prettyDoc(envJSON(leaseEnv))
	out["examples/invalid/ServingLeaseEnvelope.wrong-purpose.json"] = prettyDoc(envJSON(sign(lease, keys["desired-state"])))
	reasons["ServingLeaseEnvelope.wrong-purpose.json"] = Reason{Kind: "signature", Code: envelope.CodeKeyUntrusted,
		Note: "Signed by the trusted desired-state key, which is not allowed for the lease purpose."}

	// ---- usage checkpoint
	cp := mustCanon(usageCheckpoint)
	cpEnv := sign(cp, keys["recovery"])
	out["vectors/usage-checkpoint-signing.json"] = signingVector("Usage checkpoint after a central restore (§17.3).", "recovery", keys["recovery"], cp, cpEnv)
	out["examples/valid/UsageCheckpointEnvelope.json"] = prettyDoc(envJSON(cpEnv))
	badAlg := envelope.Envelope{Signed: cp, Signatures: []envelope.Signature{{KeyID: cpEnv.Signatures[0].KeyID, Algorithm: "ecdsa-p256", SignatureB64: cpEnv.Signatures[0].SignatureB64}}}
	out["examples/invalid/UsageCheckpointEnvelope.bad-algorithm.json"] = prettyDoc(envJSON(badAlg))
	reasons["UsageCheckpointEnvelope.bad-algorithm.json"] = Reason{Kind: "schema", Pointer: "/signatures/0/algorithm", Keyword: "enum",
		Note: "Unknown signature algorithms are rejected."}

	// ---- usage events
	ev := mustCanon(usageEvent)
	evHashed := withHash(ev, "payload_sha256")
	evGap := withHash(mustCanon(usageEventGap), "payload_sha256")
	h, _ := canonjson.HashHex(evHashed, "payload_sha256")
	canonNoHash, _ := canonjson.CanonicalizeWithout(evHashed, "payload_sha256")
	out["vectors/usage-event-payload-sha256.json"] = pretty(map[string]any{
		"description":            "Usage event payload_sha256 (§10.5): SHA-256 hex of the RFC 8785 canonical JSON of the event without the payload_sha256 member. Record 2 carries 2^53+1 bytes to prove no float conversion.",
		"event_without_hash":     rawJSON(ev),
		"canonical_without_hash": string(canonNoHash),
		"payload_sha256":         h,
		"event":                  rawJSON(evHashed),
	})
	out["examples/valid/UsageEvent.json"] = prettyDoc(evHashed)
	out["examples/valid/UsageEvent.gap.json"] = prettyDoc(evGap)
	out["examples/valid/UsageBatchRequest.json"] = prettyDoc(mustCanon(`{"node_id":"` + idNode + `","node_incarnation_id":"` + idIncarnation +
		`","ledger_epoch":"` + idLedgerEpoch + `","events":[` + string(evHashed) + `,` + string(evGap) + `]}`))
	out["examples/invalid/UsageEvent.hash-mismatch.json"] = prettyDoc(setMember(evHashed, []string{"records", "upload_bytes"}, `"123457"`))
	reasons["UsageEvent.hash-mismatch.json"] = Reason{Kind: "payload_hash", Field: "payload_sha256",
		Note: "upload_bytes changed without recomputing payload_sha256."}
	out["examples/invalid/UsageEvent.numeric-bytes.json"] = prettyDoc(setMember(evHashed, []string{"records", "upload_bytes"}, `123456`))
	reasons["UsageEvent.numeric-bytes.json"] = Reason{Kind: "schema", Pointer: "/records/0/upload_bytes", Keyword: "type",
		Note: "Bytes must be a decimal string, never a JSON number (§10.4)."}

	// Usage semantic-invalid examples keep correct hashes.
	gapOnComplete := setMember(evGap, []string{"quality"}, `"complete"`)
	out["examples/semantic-invalid/UsageEvent.gap-on-complete.json"] = prettyDoc(withHash(gapOnComplete, "payload_sha256"))
	inverted := setMember(evHashed, []string{"interval_end"}, `"2026-10-28T13:59:59Z"`)
	out["examples/semantic-invalid/UsageEvent.inverted-interval.json"] = prettyDoc(withHash(inverted, "payload_sha256"))

	// ---- update results
	ur := withHash(mustCanon(strings.ReplaceAll(updateResult, "%MANIFEST_SHA256%", manifestSHA)), "payload_hash")
	urRB := withHash(mustCanon(strings.ReplaceAll(updateResultRolledBack, "%MANIFEST_SHA256%", manifestSHA)), "payload_hash")
	urNoHash, _ := canonjson.CanonicalizeWithout(ur, "payload_hash")
	urH, _ := canonjson.HashHex(ur, "payload_hash")
	out["vectors/update-result-payload-hash.json"] = pretty(map[string]any{
		"description":            "Update result payload_hash (§13.10.3): SHA-256 hex of the RFC 8785 canonical JSON of the result without the payload_hash member.",
		"canonical_without_hash": string(urNoHash),
		"payload_hash":           urH,
		"result":                 rawJSON(ur),
	})
	out["examples/valid/UpdateResult.json"] = prettyDoc(ur)
	out["examples/valid/UpdateResult.rolled-back.json"] = prettyDoc(urRB)
	out["examples/valid/UpdateResult.null-stage.json"] = prettyDoc(withHash(setMember(ur, []string{"stage"}, "null"), "payload_hash"))
	out["examples/invalid/UpdateResult.hash-mismatch.json"] = prettyDoc(setMember(ur, []string{"state"}, `"failed"`))
	reasons["UpdateResult.hash-mismatch.json"] = Reason{Kind: "payload_hash", Field: "payload_hash",
		Note: "state changed without recomputing payload_hash; the center answers ACK_CONFLICT for a known result_id."}

	// The existing required policy_generation is hashed and remains a decimal string.
	urPolicy := setMember(ur, []string{"policy_generation"}, `"9007199254740993"`)
	urPolicy = setMember(urPolicy, []string{"state"}, `"failed"`)
	urPolicy = withHash(setMember(urPolicy, []string{"error_code"}, `"ARTIFACT_INVALID"`), "payload_hash")
	urPolicyNoHash, _ := canonjson.CanonicalizeWithout(urPolicy, "payload_hash")
	urPolicyHash, _ := canonjson.HashHex(urPolicy, "payload_hash")
	out["vectors/update-result-policy-generation-hash.json"] = pretty(map[string]any{
		"description":            "Applied configuration generation above 2^53, distinct from job generation; ARTIFACT_INVALID is a node-reported code (§13.10.3).",
		"canonical_without_hash": string(urPolicyNoHash), "payload_hash": urPolicyHash, "result": rawJSON(urPolicy),
	})
	out["examples/valid/UpdateResult.applied-policy-generation.json"] = prettyDoc(urPolicy)
	out["examples/invalid/UpdateResult.numeric-policy-generation.json"] = prettyDoc(setMember(ur, []string{"policy_generation"}, `18`))
	reasons["UpdateResult.numeric-policy-generation.json"] = Reason{Kind: "schema", Pointer: "/policy_generation", Keyword: "type", Note: "Applied policy generation must never pass through a JSON number."}
	out["examples/invalid/UpdateResult.policy-generation-tampered.json"] = prettyDoc(setMember(urPolicy, []string{"policy_generation"}, `"9007199254740994"`))
	reasons["UpdateResult.policy-generation-tampered.json"] = Reason{Kind: "payload_hash", Field: "payload_hash", Note: "Changing the applied policy generation changes the payload hash."}

	// ---- canonical JSON and decimal string vectors
	cj, err := canonicalJSONVectors()
	if err != nil {
		return nil, err
	}
	out["vectors/canonical-json.json"] = cj
	out["vectors/decimal-strings.json"] = decimalStringVectors()

	out["examples/invalid/reasons.generated.json"] = pretty(reasons)
	return out, nil
}

type cjCase struct {
	Name  string
	Input []byte
}

type cjReject struct {
	Name  string
	Input []byte
	Error string
}

var cjCases = []cjCase{
	{"depth-256", []byte(strings.Repeat("[", 256) + "0" + strings.Repeat("]", 256))},
	{"utf16-differ-only-past-bmp", []byte(`{"prefix\ud801\udc00":3,"prefix\ud800\udc01":2,"prefix\ud800\udc00":1}`)},
	{"rfc8785-3.2.2-without-numbers", []byte(`{"string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/", "literals": [null, true, false]}`)},
	{"rfc8785-3.2.3-sorting", []byte(`{"\u20ac": "Euro Sign", "\r": "Carriage Return", "\ufb33": "Hebrew Letter Dalet With Dagesh", "1": "One", "\ud83d\ude00": "Emoji: Grinning Face", "\u0080": "Control", "\u00f6": "Latin Small Letter O With Diaeresis"}`)},
	{"rfc8785-arrays", []byte(`[56, {"d": true, "10": null, "1": [ ]}]`)},
	{"rfc8785-structures-integer", []byte(`{"1": {"f": {"f": "hi","F": 5} ,"\n": 56},"10": { },"": "empty","a": { },"111": [ {"e": "yes","E": "no" } ],"A": { }}`)},
	{"rfc8785-unicode-no-normalization", []byte(`{"Unnormalized Unicode":"A\u030a"}`)},
	{"rfc8785-weird", []byte(`{"\u20ac": "Euro Sign", "\r": "Carriage Return", "\u000a": "Newline", "1": "One", "\u0080": "Control\u007f", "\ud83d\ude02": "Smiley", "\u00f6": "Latin Small Letter O With Diaeresis", "\ufb33": "Hebrew Letter Dalet With Dagesh", "</script>": "Browser Challenge"}`)},
	{"integers-at-limits", []byte(`[0, 1, -1, 9007199254740991, -9007199254740991]`)},
	{"control-escapes", []byte(`"\u0000\u0001\u0008\u0009\u000a\u000c\u000d\u001f\u007f"`)},
	{"no-html-or-solidus-escaping", []byte(`"<a href=\"x\">&amp;</a> \/ \u2028\u2029"`)},
	{"decimal-strings-stay-strings", []byte(`{"upload_bytes": "9007199254740993", "event_seq": "18446744073709551615"}`)},
	{"nested-sorting-and-whitespace", []byte("{\n  \"b\": [ {\"z\": 1, \"a\": 2} ],\n\t\"a\": { \"y\": null, \"x\": true }\r\n}")},
	{"literal-utf8-passthrough", []byte(`{"名稱":"節點","émoji":"😀"}`)},
}

var cjRejects = []cjReject{
	{"nested-duplicate-key", []byte(`{"outer":[{"a":1,"a":2}]}`), "duplicate_key"},
	{"depth-257", []byte(strings.Repeat("[", 257) + "0" + strings.Repeat("]", 257)), "too_deep"},
	{"leading-plus", []byte(`+1`), "syntax"},
	{"positive-exponent", []byte(`1e+3`), "non_integer"},
	{"fraction", []byte(`{"a":1.0}`), "non_integer"},
	{"exponent", []byte(`[1e3]`), "non_integer"},
	{"negative-zero", []byte(`-0`), "negative_zero"},
	{"above-2^53-1", []byte(`9007199254740992`), "integer_range"},
	{"below--(2^53-1)", []byte(`-9007199254740992`), "integer_range"},
	{"duplicate-key", []byte(`{"a":1,"a":2}`), "duplicate_key"},
	{"duplicate-key-after-unescape", []byte(`{"a":1,"\u0061":2}`), "duplicate_key"},
	{"invalid-utf8", []byte("{\"a\":\"\xff\"}"), "invalid_utf8"},
	{"utf8-encoded-surrogate", []byte("\"\xed\xa0\x80\""), "invalid_utf8"},
	{"lone-high-surrogate", []byte(`"\ud800"`), "lone_surrogate"},
	{"lone-low-surrogate", []byte(`"\udc00"`), "lone_surrogate"},
	{"leading-zero", []byte(`01`), "syntax"},
	{"trailing-comma", []byte(`[1,]`), "syntax"},
	{"raw-control-character", []byte("\"a\tb\""), "syntax"},
	{"byte-order-mark", []byte("\xef\xbb\xbf{}"), "syntax"},
	{"trailing-data", []byte(`{} {}`), "syntax"},
}

func canonicalJSONVectors() ([]byte, error) {
	type caseOut struct {
		Name      string `json:"name"`
		InputB64  string `json:"input_b64"`
		Input     string `json:"input"`
		Canonical string `json:"canonical"`
		SHA256    string `json:"sha256"`
	}
	type rejectOut struct {
		Name     string `json:"name"`
		InputB64 string `json:"input_b64"`
		Error    string `json:"error"`
	}
	var cases []caseOut
	for _, c := range cjCases {
		canon, err := canonjson.Canonicalize(c.Input)
		if err != nil {
			return nil, fmt.Errorf("canonical vector %s: %w", c.Name, err)
		}
		cases = append(cases, caseOut{c.Name, base64.StdEncoding.EncodeToString(c.Input), string(c.Input), string(canon), hexSHA(canon)})
	}
	var rejects []rejectOut
	for _, r := range cjRejects {
		if _, err := canonjson.Canonicalize(r.Input); err == nil {
			return nil, fmt.Errorf("reject vector %s was accepted", r.Name)
		} else if got := ErrorClass(err); got != r.Error {
			return nil, fmt.Errorf("reject vector %s: class %s, want %s", r.Name, got, r.Error)
		}
		rejects = append(rejects, rejectOut{r.Name, base64.StdEncoding.EncodeToString(r.Input), r.Error})
	}
	return pretty(map[string]any{
		"description": "RFC 8785 (JCS) vectors for the Freedom VPN JSON subset (objects, arrays, strings, true, false, null, integers within ±(2^53-1)). input_b64 holds the exact input bytes; canonical is the UTF-8 canonical form; sha256 is its lowercase hex SHA-256. Rejects must fail with the given error class.",
		"error_classes": map[string]string{
			"too_deep": "nesting exceeds 256",
			"syntax":   "invalid JSON", "non_integer": "fraction or exponent", "negative_zero": "-0",
			"integer_range": "outside ±(2^53-1)", "duplicate_key": "duplicate member name after unescaping",
			"invalid_utf8": "invalid UTF-8 bytes", "lone_surrogate": "unpaired UTF-16 surrogate escape",
		},
		"cases":   cases,
		"rejects": rejects,
	}), nil
}

// ErrorClass maps a canonjson error to its vector error class.
func ErrorClass(err error) string {
	switch {
	case isErr(err, canonjson.ErrTooDeep):
		return "too_deep"
	case isErr(err, canonjson.ErrNonInteger):
		return "non_integer"
	case isErr(err, canonjson.ErrNegativeZero):
		return "negative_zero"
	case isErr(err, canonjson.ErrIntegerRange):
		return "integer_range"
	case isErr(err, canonjson.ErrDuplicateKey):
		return "duplicate_key"
	case isErr(err, canonjson.ErrInvalidUTF8):
		return "invalid_utf8"
	case isErr(err, canonjson.ErrLoneSurrogate):
		return "lone_surrogate"
	case isErr(err, canonjson.ErrSyntax):
		return "syntax"
	}
	return "other"
}

// DecimalAccept and DecimalReject are the decimal string vectors (§10.4).
var DecimalAccept = []string{"0", "1", "9", "10", "42", "9007199254740991", "9007199254740992", "9007199254740993",
	"9223372036854775807", "9223372036854775808", "18446744073709551614", "18446744073709551615"}

// DecimalReject entries hold the raw JSON value text and the reason class.
var DecimalReject = []struct{ JSON, Reason string }{
	{`""`, "empty"},
	{`"-0"`, "sign"},
	{`"-1"`, "sign"},
	{`"+1"`, "sign"},
	{`"01"`, "leading_zero"},
	{`"00"`, "leading_zero"},
	{`" 1"`, "whitespace"},
	{`"1 "`, "whitespace"},
	{`"1e3"`, "exponent"},
	{`"1E3"`, "exponent"},
	{`"1.0"`, "fraction"},
	{`"0x10"`, "non_digit"},
	{`"NaN"`, "non_digit"},
	{`"1_000"`, "non_digit"},
	{`"\uff11"`, "non_digit"},
	{`"18446744073709551616"`, "out_of_range"},
	{`"99999999999999999999"`, "out_of_range"},
	{`"100000000000000000000"`, "out_of_range"},
	{`123`, "not_a_string"},
	{`9007199254740993`, "not_a_string"},
	{`null`, "not_a_string"},
	{`true`, "not_a_string"},
}

func decimalStringVectors() []byte {
	type rej struct {
		JSON   string `json:"json"`
		Reason string `json:"reason"`
	}
	var rejects []rej
	for _, r := range DecimalReject {
		rejects = append(rejects, rej{r.JSON, r.Reason})
	}
	return pretty(map[string]any{
		"description": "JSON decimal strings for unsigned 64-bit integers (§10.4, §14.1). accept: string values that must parse to exactly the same digits. reject: raw JSON value texts that must be rejected. Pattern ^(0|[1-9][0-9]*)$, at most 20 digits, at most 18446744073709551615.",
		"pattern":     "^(0|[1-9][0-9]*)$",
		"max":         "18446744073709551615",
		"accept":      DecimalAccept,
		"reject":      rejects,
	})
}

func isErr(err, target error) bool { return errors.Is(err, target) }
