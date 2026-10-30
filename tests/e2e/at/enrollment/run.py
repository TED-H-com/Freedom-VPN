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
    "counted_unacked_usage": "ENR-05 / devstack follow-up 13: HoldUsageAcknowledgements, UsageLedgerSnapshot and CountedMemberTraffic are not exported",
    "installer_rerun": "ENR-06 / devstack follow-up 13: RerunInstaller is not exported",
    "identity_ledger_preserved": "ENR-05/ENR-06: counted unacknowledged ledger and installer rerun required",
    "controlled_reinstall": "ENR-06 / devstack follow-up 13: ControlledReinstall is not exported",
    "old_certificate_denied": "ENR-06: retain old certificate locally and exercise mTLS after controlled reinstall",
    "historical_usage_retained": "ENR-05/ENR-06 / devstack follow-up 13: counted usage across RerunInstaller/ControlledReinstall is not exported; listNodeIncarnations usage retention is measured separately",
    "cloned_identity_conflict": "ENR-07 StartClonedIdentityFixture was not observed; devstack follow-up 12 PR #204 exports it",
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


def previous(docdir, at):
    path = docdir / (at + ".json")
    if not path.exists():
        return []
    old = json.loads(path.read_text())
    # Retain every previous check, including failures and blocked prerequisites.
    return old.get("prior_checks", []) + [dict(c, run_id=old["run_id"]) for c in old["checks"]]


def verdict(checks, prior):
    # A failed subcheck remains explicit evidence and blocks the complete AT.
    # The follow-up's aggregate gate is pass/blocked, never green on a failure.
    if any(c["status"] != "pass" for c in checks):
        return "blocked"
    if any(c["status"] == "fail" for c in prior):
        return "blocked"  # explicit orchestrator review is required to close a prior failure
    return "pass"


def records(meta, observed, prior):
    result = {}
    for at, required in REQUIRED.items():
        checks = [c for c in observed if c["at"] == at]
        seen = {c["name"] for c in checks}
        checks += [{"at": at, "name": name, "status": "blocked", "observed_at": meta["finished_at"],
                    "detail": MISSING.get(name, "Not reached: fresh-stack/API prerequisite unavailable")}
                   for name in required if name not in seen]
        status = verdict(checks, prior[at])
        failed = next((c for c in checks if c["status"] == "fail"), None)
        decisive = failed or next((c for c in checks if c["status"] == "blocked"), None)
        actual = (str(decisive["detail"]) if decisive else
                  "Prior failed check requires explicit review" if status != "pass" else "All required checks passed")
        result[at] = dict(meta, id=at, status=status, actual=actual,
                          requirements=["ENR01", "ENR02", "SEC01"] if at == "AT06" else ["ENR03", "PKG01", "DATA01"],
                          spec=["§19 " + at, "§6.1", "§6.2", "§6.3", "§6.4"],
                          checks=checks, prior_checks=prior[at])
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
                    code = run("live_probe", [str(binary), "-test.v", "-test.run=^TestAT06_AT07_Live$", "-test.timeout=30m"],
                               {"FVPN_E2E": "1", "FVPN_AT_ENROLLMENT_PHASE": "full", "FVPN_AT_ENROLLMENT_PROBE": str(probe_path)})
                    if code:
                        for at in REQUIRED:
                            note(at, "probe_process", "fail", "Live probe process returned nonzero; completed checks retained")
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
                    if code:
                        note("AT06", "probe_process", "fail", "Join-cycle probe process returned nonzero; completed checks retained")
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
