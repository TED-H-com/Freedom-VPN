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
        return checks

    def test_reviewed_429_prior_passes_when_current_run_has_zero(self):
        prior = {at: [] for at in run.REQUIRED}
        group = run.REVIEWED["AT06"]["runs"][0]
        prior["AT06"] = [dict(name=name, status="fail", run_id=group["run_id"]) for name in group["names"]]
        recs = run.records(self.meta(), self.passing(), prior)
        self.assertEqual(recs["AT06"]["status"], "pass")
        self.assertEqual(recs["AT06"]["prior_failure_review"]["disposition"], "resolved")
        self.assertIn("286eb57b44c2e783257767918680a0e1f097f106", recs["AT06"]["actual"])
        self.assertEqual(recs["AT07"]["status"], "pass")

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

    def test_process_exit_from_review_signal_does_not_block_a_later_zero(self):
        prior = {at: [] for at in run.REQUIRED}
        prior["AT06"] = [
            dict(name="probe_process", status="fail", run_id="35e542f8-730e-4be0-8214-e4a9dd818c40"),
            dict(name="rate_limit_review", status="fail", run_id="35e542f8-730e-4be0-8214-e4a9dd818c40"),
        ]
        recs = run.records(self.meta(), self.passing(), prior)
        self.assertEqual(recs["AT06"]["status"], "pass")
        self.assertEqual(recs["AT06"]["prior_checks"][0]["status"], "fail")


if __name__ == "__main__":
    unittest.main()
