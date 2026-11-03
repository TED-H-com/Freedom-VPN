"""TEST ONLY synthetic evidence; disk scratch, no host or container mutation."""
import json
from pathlib import Path
import tempfile
import unittest

import followup43_results as results


class ResultTests(unittest.TestCase):
    def test_nearest_rank_and_empty_distribution(self):
        self.assertEqual(results.stats([])['p99_ns'], None)
        self.assertEqual(results.stats(range(1, 101)),
                         dict(count=100, p50_ns='50', p90_ns='90', p99_ns='99', max_ns='100'))

    def test_failed_apply_complete_guard_and_failed_boundary_are_preserved(self):
        with tempfile.TemporaryDirectory(dir='.agent/tmp') as directory:
            path = Path(directory)
            (path / 'suite').mkdir()
            (path / 'assembled').mkdir()
            row = dict(run_id='test-run', boot_id='test-boot', sequence='1', source='apply',
                       timing=dict(operation='apply', succeeded=False, deadline=dict(
                           install_ok=True, guard_released=True, render_start_ns='10',
                           install_start_ns='20', installed_by_ns='30', release_start_ns='40',
                           release_end_ns='50')))
            # Assembled is a subset of suite, never a second observation.
            for sub in ['suite', 'assembled']:
                (path / sub / 'operations.jsonl').write_text(json.dumps(row) + '\n')
            spans = [dict(kind='whole_step', start_ns='1', end_ns='80', serving_before=True,
                          serving_after=False, returned_error=True),
                     dict(kind='pre_apply_accounting', start_ns='2', end_ns='9', reached_apply=False)]
            (path / 'spans.jsonl').write_text(''.join(json.dumps(r) + '\n' for r in spans))
            got = results.evidence(path)
            self.assertEqual(got['snapshots'], 1)
            self.assertEqual(got['metrics']['outer_guard']['max_ns'], '30')
            self.assertEqual(got['metrics']['whole_serving_step']['max_ns'], '79')
            self.assertEqual(got['metrics']['pre_apply_accounting']['count'], 0)
            self.assertEqual([r['kind'] for r in got['failed_operations']],
                             ['operation', 'step_return', 'pre_apply_aborted'])
            row['timing']['succeeded'] = True
            (path / 'assembled/operations.jsonl').write_text(json.dumps(row) + '\n')
            with self.assertRaises(ValueError):
                results.evidence(path)


if __name__ == '__main__':
    unittest.main()
