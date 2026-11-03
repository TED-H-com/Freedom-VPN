"""TEST ONLY synthetic monitors/evidence. No Docker, network or tmpfs scratch."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch, Mock

import followup44_timing as runner
import followup44_results as results


def sample(start=20, end=40, cores=0.6):
    row = dict(utc='2026-11-03T00:00:00Z', interval_start_ns=str(start), interval_end_ns=str(end),
               mono_ns=str(end), seconds=4, cpu_cores={'foreign': cores}, foreign_suites=[],
               foreign_containers=[], unreadable=[], foreign_cpu_processes=[
                   dict(pid=i, comm='codex' if i == 8 else 'foreign', ticks=i * 10) for i in range(1, 9)])
    runner.qualify(row)
    return row


class MonitorTests(unittest.TestCase):
    def test_mark_does_not_stop_and_keeps_largest_five(self):
        row = sample()
        self.assertTrue(row['qualified'])
        self.assertTrue(row['marked'])
        self.assertEqual([p['pid'] for p in row['top_foreign_processes']], [8, 7, 6, 5, 4])
        self.assertFalse(sample(cores=0.5)['marked'])

    def test_each_other_breach_still_disqualifies(self):
        for field, value in [('foreign_suites', ['1 suite']), ('foreign_containers', ['other']),
                             ('unreadable', ['proc']), ('seconds', 5.001)]:
            with self.subTest(field=field):
                row = sample()
                row[field] = value
                runner.qualify(row)
                self.assertFalse(row['qualified'])

    def test_no_new_foreign_process_exclusions(self):
        for name in ['codex', 'postgres', 'python3', 'go', 'cron']:
            self.assertEqual(runner.cpu_class(8, dict(ppid=1, comm=name, cgroup='0::/other'), set(), set()), 'foreign')
        for name in ['dockerd', 'containerd', 'containerd-shim-runc-v2', 'docker-proxy', 'runc']:
            self.assertEqual(runner.cpu_class(8, dict(ppid=1, comm=name, cgroup='0::/other'), set(), set()), 'container_runtime')

    def test_real_sample_records_the_clock_and_mark(self):
        monitor = runner.Monitor.__new__(runner.Monitor)
        monitor.label, monitor.at, monitor.at_ns, monitor.hz = 'A1', 10, 10_000_000_000, 100
        monitor.events = Mock()
        monitor.events.snapshot.return_value = ([], [], set())
        monitor.remembered, monitor.container_ids, monitor.unreadable = set(), set(), []
        monitor.previous = {80: dict(ppid=1, comm='codex', cgroup='', ticks=1, start=1)}
        current = {80: dict(monitor.previous[80], ticks=241)}
        with patch.object(runner, 'proc_snapshot', return_value=(current, [])), \
             patch.object(runner.os, 'getpid', return_value=99), \
             patch.object(runner, 'mono_ns', return_value=14_000_000_000), \
             patch.object(runner.subprocess, 'run', return_value=Mock(returncode=1, stdout='')), \
             patch.object(runner.subprocess, 'check_output', return_value='context'), \
             patch.object(runner, 'append') as record:
            row = monitor.sample()
        self.assertEqual(row['mono_ns'], '14000000000')
        self.assertEqual(row['interval_start_ns'], '10000000000')
        self.assertEqual(row['cpu_cores']['foreign'], 0.6)
        self.assertTrue(row['marked'] and row['qualified'])
        self.assertTrue(row['utc'])
        record.assert_called_once()

    def test_admission_requires_exact_free(self):
        with patch.dict(runner.os.environ, {'FVPN_HOLD_SH': 'TEST ONLY'}), \
             patch.object(runner, 'append'), patch('builtins.print'), \
             patch.object(runner.subprocess, 'run') as run:
            for code, stdout, expected in [(0, 'free\n', True), (0, 'busy', False), (125, 'free', False)]:
                run.return_value = Mock(returncode=code, stdout=stdout)
                self.assertEqual(runner.admission(), expected)

    def test_one_clock_requires_bracket_namespace_and_zero_offsets(self):
        row = dict(host_before_ns='1', host_after_ns='5', host_time_namespace='time:[1]',
                   host_initial_time_namespace='time:[1]', probe=dict(before_ns='2', mono_ns='3', after_ns='4',
                   time_namespace='time:[1]', timens_offsets='monotonic 0 0\nboottime 0 0'))
        self.assertTrue(runner.clock_matches(row))
        for key, value in [('mono_ns', '6'), ('time_namespace', 'time:[2]'),
                           ('timens_offsets', 'monotonic 1 0\nboottime 0 0')]:
            self.assertFalse(runner.clock_matches(dict(row, probe=dict(row['probe'], **{key: value}))))

    def test_clock_mismatch_raises_before_suite(self):
        with tempfile.TemporaryDirectory(dir='.agent/tmp') as directory, \
             patch.object(runner, 'mono_ns', side_effect=[1, 10] * 3), \
             patch.object(runner.os, 'readlink', return_value='time:[1]'), \
             patch.object(runner.subprocess, 'check_output', return_value=json.dumps(dict(
                 before_ns='2', mono_ns='3', after_ns='4', time_namespace='time:[2]',
                 timens_offsets='monotonic 0 0\nboottime 0 0'))):
            with self.assertRaisesRegex(RuntimeError, 'clock mismatch'):
                runner.measure_clock(Path(directory))
            self.assertFalse(results.read(Path(directory) / 'clock.json')['matched'])


class FilteringTests(unittest.TestCase):
    def test_strict_overlap_boundaries(self):
        marked = [sample()]
        for start, end, count in [(1, 20, 0), (40, 50, 0), (19, 21, 1), (39, 41, 1), (1, 60, 1), (25, 30, 1)]:
            self.assertEqual(len(results.overlapping(start, end, marked)), count)

    def test_all_five_spans_filter_and_failed_operations_remain(self):
        with tempfile.TemporaryDirectory(dir='.agent/tmp') as directory:
            path = Path(directory)
            (path / 'suite').mkdir()
            op = dict(run_id='TEST ONLY', boot_id='TEST ONLY', sequence='1', source='apply',
                      timing=dict(operation='apply', succeeded=False, start_ns='1', end_ns='90',
                      deadline=dict(install_ok=True, guard_released=True, render_start_ns='10',
                      install_start_ns='20', installed_by_ns='30', release_start_ns='40', release_end_ns='50')))
            (path / 'suite/operations.jsonl').write_text(json.dumps(op) + '\n')
            spans = [dict(kind='whole_step', start_ns='1', end_ns='80', serving_before=True,
                          serving_after=False, returned_error=True),
                     dict(kind='pre_apply_accounting', start_ns='2', end_ns='27', reached_apply=True)]
            (path / 'spans.jsonl').write_text(''.join(json.dumps(r) + '\n' for r in spans))
            kept = results.evidence(path, [sample(80, 90)])
            excluded = results.evidence(path, [sample(25, 26)])
            for kind in results.METRICS:
                self.assertEqual(kept['metrics'][kind]['count'], 1)
                self.assertEqual(excluded['metrics'][kind]['count'], 0)
                self.assertEqual(excluded['excluded_counts'][kind], 1)
            self.assertEqual(len(excluded['failed_operations']), 2)
            self.assertTrue(all(f['marked'] for f in excluded['failed_operations']))

    def test_pool_threshold_is_inclusive_and_qualification_required(self):
        for kept, removed, qualified, expected in [(5, 5, True, True), (4, 6, True, False),
                (10, 0, False, False), (0, 0, True, False)]:
            measurement = dict(metrics={'whole_serving_step': dict(count=kept)}, excluded_counts={'whole_serving_step': removed})
            self.assertEqual(results.pool_decision(dict(qualified=qualified), measurement)['pooled'], expected)

    def test_failure_uses_operation_overlap_not_test_duration(self):
        for start, end, expected in [(10, 30, True), (1, 20, False), (40, 50, False)]:
            failure = results.classify_failure('TestLabD087_D104_TEST', [dict(start_ns=str(start), end_ns=str(end))], [sample()])
            self.assertEqual(failure['marked'], expected)
            self.assertEqual(bool(failure['operations'][0]['marked_intervals']), expected)
        self.assertFalse(results.classify_failure('TestLabD087_D104_TEST', [], [sample()])['marked'])

    def test_same_b_test_twice_in_pooled_legs_is_failure_even_marked(self):
        base = dict(arm='B', pooled=True, failed_scenarios=['TestLabD087_D104_TEST'], marked=True)
        self.assertEqual(results.repeated_b_failures([dict(base, leg='B1'), dict(base, leg='B2')]), ['TestLabD087_D104_TEST'])
        self.assertEqual(results.repeated_b_failures([base, dict(base, pooled=False)]), [])
        self.assertEqual(results.repeated_b_failures([base, dict(base, arm='A')]), [])


if __name__ == '__main__':
    unittest.main()
