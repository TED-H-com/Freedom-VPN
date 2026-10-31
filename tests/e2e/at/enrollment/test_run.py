import json
from pathlib import Path
import tempfile
import unittest

import run


class EvidenceTests(unittest.TestCase):
    def meta(self):
        return dict(run_id="test-run", started_at=run.now(), finished_at=run.now(),
                    source={"commit": "test-source", "tree_sha256": "test-tree"},
                    image_evidence="unavailable: TEST ONLY")

    def test_missing_live_results_write_both_records_and_block(self):
        prior = {at: [] for at in run.REQUIRED}
        recs = run.records(self.meta(), [], prior)
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)
            run.persist(path, recs, path / "history/test-run")
            for at in run.REQUIRED:
                record = json.loads((path / (at + ".json")).read_text())
                self.assertEqual(record["status"], "blocked")
                self.assertTrue((path / (at + ".md")).exists())
                self.assertTrue((path / "history/test-run" / (at + ".json")).exists())
                self.assertEqual({c["name"] for c in record["checks"]}, set(run.REQUIRED[at]))

    def test_digest_relabel_preserves_prior_status_and_archive(self):
        hashes = dict(Key="a" * 64, Cert="b" * 64, NodeJSON="c" * 64,
                      UsageDB="d" * 64, UsageWAL="absent", MachineID="e" * 64,
                      Current="/opt/vpn-node/releases/test-only")
        original = json.dumps(dict(run_id="prior-run", checks=[dict(
            name="installer_rerun", status="blocked", detail={"after": hashes})]))
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)
            record = path / "AT07.json"
            record.write_text(original)
            copied = run.previous(path, "AT07")
            self.assertEqual(record.read_text(), original)
            self.assertEqual(copied[0]["status"], "blocked")
            self.assertEqual(copied[0]["run_id"], "prior-run")
            labels = copied[0]["detail"]["after"]
            self.assertEqual(labels["key_sha256"], hashes["Key"])
            self.assertEqual(labels["usage_wal_sha256"], "absent")
            self.assertEqual(labels["current"], hashes["Current"])
            self.assertNotIn("Key", labels)
            self.assertNotIn("Cert", labels)

    def test_archived_followup6_retains_prior_failures(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)
            archive = path / "history/follow-up-6"
            archive.mkdir(parents=True)
            (archive / "AT06.json").write_text(json.dumps(dict(
                run_id="followup6", prior_checks=[dict(name="old", status="fail")],
                checks=[dict(name="current", status="pass")])))
            rows = run.previous(path, "AT06")
            self.assertEqual([r["status"] for r in rows], ["fail", "pass"])
            self.assertEqual(rows[1]["run_id"], "followup6")

    def test_committed_evidence_hygiene(self):
        # Scan all immutable attempts as well as the current publication.
        import re
        directory = Path(__file__).resolve().parents[4] / "docs/e2e/at/enrollment"
        files = list(directory.rglob("*.json")) + list(directory.rglob("*.md"))
        self.assertTrue(files)
        unsafe = re.compile(r"fvpn_enr_[A-Za-z0-9_-]{20,}|-----BEGIN .*PRIVATE KEY|/" + r"home/|sk_" + r"live_|gh" + r"p_|gh" + r"o_")
        for path in files:
            raw = path.read_text()
            self.assertIsNone(unsafe.search(raw), str(path.relative_to(directory)))
            if path.suffix == ".json":
                rec = json.loads(raw)
                self.assertIn(rec["status"], ("pass", "blocked", "fail"))
                self.assertTrue(rec["run_id"])

    def test_failed_prior_cannot_be_silently_superseded(self):
        checks = [dict(at=at, name=name, status="pass", detail="TEST ONLY")
                  for at, names in run.REQUIRED.items() for name in names]
        prior = {at: [] for at in run.REQUIRED}
        prior["AT06"] = [dict(name="different_key_race", status="fail", run_id="prior-run")]
        recs = run.records(self.meta(), checks, prior)
        self.assertEqual(recs["AT06"]["status"], "blocked")
        self.assertEqual(recs["AT07"]["status"], "pass")
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)
            run.persist(path, recs, path / "history/test-run")
            retained = run.previous(path, "AT06")
            self.assertEqual(retained[0]["status"], "fail")
            self.assertEqual(retained[0]["run_id"], "prior-run")

    def test_failure_dominates_blocker_and_repeated_check(self):
        checks = [dict(at="AT06", name="same_holder_retry", status=s, detail="TEST ONLY")
                  for s in ("fail", "pass")]
        recs = run.records(self.meta(), checks, {at: [] for at in run.REQUIRED})
        self.assertEqual(recs["AT06"]["status"], "blocked")
        self.assertEqual(recs["AT06"]["checks"][0]["status"], "fail")

    def test_secret_rejected_before_publication(self):
        recs = run.records(self.meta(), [], {at: [] for at in run.REQUIRED})
        # TEST ONLY - NOT A SECRET: assembled scanner sentinel.
        recs["AT06"]["actual"] = "fvpn_" + "enr_" + "a" * 43
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)
            with self.assertRaises(ValueError):
                run.persist(path, recs, path / "history/test-run")
            self.assertFalse((path / "AT06.json").exists())


    def passing(self):
        checks = [dict(at=at, name=name, status="pass", detail="TEST ONLY")
                  for at, names in run.REQUIRED.items() for name in names]
        checks.append(dict(at="AT06", name="rate_limit_review", status="pass", detail={"http_429": 0}))
        # These are emitted by the probe and are not required-name fillers.
        for name in ("audit_read", "issue_reinstall"):
            checks.append(dict(at="AT06", name=name, status="pass", detail="TEST ONLY"))
        for at in run.REQUIRED:
            checks.append(dict(at=at, name="probe_process", status="pass", detail={"exit_code": 0, "phase": "full"}))
        return checks

    def reviewed_prior(self):
        prior = {at: [] for at in run.REQUIRED}
        group = run.REVIEWED["AT06"]["runs"][0]
        prior["AT06"] = [dict(name=name, status="fail", run_id=group["run_id"]) for name in group["names"]]
        return prior

    def with_review(self, review):
        checks = [c for c in self.passing() if c["name"] != "rate_limit_review"]
        checks.append(review)
        return checks

    def review(self, **detail):
        body = {"http_429": 0, "behaviour": "pass", "phase": "full", "samples": []}
        body.update(detail)
        return dict(at="AT06", name="rate_limit_review", status="pass", detail=body)

    def browser_samples(self, retry_after=True, retry_succeeded=True, klass="browser"):
        return [
            {"class": klass, "method": "GET", "path": "/api/v1/admin/audit-events",
             "retry_after": retry_after, "retry_succeeded": retry_succeeded},
            {"class": klass, "method": "GET", "path": "/api/v1/admin/nodes/{id}/incarnations",
             "retry_after": retry_after, "retry_succeeded": retry_succeeded},
        ]

    def test_reviewed_429_prior_passes_when_current_run_has_zero(self):
        recs = run.records(self.meta(), self.passing(), self.reviewed_prior())
        self.assertEqual(recs["AT06"]["status"], "pass")
        self.assertEqual(recs["AT06"]["prior_failure_review"]["disposition"], "resolved")
        self.assertIn("286eb57b44c2e783257767918680a0e1f097f106", recs["AT06"]["actual"])
        self.assertEqual(recs["AT06"]["prior_failure_review"]["rescored_follow_up_3"]["http_429"], 2)
        self.assertFalse(recs["AT06"]["prior_failure_review"]["rescored_follow_up_3"]["header_value_recorded"])
        self.assertEqual(recs["AT07"]["status"], "pass")

    def test_retry_after_success_passes_with_informational_count(self):
        checks = self.with_review(self.review(
            http_429=2, samples=self.browser_samples(),
        ))
        checks.append(self.review(phase="join", http_429=0, samples=[]))
        recs = run.records(self.meta(), checks, self.reviewed_prior())
        self.assertEqual(recs["AT06"]["status"], "pass")
        self.assertEqual(recs["AT06"]["prior_failure_review"]["disposition"], "resolved")

    def test_missing_retry_after_is_a_finding(self):
        checks = self.with_review(self.review(http_429=2, samples=self.browser_samples(retry_after=False)))
        recs = run.records(self.meta(), checks, self.reviewed_prior())
        self.assertEqual(recs["AT06"]["status"], "blocked")
        self.assertEqual(recs["AT06"]["prior_failure_review"]["disposition"], "recurs")

    def test_failed_retry_is_a_finding(self):
        checks = self.with_review(self.review(http_429=2, samples=self.browser_samples(retry_succeeded=False)))
        recs = run.records(self.meta(), checks, self.reviewed_prior())
        self.assertEqual(recs["AT06"]["status"], "blocked")
        self.assertEqual(recs["AT06"]["prior_failure_review"]["disposition"], "recurs")

    def test_non_browser_class_is_a_finding(self):
        checks = self.with_review(self.review(http_429=2, samples=self.browser_samples(klass="anonymous")))
        recs = run.records(self.meta(), checks, self.reviewed_prior())
        self.assertEqual(recs["AT06"]["status"], "blocked")
        self.assertEqual(recs["AT06"]["prior_failure_review"]["disposition"], "recurs")

    def test_join_phase_429_is_a_finding(self):
        checks = self.with_review(self.review(phase="join", http_429=1, samples=[{
            "class": "browser", "method": "GET", "path": "/api/v1/admin/audit-events",
            "retry_after": True, "retry_succeeded": True,
        }]))
        recs = run.records(self.meta(), checks, self.reviewed_prior())
        self.assertEqual(recs["AT06"]["status"], "blocked")
        self.assertEqual(recs["AT06"]["prior_failure_review"]["disposition"], "recurs")

    def test_reviewed_429_prior_stays_blocked_when_429_recurs(self):
        checks = self.passing()
        for check in checks:
            if check["name"] == "rate_limit_review":
                check["status"] = "fail"
                check["detail"] = {"http_429": 2, "owner_if_recurs": "control", "limiter_class_for_prior_failures": "browser"}
        prior = {at: [] for at in run.REQUIRED}
        prior["AT06"] = [dict(name="audit_read", status="fail", run_id=run.REVIEWED["AT06"]["runs"][0]["run_id"])]
        recs = run.records(self.meta(), checks, prior)
        self.assertEqual(recs["AT06"]["status"], "blocked")
        self.assertEqual(recs["AT06"]["prior_failure_review"]["disposition"], "recurs")

    def test_unreviewed_prior_fail_still_blocks_a_clean_run(self):
        prior = {at: [] for at in run.REQUIRED}
        prior["AT06"] = [dict(name="audit_read", status="fail", run_id="some-other-run")]
        recs = run.records(self.meta(), self.passing(), prior)
        self.assertEqual(recs["AT06"]["status"], "blocked")
        self.assertEqual(recs["AT06"]["prior_failure_review"]["disposition"], "not_applied")

    def test_clone_parser_miss_does_not_block_after_the_check_passes(self):
        prior = {at: [] for at in run.REQUIRED}
        prior["AT06"] = [dict(name="probe_process", status="fail", run_id="1d5c02a4-b254-4a9b-b687-b1535428fb3f")]
        prior["AT07"] = [
            dict(name="cloned_identity_conflict", status="fail", run_id="1d5c02a4-b254-4a9b-b687-b1535428fb3f"),
            dict(name="probe_process", status="fail", run_id="1d5c02a4-b254-4a9b-b687-b1535428fb3f"),
        ]
        recs = run.records(self.meta(), self.passing(), prior)
        self.assertEqual(recs["AT06"]["status"], "pass")
        self.assertEqual(recs["AT07"]["status"], "pass")
        self.assertEqual(recs["AT07"]["prior_failure_review"]["disposition"], "resolved")
        self.assertIn("content-on-error", recs["AT07"]["prior_failure_review"]["what_changed"])

    def test_process_exit_from_review_signal_does_not_block_a_later_zero(self):
        prior = {at: [] for at in run.REQUIRED}
        prior["AT06"] = [
            dict(name="probe_process", status="fail", run_id="35e542f8-730e-4be0-8214-e4a9dd818c40"),
            dict(name="rate_limit_review", status="fail", run_id="35e542f8-730e-4be0-8214-e4a9dd818c40"),
        ]
        recs = run.records(self.meta(), self.passing(), prior)
        self.assertEqual(recs["AT06"]["status"], "pass")
        self.assertEqual(recs["AT06"]["prior_checks"][0]["status"], "fail")

    def test_at07_shared_process_failures_resolve_with_current_cross_case_evidence(self):
        prior = {at: [] for at in run.REQUIRED}
        prior["AT07"] = [dict(name="probe_process", status="fail", run_id=rid) for rid in (
            "bebfe230-26d4-4e3d-b052-81d258d04e07", "35e542f8-730e-4be0-8214-e4a9dd818c40")]
        recs = run.records(self.meta(), self.passing(), prior)
        self.assertEqual(recs["AT07"]["prior_failure_review"]["disposition"], "resolved")
        self.assertEqual(recs["AT07"]["prior_checks"], prior["AT07"])
        for name in ("probe_process", "audit_read", "incarnation_history", "rate_limit_review"):
            checks = [c for c in self.passing() if c["name"] != name]
            recs = run.records(self.meta(), checks, prior)
            self.assertEqual(recs["AT07"]["status"], "blocked", name)
            self.assertEqual(recs["AT07"]["prior_failure_review"]["disposition"], "not_applied", name)

    def test_at07_process_review_can_resolve_while_installer_is_blocked(self):
        prior = {at: [] for at in run.REQUIRED}
        prior["AT07"] = [dict(name="probe_process", status="fail", run_id="bebfe230-26d4-4e3d-b052-81d258d04e07")]
        checks = self.passing()
        for c in checks:
            if c["name"] == "installer_rerun":
                c["status"] = "blocked"
        recs = run.records(self.meta(), checks, prior)
        self.assertEqual(recs["AT07"]["status"], "blocked")
        self.assertEqual(recs["AT07"]["prior_failure_review"]["disposition"], "resolved")

    def test_read_bucket_429_success_uses_same_behavior_rule(self):
        checks = self.with_review(self.review(http_429=2, samples=self.browser_samples(klass="read")))
        recs = run.records(self.meta(), checks, self.reviewed_prior())
        self.assertEqual(recs["AT06"]["status"], "pass")

    def test_at07_unreviewed_process_exit_is_not_covered(self):
        prior = {at: [] for at in run.REQUIRED}
        prior["AT07"] = [dict(name="probe_process", status="fail", run_id="another-run")]
        recs = run.records(self.meta(), self.passing(), prior)
        self.assertEqual(recs["AT07"]["status"], "blocked")
        self.assertEqual(recs["AT07"]["prior_failure_review"]["disposition"], "not_applied")


if __name__ == "__main__":
    unittest.main()
