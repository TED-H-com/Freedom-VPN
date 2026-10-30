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


if __name__ == "__main__":
    unittest.main()
