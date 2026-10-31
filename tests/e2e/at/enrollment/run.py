#!/usr/bin/env python3
"""Records-first AT06/AT07 runner; all external effects use the owned lab."""

import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import tempfile
from datetime import datetime, timezone
import uuid

ROOT = Path(__file__).resolve().parents[4]
DOCS = ROOT / "docs/e2e/at/enrollment"
PREFIX = "w4-enroll"
PORT_BASE = "12000"
REQUIRED = {
    "AT06": ["fresh_stack", "response_dropped", "same_holder_retry", "one_node",
             "current_incarnation", "consumed_binding", "single_consumption_audit",
             "incarnation_uniqueness", "incarnation_history", "token_sha256_create",
             "token_sha256_get", "token_sha256_revoke", "audit_token_hash",
             "peer_uniqueness", "csr_proof",
             "different_key_race", "race_winner_identity", "expired_token", "revoked_token", "token_only",
             "invalid_proof", "bootstrap_requires_mtls", "retry_no_store", "evidence_set_scan",
             "secret_scan_admin_reads", "secret_scan_current_probe_evidence",
             "secret_scan_control/console", "secret_scan_worker/console",
             "secret_scan_node/console", "secret_scan_node/agent.log",
             "secret_scan_node/watchdog.log", "secret_scan_node/netbird-service.log",
             "secret_scan_node/netbird-up.log", "secret_scan_node/resolver-proxy.log"],
    "AT07": ["fresh_stack", "distinct_api_first_boot", "distinct_first_boot", "current_incarnation",
             "incarnation_uniqueness", "incarnation_history",
             "machine_identity", "counted_unacked_usage", "installer_rerun",
             "identity_ledger_preserved", "controlled_reinstall", "old_certificate_denied",
             "historical_usage_retained", "cloned_identity_conflict"],
}
MISSING = {
    "peer_uniqueness": "ENR-04 join cycle did not record EnrollmentPeerInventory; devstack follow-up 12 PR #204 exports JoinEnrolledIdentity",
    "distinct_first_boot": "ENR-07 MachineIdentityFingerprints was not read; devstack follow-up 12 PR #204 exports it",
    "machine_identity": "ENR-07 MachineIdentityFingerprints was not read; devstack follow-up 12 PR #204 exports it",
    "counted_unacked_usage": "ENR-05: devstack #266 usage hooks exclude installer-node. Owner devstack: expose the hold, ledger snapshot and selected member transport on the installer slot. Owner usage/installer: define authorized delivery of retained old-incarnation events after reenroll; current Reenroll preserves them unuploaded.",
    "installer_rerun": "ENR-06: installer probe not reached. Installer #279 fixes the prior startup cause; owner devstack: restore prior service state after wrapper hash sampling.",
    "identity_ledger_preserved": "ENR-05/ENR-06: requires a running installer-node with counted unACKed usage, retired mode 0700 and newest-first incarnation history. Not reached.",
    "controlled_reinstall": "ENR-06: ControlledReinstall on installer-node not reached; a successful initial install and documented service restoration are prerequisites.",
    "old_certificate_denied": "ENR-06: retired installer certificate unavailable until ControlledReinstall; desired-state and observation denial not reached.",
    "historical_usage_retained": "ENR-05/ENR-06: nonzero retired usage plus new-incarnation traffic requires usage hooks on installer-node and authorized backlog delivery. Not reached.",
    "cloned_identity_conflict": "ENR-07: StartClonedIdentityFixture was not observed on fvpn-<prefix>-node-clone-net.",
}

