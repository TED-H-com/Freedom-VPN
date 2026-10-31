// Cross-language check of the shared contract vectors (contracts/vectors).
//
// Run with Node 24 or newer, which executes TypeScript by stripping types:
//
//   node contracts/ts/verify-vectors.ts
//
// It re-implements, independently of the Go code, the parts a TypeScript
// client needs: RFC 8785 canonical JSON for the Freedom VPN subset, the
// usage-event payload_sha256 and update-result payload_hash, Ed25519 envelope
// verification and decimal-string validation. Any mismatch exits non-zero.
// Copy these helpers (or this approach) into TypeScript code that hashes or
// verifies documents; see contracts/vectors/README.md.

import { createHash, createPublicKey, verify } from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { parseStrictJSON, type Json } from "./strict-json.ts";

const vectorsDir = join(dirname(fileURLToPath(import.meta.url)), "..", "vectors");
const load = (name: string): any => JSON.parse(readFileSync(join(vectorsDir, name), "utf8"));

let failures = 0;
let checks = 0;
function check(ok: boolean, what: string): void {
  checks++;
  if (!ok) {
    failures++;
    console.error("FAIL:", what);
  }
}

// RFC 8785 for the subset: integers within ±(2^53-1), no floats.
// JavaScript's default sort compares UTF-16 code units, exactly as JCS
// requires, and JSON.stringify escapes strings exactly as JCS requires.
export function canonicalize(v: Json): string {
  if (v === null || typeof v === "boolean") return JSON.stringify(v);
  if (typeof v === "number") {
    if (!Number.isSafeInteger(v) || Object.is(v, -0)) throw new Error("number outside the integer subset");
    return String(v);
  }
  if (typeof v === "string") return JSON.stringify(v);
  if (Array.isArray(v)) return "[" + v.map(canonicalize).join(",") + "]";
  const keys = Object.keys(v).sort();
  return "{" + keys.map((k) => JSON.stringify(k) + ":" + canonicalize(v[k])).join(",") + "}";
}

const sha256hex = (s: string | Buffer): string => createHash("sha256").update(s).digest("hex");

// Hash of the canonical form without the named top-level members.
export function hashWithout(doc: { [k: string]: Json }, ...exclude: string[]): string {
  const copy: { [k: string]: Json } = {};
  for (const [k, val] of Object.entries(doc)) if (!exclude.includes(k)) copy[k] = val;
  return sha256hex(canonicalize(copy));
}

// Decimal strings: digits only, no leading zero, at most 2^64-1. Use BigInt,
// never Number, to hold the value.
const DEC = /^(0|[1-9][0-9]*)$/;
export function isDecimalString(v: unknown): v is string {
  return typeof v === "string" && v.length <= 20 && DEC.test(v) && BigInt(v) <= 18446744073709551615n;
}

function ed25519Verify(pubB64: string, message: string, sigB64: string): boolean {
  const x = Buffer.from(pubB64, "base64").toString("base64url");
  const key = createPublicKey({ key: { kty: "OKP", crv: "Ed25519", x }, format: "jwk" });
  return verify(null, Buffer.from(message, "utf8"), key, Buffer.from(sigB64, "base64"));
}

// D-056: compare the exact external bytes to the signed reference before apply.
const resolverBytes = readFileSync(join(vectorsDir, "../examples/valid/ResolverDocument.json"));
const resolverState = load("desired-state-resolver-probe-signing.json").signed;
const resolverRef = resolverState.dns.resolver_document;
check(resolverBytes.toString("utf8") === canonicalize(parseStrictJSON(resolverBytes)), "resolver bytes already canonical");
check(resolverBytes.length === resolverRef.size_bytes, "signed resolver byte length");
check(sha256hex(resolverBytes) === resolverRef.content_sha256, "signed resolver byte digest");
check(sha256hex(Buffer.concat([resolverBytes, Buffer.from("\n")])) !== resolverRef.content_sha256, "resolver trailing newline rejected");
check(resolverRef.generation === "9007199254740993", "resolver policy generation retains precision");

// ---- canonical JSON vectors
const cj = load("canonical-json.json");
for (const c of cj.cases) {
  const input = Buffer.from(c.input_b64, "base64");
  const canon = canonicalize(parseStrictJSON(input));
  check(canon === c.canonical, `canonical-json ${c.name}`);
  check(sha256hex(Buffer.from(canon, "utf8")) === c.sha256, `canonical-json sha256 ${c.name}`);
}
// Check every reject with an independent lexical parser, including duplicate
// keys, UTF-8, surrogates, depth and original number spelling.
for (const r of cj.rejects) {
  let error = "accepted";
  try { parseStrictJSON(Buffer.from(r.input_b64, "base64")); }
  catch (e) { error = (e as Error).message; }
  check(error === r.error, `canonical-json reject ${r.name}: ${error}`);
}

// ---- decimal strings
const ds = load("decimal-strings.json");
for (const a of ds.accept) check(isDecimalString(a), `decimal accept ${a}`);
for (const r of ds.reject) check(!isDecimalString(JSON.parse(r.json)), `decimal reject ${r.json}`);

