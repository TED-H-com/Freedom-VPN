"""Partial runs must never replace the root measurement summaries."""
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import followup12_records as records
from run import ROOT


class Followup12PublicationTests(unittest.TestCase):
    def test_both_completed_runs_required(self):
        # Explicit disk scratch even when invoked outside diskexec.py.
        scratch = ROOT / '.agent/tmp'
        scratch.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(dir=scratch) as directory:
            public = Path(directory)
            (public / 'summary.md').write_text('historical measurement')
            with patch.object(records, 'PUBLIC', public):
                for phase in ('transports', 'restore'):
                    with self.assertRaises(SystemExit):
                        records.main()
                    self.assertEqual((public / 'summary.md').read_text(), 'historical measurement')
                    (public / f'followup12-{phase}-run.json').write_text('{"finished_at":"TEST ONLY"}')
                    (public / f'followup12-{phase}-live.json').write_text('{}')
                records.completed_runs()
                (public / 'followup12-restore-run.json').write_text('{}')
                with self.assertRaises(SystemExit):
                    records.completed_runs()