# D-078. A prior fail stays in prior_checks. It stops blocking only when this
# run's rate_limit_review passes the behaviour rule and the named checks pass.
# http_429 count is evidence, not the pass condition. Any other prior fail
# still blocks.
REVIEWED = {
    "AT06": {
        "commit": "286eb57b44c2e783257767918680a0e1f097f106",
        "what_changed": "D-078 scores rate_limit_review by behaviour. A 429 that carries Retry-After and whose retry of the same request succeeds is the browser limiter working. The criterion that the review resolves only when a later run records http_429 0 is withdrawn. Harness Retry-After retries remain 286eb57 (#210).",
        "sentence": "Prior run bebfe230 429 RATE_LIMITED failures on audit_read, issue_reinstall, incarnation_history and probe_process are reviewed as resolved under D-078. Harness Retry-After retries landed in 286eb57b44c2e783257767918680a0e1f097f106 (#210). Run 60a7102e is re-scored in this record: raw http_429 2, both class browser, named checks passed, join phase http_429 0. The saved samples do not include a Retry-After value. Run 35e542f8 probe_process and rate_limit_review stay in prior_checks and stop blocking on that behaviour evidence. Run 1d5c02a4 probe_process was the clone-present parser (wget dropped the 403 body); it stops blocking when this probe exits 0. Limiter class was browser BrowserRate 10/s burst 30.",
        "evidence_names": ("audit_read", "issue_reinstall", "incarnation_history", "probe_process"),
        "rate_limit": True,
        "runs": (
            {"run_id": "bebfe230-26d4-4e3d-b052-81d258d04e07",
             "names": ("audit_read", "issue_reinstall", "incarnation_history", "probe_process")},
            {"run_id": "35e542f8-730e-4be0-8214-e4a9dd818c40",
             "names": ("probe_process", "rate_limit_review")},
            {"run_id": "1d5c02a4-b254-4a9b-b687-b1535428fb3f",
             "names": ("probe_process",)},
        ),
        "rescored_follow_up_3": {
            "run_id": "60a7102e-b824-4927-a576-62033f458157",
            "raw_status": "blocked",
            "http_429": 2,
            "join_http_429": 0,
            "header_value_recorded": False,
            "samples": [
                {"class": "browser", "method": "GET", "path": "/api/v1/admin/audit-events"},
                {"class": "browser", "method": "GET", "path": "/api/v1/admin/nodes/01a18f10-3f31-7ae7-890a-55354e77cec0/incarnations"},
            ],
            "named_checks_passed": ["audit_read", "issue_reinstall", "incarnation_history"],
            "reasoning": "D-078 re-score of follow-up 3 run 60a7102e. The raw rate_limit_review stayed blocked with http_429 2 and those two paths; that count and the samples stay in prior_checks. Both samples are limiter class browser. The saved samples do not include a Retry-After header value, so none is invented here. On that run callHeaders retried each 429 with the same Idempotency-Key using the Retry-After header (1-30s; the 1s default applies only when the header is absent). The product emits Retry-After on RATE_LIMITED. audit_read, issue_reinstall and incarnation_history passed, which is the evidence those retried requests succeeded inside the window. The join-phase rate_limit_review on that run recorded http_429 0. Under D-078 that is the limiter working. A 429 without Retry-After, a retry that is still 429, a non-browser class on this browser-table review, or any join-phase 429 remains a finding. The zero-count resolution criterion is withdrawn.",
        },
    },
    "AT07": {
        "commit": "#210 retries; #248 wget --content-on-error; follow-up 5 cross-case process review",
        "what_changed": "Runs bebfe230 and 35e542f8 each contain exactly one AT07 failure: probe_process. The runner copied the shared live_probe exit 1 to both ATs. bebfe230 AT06 failed audit_read, issue_reinstall and incarnation_history on browser 429; 35e542f8 AT06 failed only the obsolete zero-429 rate_limit_review. Neither has an independent AT07 assertion failure. Their review now requires this run's explicit successful probe processes and passing AT06 audit_read, issue_reinstall, incarnation_history and D-078 behavior review (browser/read under #252). Run 1d5c02a4 observed the second /24, one frozen conflict, one audit, a stable count, revoke 202, a new incarnation, and identity_conflict {mapping_frozen:false, conflict_count:0, last_conflict_at:null}. The clone's desired-state request returned HTTP 403. wget omitted --content-on-error, so the body was empty and the error code was not parsed. That process exit is the same parser miss. It is not a product conflict failure.",
        "sentence": "AT07 bebfe230 and 35e542f8 probe_process rows are resolved by current successful processes and AT06 behavioral evidence; the historical failures stay. Run 1d5c02a4 cloned_identity_conflict and probe_process failed because wget dropped the HTTP 403 body. The conflict, freeze, audit, revoke, and re-enroll observations in that record stay. This run reads the error code with --content-on-error.",
        "evidence_names": ("cloned_identity_conflict", "probe_process"),
        "supporting_at06": ("audit_read", "issue_reinstall", "incarnation_history", "rate_limit_review", "probe_process"),
        "runs": (
            {"run_id": "1d5c02a4-b254-4a9b-b687-b1535428fb3f",
             "names": ("cloned_identity_conflict", "probe_process")},
            {"run_id": "bebfe230-26d4-4e3d-b052-81d258d04e07", "names": ("probe_process",)},
            {"run_id": "35e542f8-730e-4be0-8214-e4a9dd818c40", "names": ("probe_process",)},
        ),
    },
}