// ---- payload hashes
const ue = load("usage-event-payload-sha256.json");
check(canonicalize(ue.event_without_hash) === ue.canonical_without_hash, "usage event canonical form");
check(hashWithout(ue.event, "payload_sha256") === ue.payload_sha256, "usage event payload_sha256");
check(ue.event.payload_sha256 === ue.payload_sha256, "usage event carries payload_sha256");
check(isDecimalString(ue.event.records[1].upload_bytes), "2^53+1 bytes stay a decimal string");
const ur = load("update-result-payload-hash.json");
check(hashWithout(ur.result, "payload_hash") === ur.payload_hash, "update result payload_hash");

const urPolicy = load("update-result-policy-generation-hash.json");
check(hashWithout(urPolicy.result, "payload_hash") === urPolicy.payload_hash, "applied policy generation hash");
check(isDecimalString(urPolicy.result.policy_generation) && BigInt(urPolicy.result.policy_generation) === 9007199254740993n, "applied generation preserves precision");
check(hashWithout({ ...urPolicy.result, policy_generation: "9007199254740994" }, "payload_hash") !== urPolicy.payload_hash, "changed applied generation changes hash");

// ---- signing vectors
for (const f of ["release-trust-update-signing.json", "release-manifest-signing.json", "desired-state-signing.json", "desired-state-revocations-signing.json", "desired-state-resolver-probe-signing.json", "serving-lease-signing.json", "usage-checkpoint-signing.json"]) {
  const v = load(f);
  const canon = canonicalize(v.signed);
  check(canon === v.canonical_signed, `${f} canonical signed`);
  check(sha256hex(canon) === v.canonical_signed_sha256, `${f} canonical sha256`);
  check(ed25519Verify(v.public_key_b64, canon, v.signature_b64), `${f} signature verifies`);
  const env = v.envelope;
  check(env.signatures.length === 1 && env.signatures[0].algorithm === "ed25519" && env.signatures[0].key_id === v.key_id, `${f} envelope signature entry`);
  check(canonicalize(env.signed) === canon, `${f} envelope signed`);
  check(sha256hex(canonicalize(env)) === v.envelope_sha256, `${f} envelope sha256`);
  const tampered = canon.replace(/"(\d+)"/, (_m, d) => `"${BigInt(d) + 1n}"`);
  check(tampered !== canon && !ed25519Verify(v.public_key_b64, tampered, v.signature_b64), `${f} tampered content rejected`);
}

const man = load("release-manifest-signing.json");
const dsv = load("desired-state-signing.json");
check(dsv.signed.update.manifest_sha256 === man.envelope_sha256, "desired state binds the manifest envelope hash");

const revoked = load("desired-state-revocations-signing.json");
check(!ed25519Verify(revoked.public_key_b64, canonicalize({ ...revoked.signed, revoked_release_ids: [] }), revoked.signature_b64), "removed revocation invalidates signature");

// ---- revision 14: exact original-stream bytes/ACK and bounded TC digests
const handoff = load("usage-retired-handoff.json");
const acceptedHandoff = handoff.cases.find((c: any) => c.name === "accepted");
for (const c of handoff.cases) {
  const req = c.request;
  check(req.node_incarnation_id === handoff.context.presenting_incarnation_id, `D-105 ${c.name} current identity`);
  check(c.response.node_incarnation_id === req.origin_incarnation_id, `D-105 ${c.name} original-stream ACK`);
  for (const event of req.events) {
    check(event.node_incarnation_id === req.origin_incarnation_id, `D-105 ${c.name} immutable origin`);
    check(hashWithout(event, "payload_sha256") === event.payload_sha256, `D-105 ${c.name} original hash`);
  }
  if (c.name === "idempotent-replay") {
    check(canonicalize(req) === canonicalize(acceptedHandoff.request), "D-105 byte-identical replay");
    check(c.response.results[0].status === "duplicate", "D-105 duplicate without recounting");
  }
  if (c.name === "after-retirement-window") {
    check(c.audited && c.response.results[0].error.code === "VALIDATION_FAILED", "D-105 late event audited rejection");
    check(c.response.highest_contiguous_seq === handoff.context.previous_highest_contiguous_seq, "D-105 rejected ACK unchanged");
  }
}
for (const c of load("tc-readback-evidence.json").cases) {
  const e = c.evidence;
  const projection = canonicalize({ reads: e.reads_total, objects: e.objects_total, truncated: e.truncated, tc: e.attachments });
  check(projection === c.canonical_projection, `D-106 ${c.name} canonical rows/counts`);
  check(sha256hex(projection) === e.sha256, `D-106 ${c.name} digest`);
  check(isDecimalString(e.candidate_generation), `D-106 ${c.name} exact generation`);
  check(e.attachments.length <= 16 && e.attachments.every((r: any) => r.objects.length <= 4), `D-106 ${c.name} bounded rows`);
  check(e.rejected === null || e.rejected.objects.length <= 1, `D-106 ${c.name} deciding row bound`);
}

if (failures > 0) {
  console.error(`${failures} of ${checks} vector checks failed`);
  process.exit(1);
}
console.log(`ok: ${checks} vector checks passed (TypeScript, independent of Go)`);
