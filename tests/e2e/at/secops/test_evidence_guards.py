"""Check committed secops evidence without git history or a live lane run."""
import hashlib
import json
import unittest

from run import PUBLIC, ROOT, scan


class CommittedEvidenceTests(unittest.TestCase):
    def _check_pin(self, path, expected):
        relative = path.relative_to(ROOT).as_posix()
        with self.subTest(file=relative):
            self.assertTrue(path.is_file(), f"missing pinned file: {relative}")
            self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(),
                             expected, f"SHA-256 mismatch: {relative}")

    def test_history_snapshot_pins(self):
        history = PUBLIC / "history"
        index = json.loads((history / "index.json").read_text(encoding="utf-8"))
        count = 0
        for name, entry in index.items():
            for pin in entry.get("snapshot_files", []):
                self._check_pin(history / name.replace("_", "-") / pin["file"],
                                pin["sha256"])
                count += 1
        self.assertGreater(count, 0, "history/index.json has no snapshot_files pins")
        print(f"Archive snapshot pins: {count} across {len(index)} entries")

    def test_evidence_hygiene(self):
        # Follow-up 13 also scanned newly committed archives. Scan all JSON and
        # Markdown recursively so shallow CI covers history without a branch diff.
        # No committed findings need an allow-list at the migration baseline.
        paths = sorted(path for path in PUBLIC.rglob("*")
                       if path.is_file() and path.suffix in (".json", ".md"))
        self.assertTrue(paths, "no secops evidence files to scan")
        for path in paths:
            with self.subTest(file=path.relative_to(ROOT).as_posix()):
                # scan returns only categories and line numbers, never secrets.
                self.assertEqual(scan(path.read_bytes()), [])
        print(f"Evidence hygiene: {len(paths)} JSON/Markdown files")


if __name__ == "__main__":
    unittest.main()