def now():
    return datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")


def output(args):
    return subprocess.check_output(args, cwd=ROOT, text=True).strip()


def source():
    digest = hashlib.sha256()
    names = output(["git", "ls-files", "--cached", "--others", "--exclude-standard"]).splitlines()
    for name in sorted(set(names)):
        if name.startswith("docs/e2e/at/enrollment/"):
            continue
        path = ROOT / name
        if path.is_file():
            digest.update(name.encode() + b"\0" + hashlib.sha256(path.read_bytes()).digest())
    return {"commit": output(["git", "rev-parse", "HEAD"]),
            "tree_sha256": digest.hexdigest(),
            "dirty": bool(output(["git", "status", "--porcelain"])),
            "tree_hash_scope": "sorted tracked and nonignored files, excluding generated enrollment evidence"}


def digest_labels(value):
    """Relabel copied prior evidence; immutable history files remain byte-identical."""
    names = {"Key": "key_sha256", "Cert": "cert_sha256", "NodeJSON": "node_json_sha256",
             "UsageDB": "usage_db_sha256", "UsageWAL": "usage_wal_sha256",
             "MachineID": "machine_id_sha256", "Current": "current"}
    if isinstance(value, list):
        return [digest_labels(item) for item in value]
    if isinstance(value, dict):
        if set(value) == set(names):
            return {names[key]: item for key, item in value.items()}
        return {key: digest_labels(item) for key, item in value.items()}
    return value


def previous(docdir, at):
    path = docdir / (at + ".json")
    if not path.exists():
        return []
    old = json.loads(path.read_text())
    # Retain every previous check, including failures and blocked prerequisites.
    return digest_labels(old.get("prior_checks", []) + [dict(c, run_id=old["run_id"]) for c in old["checks"]])


def observation(detail):
    if isinstance(detail, dict) and isinstance(detail.get("gap"), str):
        return detail["gap"]
    return str(detail)


def flag(value):
    return value is True or value == "true"


def behaviour_ok(detail):
    """D-078. A zero count still passes. A positive count passes only when every
    sample is browser/read class (#252), carried Retry-After, and the next attempt succeeded.
    The join phase must stay at 0. The count itself is not the pass condition.
    """
    if not isinstance(detail, dict) or detail.get("behaviour") == "fail":
        return False
    count = detail.get("http_429")
    if detail.get("phase") == "join":
        return count == 0 and detail.get("behaviour", "pass") == "pass"
    if count == 0:
        return detail.get("behaviour", "pass") == "pass"
    if detail.get("behaviour") != "pass" or not isinstance(count, int):
        return False
    samples = detail.get("samples")
    if not isinstance(samples, list) or len(samples) != count:
        return False
    for sample in samples:
        if not isinstance(sample, dict) or sample.get("class") not in ("browser", "read"):
            return False
        if not flag(sample.get("retry_after")) or not flag(sample.get("retry_succeeded")):
            return False
    return True


def review_evidence(at, checks):
    spec = REVIEWED.get(at)
    if not spec:
        return False
    by_name = {}
    for check in checks:
        if check.get("at") == at:
            by_name.setdefault(check["name"], []).append(check)
    for name in spec["evidence_names"]:
        hits = by_name.get(name, [])
        if not hits or any(hit["status"] != "pass" for hit in hits):
            return False
    for name in spec.get("supporting_at06", ()):
        hits = [c for c in checks if c.get("at") == "AT06" and c["name"] == name]
        if not hits or any(c["status"] != "pass" for c in hits):
            return False
        if name == "rate_limit_review" and not all(behaviour_ok(c["detail"]) for c in hits):
            return False
    if not spec.get("rate_limit"):
        return True
    reviews = by_name.get("rate_limit_review", [])
    return bool(reviews) and all(hit["status"] == "pass" and behaviour_ok(hit["detail"]) for hit in reviews)


def review_covers(at, prior, checks):
    spec = REVIEWED.get(at)
    if not spec or not review_evidence(at, checks):
        return False
    return any(prior.get("run_id") == group["run_id"] and prior.get("name") in group["names"]
               for group in spec["runs"])


def verdict(at, checks, prior, review_checks=None):
    # A failed subcheck remains explicit evidence and blocks the complete AT.
    # An unreviewed prior failure still blocks. A reviewed prior failure stays
    # visible and stops blocking only when review_evidence is true.
    if review_checks is None:
        review_checks = checks
    if any(c["status"] != "pass" for c in checks):
        return "blocked"
    if any(c["status"] == "fail" and not review_covers(at, c, review_checks) for c in prior):
        return "blocked"
    return "pass"


def records(meta, observed, prior):
    result = {}
    for at, required in REQUIRED.items():
        checks = [c for c in observed if c["at"] == at]
        seen = {c["name"] for c in checks}
        checks += [{"at": at, "name": name, "status": "blocked", "observed_at": meta["finished_at"],
                    "detail": MISSING.get(name, "Not reached: fresh-stack/API prerequisite unavailable")}
                   for name in required if name not in seen]
        review_checks = checks + [c for c in observed if c["at"] != at]
        status = verdict(at, checks, prior[at], review_checks)
        failed = next((c for c in checks if c["status"] == "fail"), None)
        decisive = failed or next((c for c in checks if c["status"] == "blocked"), None)
        reviewed_pass = status == "pass" and any(c["status"] == "fail" for c in prior[at])
        actual = (observation(decisive["detail"]) if decisive else
                  "Prior failed check requires explicit review" if status != "pass" else
                  REVIEWED[at]["sentence"] if reviewed_pass else "All required checks passed")
        record = dict(meta, id=at, status=status, actual=actual,
                      requirements=["ENR01", "ENR02", "SEC01"] if at == "AT06" else ["ENR03", "PKG01", "DATA01"],
                      spec=["§19 " + at, "§6.1", "§6.2", "§6.3", "§6.4"],
                      checks=checks, prior_checks=prior[at])
        if at in REVIEWED:
            disposition = "not_applied"
            uncovered = any(c["status"] == "fail" and not review_covers(at, c, review_checks) for c in prior[at])
            if review_evidence(at, review_checks) and not uncovered:
                disposition = "resolved"
            elif any(c["name"] == "rate_limit_review" and (c["status"] != "pass" or not behaviour_ok(c["detail"])) for c in checks):
                disposition = "recurs"
            review = {
                "disposition": disposition,
                "run_id": REVIEWED[at]["runs"][0]["run_id"],
                "names": list(REVIEWED[at]["runs"][0]["names"]),
                "also_covers": [{"run_id": group["run_id"], "names": list(group["names"])}
                                for group in REVIEWED[at]["runs"][1:]],
                "commit": REVIEWED[at]["commit"],
                "what_changed": REVIEWED[at]["what_changed"],
            }
            if REVIEWED[at].get("rate_limit"):
                review.update({
                    "limiter_class": "browser",
                    "bucket": "BrowserRate",
                    "per_second": 10,
                    "burst": 30,
                    "owner_if_recurs": "control",
                    "admin_list_class": "read", "admin_list_per_second": 30, "admin_list_burst": 60,
                    "calibration_prs": [233, 252],
                    "rule": "D-078 with #252: every 429 carried Retry-After, the same request succeeded next attempt, class browser/read, join http_429 0. Count is informational.",
                })
            if "rescored_follow_up_3" in REVIEWED[at]:
                review["rescored_follow_up_3"] = REVIEWED[at]["rescored_follow_up_3"]
            record["prior_failure_review"] = review
        result[at] = record
    return result


def persist(docdir, recs, archive):
    docdir.mkdir(parents=True, exist_ok=True)
    archive.mkdir(parents=True, exist_ok=True)
    summary = ["# Enrollment acceptance", "", "Every required check must pass. Missing hooks keep the complete AT blocked.", "",
               "| AT | Status | Decisive observation | Evidence |", "|---|---|---|---|"]
    for at, rec in recs.items():
        raw = json.dumps(rec, indent=2, ensure_ascii=False) + "\n"
        # Fail before publishing if an unexpected raw secret or local home path
        # enters the explicitly selected evidence. Do not silently redact a leak.
        if re.search(r"fvpn_enr_[A-Za-z0-9_-]{20,}|-----BEGIN .*PRIVATE KEY|/" + r"home/", raw):
            raise ValueError("unsafe evidence rejected; raw value withheld")
        text = (f"# {at}\n\nStatus: **{rec['status']}**\n\n{rec['actual']}\n\n"
                f"Run: `{rec['run_id']}`; {rec['started_at']} → {rec['finished_at']}.\n\n"
                f"Source: `{rec['source']['commit']}`; tree SHA-256 `{rec['source']['tree_sha256']}`.\n\n"
                f"Images: {rec['image_evidence']}.\n\nRequirements: {', '.join(rec['requirements'])}.\n\n"
                f"Prior checks retained: {len(rec['prior_checks'])}.\n\n"
                "| Check | Status | Observation |\n|---|---|---|\n")
        for c in rec["checks"]:
            detail = json.dumps(c["detail"], ensure_ascii=False) if not isinstance(c["detail"], str) else c["detail"]
            text += f"| {c['name']} | {c['status']} | {detail.replace('|', '/')} |\n"
        text += f"\nFull provenance and prior checks: [{at}.json]({at}.json). Requests: [requests.md](requests.md).\n"
        for directory in (docdir, archive):
            (directory / (at + ".json")).write_text(raw)
            (directory / (at + ".md")).write_text(text)
        summary.append(f"| {at} | {rec['status']} | {rec['actual']} | [{at}.json]({at}.json), [{at}.md]({at}.md) |")
    summary += ["", "See [RUN.md](RUN.md) and [requests.md](requests.md). Immutable attempts are in `history/`.", ""]
    (docdir / "summary.md").write_text("\n".join(summary))
    (archive / "summary.md").write_text("\n".join(summary))
    (archive / "RUN.md").write_text("# Archived attempt\n\nSee [current runbook](../../RUN.md). "
                                    "Exact source, UTC times and command exits are in the AT JSON records.\n")
    requests = docdir / "requests.md"
    if requests.exists():
        (archive / "requests.md").write_text(requests.read_text().replace("(../../RUN.md)", "(../../../../RUN.md)"))


def inventory():
    queries = {"containers": ["docker", "ps", "-a", "--format", "{{.Names}}"],
               "networks": ["docker", "network", "ls", "--format", "{{.Name}}"],
               "volumes": ["docker", "volume", "ls", "--format", "{{.Name}}"],
               "images": ["docker", "images", "--format", "{{.Repository}}:{{.Tag}}"]}
    counts = {}
    for kind, cmd in queries.items():
        names = output(cmd).splitlines()
        counts[kind] = sum(n.startswith("fvpn-" + PREFIX + "-") or n == "fvpn-" + PREFIX for n in names)
    return counts


def main():
    run_id = str(uuid.uuid4())
    private = ROOT / ".agent/at-enrollment" / run_id
    private.mkdir(parents=True)
    archive = DOCS / "history" / run_id
    prior = {at: previous(DOCS, at) for at in REQUIRED}
    meta = {"run_id": run_id, "started_at": now(), "finished_at": now(), "source": source(),
            "prefix": PREFIX, "port_base": int(PORT_BASE), "image_digests": {},
            "image_evidence": "unavailable: no stack has started", "commands": [],
            "test_binary_sha256": None, "cleanup": None}
    observed = []
    persist(DOCS, records(meta, observed, prior), archive)
    (ROOT / ".agent/tmp").mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="at-enrollment-", dir=ROOT / ".agent/tmp") as scratch:
        env = dict(os.environ, FVPN_E2E_PREFIX=PREFIX, FVPN_E2E_PORT_BASE=PORT_BASE,
                   FVPN_E2E_STATE=str(ROOT / ".agent/e2e-state-w4-enroll"),
                   TMPDIR=scratch, GOTMPDIR=scratch, GOTOOLCHAIN="local", CGO_ENABLED="0")

        def run(label, cmd, extra=None, timeout=1800):
            started = now()
            with (private / (label + ".log")).open("wb") as log:
                proc = subprocess.Popen(cmd, cwd=ROOT, env=dict(env, **(extra or {})), stdout=log,
                                        stderr=subprocess.STDOUT, start_new_session=True)
                try:
                    code = proc.wait(timeout=timeout)
                except subprocess.TimeoutExpired:
                    os.killpg(proc.pid, signal.SIGKILL)
                    proc.wait()
                    code = 124
                except BaseException:
                    os.killpg(proc.pid, signal.SIGKILL)
                    proc.wait()
                    raise
            meta["commands"].append({"name": label, "started_at": started, "finished_at": now(), "exit_code": code})
            print(f"{label}: exit {code}", flush=True)
            return code

        def note(at, name, status, detail):
            observed.append(dict(at=at, name=name, status=status, detail=detail, observed_at=now()))

        probe_path = private / "probe.json"
        join_probe_path = private / "probe-join.json"
        binary = Path(scratch) / "enrollment.test"

        def compile_probe():
            if binary.exists():
                return 0
            built = run("compile_probe", ["go", "test", "-tags", "e2e", "-c", "-o", str(binary), "./tests/e2e/at/enrollment"])
            if built == 0:
                meta["test_binary_sha256"] = hashlib.sha256(binary.read_bytes()).hexdigest()
            return built

        def bring_up(label):
            down = run(label + "_teardown", ["make", "-f", "mk/e2e.mk", "e2e-down"])
            if down or any(inventory().values()):
                return False, "Owned-prefix fresh teardown did not complete"
            up = run(label + "_startup", ["make", "-f", "mk/e2e.mk", "e2e-up"], {"E2E_RESET": "1"})
            startup = (private / (label + "_startup.log")).read_text(errors="replace")
            if "port base slot is out of range" in startup:
                return False, f"ENR-01: shared prepare rejected assigned port base {PORT_BASE}: port base slot is out of range"
            if up:
                return False, "Fresh stack startup failed; private diagnostic log retained"
            return True, ""

        def digests():
            return {s: output(["docker", "inspect", "--format={{.Image}}", "fvpn-w4-enroll-" + s])
                    for s in ("control", "worker", "node")}

        try:
            # Acceptance stack: API enrollments, both clean first boots, fingerprints
            # and the clone fixture. The own-key join needs node2 while it is still
            # waiting, so it runs on the next fresh stack.
            up, reason = bring_up("fresh")
            for at in REQUIRED:
                note(at, "fresh_stack", "pass" if up else "blocked",
                     "E2E_RESET=1 fresh stack started" if up else reason)
            if up:
                meta["image_digests"] = digests()
                meta["image_evidence"] = "immutable running container image IDs"
                if compile_probe() == 0:
                    code = run("live_probe", [str(binary), "-test.v", "-test.run=^TestAT06_AT07_Live$", "-test.timeout=45m"],
                               {"FVPN_E2E": "1", "FVPN_AT_ENROLLMENT_PHASE": "full", "FVPN_AT_ENROLLMENT_PROBE": str(probe_path)},
                               timeout=2700)
                    for at in REQUIRED:
                        note(at, "probe_process", "fail" if code else "pass",
                             {"phase": "full", "exit_code": code})
                else:
                    for at in REQUIRED:
                        note(at, "probe_compile", "fail", "Live acceptance probe did not compile")
            # Join stack: JoinEnrolledIdentity consumes the only fresh waiting slot.
            join_up, join_reason = bring_up("join")
            if not join_up:
                note("AT06", "peer_uniqueness", "blocked", "ENR-04 join cycle did not start: " + join_reason)
            else:
                meta["join_image_digests"] = digests()
                if compile_probe() == 0:
                    code = run("live_probe_join", [str(binary), "-test.v", "-test.run=^TestAT06_AT07_Live$", "-test.timeout=30m"],
                               {"FVPN_E2E": "1", "FVPN_AT_ENROLLMENT_PHASE": "join", "FVPN_AT_ENROLLMENT_PROBE": str(join_probe_path)})
                    note("AT06", "probe_process", "fail" if code else "pass",
                         {"phase": "join", "exit_code": code})
                else:
                    note("AT06", "probe_compile", "fail", "Live acceptance probe did not compile")
        except (OSError, subprocess.CalledProcessError, ValueError):
            for at in REQUIRED:
                note(at, "runner", "fail", "Runner prerequisite failed; exception content withheld")
        finally:
            for path in (probe_path, join_probe_path):
                if path.exists():
                    try:
                        observed += json.loads(path.read_text())
                    except (OSError, ValueError):
                        for at in REQUIRED:
                            note(at, "probe_evidence", "fail", "Probe evidence could not be decoded")
            down = run("final_teardown", ["make", "-f", "mk/e2e.mk", "e2e-down"])
            try:
                meta["cleanup"] = inventory()
            except (OSError, subprocess.CalledProcessError):
                meta["cleanup"] = {"unavailable": True}
            if down or any(meta["cleanup"].values()):
                for at in REQUIRED:
                    note(at, "cleanup", "fail", "Owned-prefix cleanup was not confirmed")
            meta["finished_at"] = now()
            recs = records(meta, observed, prior)
            persist(DOCS, recs, archive)
    for at, rec in recs.items():
        print(f"{at}: {rec['status']}: {rec['actual']}")
    print("records: docs/e2e/at/enrollment; run " + run_id)
    return 0 if all(r["status"] == "pass" for r in recs.values()) else 1


if __name__ == "__main__":
    raise SystemExit(main())
