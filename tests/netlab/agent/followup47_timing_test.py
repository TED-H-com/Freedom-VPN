"""TEST ONLY synthetic monitors/evidence. No Docker, network or tmpfs scratch."""
import json
from pathlib import Path
import tempfile
import os
import subprocess
import sys
import unittest
from unittest.mock import patch, Mock

import followup47_timing as runner
import followup47_results as results

SCRATCH = Path(__file__).resolve().parents[3] / ".agent/tmp"
SCRATCH.mkdir(parents=True, exist_ok=True)


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
                run.return_value = Mock(returncode=code, stdout=stdout, stderr='')
                self.assertEqual(runner.admission(), expected)

    def test_admitted_tool_stdout_remains_machine_readable(self):
        # Real wrapper composition against TEST ONLY executables, no Docker.
        with tempfile.TemporaryDirectory(dir=SCRATCH) as directory:
            root = Path(directory).resolve()
            real, wrapped = root / 'real', root / 'wrapped'
            real.mkdir()
            wrapped.mkdir()
            hold = root / 'hold.sh'
            hold.write_text("#!/bin/sh\nprintf 'free\\n'\n")
            fake = real / 'docker'
            fake.write_text("#!/bin/sh\nprintf '%s\\n' '{\"ok\":true}'\n")
            fake.chmod(0o755)
            (wrapped / 'docker').symlink_to(Path(runner.__file__).resolve())
            env = dict(os.environ, FVPN_HOLD_SH=str(hold), F47_TIMING_ROOT=str(root),
                       F47_ORIGINAL_PATH=str(real), F47_LEG=str(root / 'leg'))
            got = subprocess.run([sys.executable, '-B', str(wrapped / 'docker'), 'inspect'],
                                 env=env, capture_output=True, text=True, check=True)
            self.assertEqual(json.loads(got.stdout), {'ok': True})
            self.assertEqual(got.stderr, '')
            receipt = results.lines(root / 'leg/admission.jsonl')[0]
            self.assertTrue(receipt['admitted'])
            self.assertEqual(receipt['stdout'], 'free\n')

    def clock_row(self):
        return dict(host_before_ns='1', host_after_ns='5', host_time_namespace='time:[1]',
                    host_timens_offsets='monotonic 0 0\nboottime 0 0',
                    probe=dict(before_ns='2', mono_ns='3', after_ns='4',
                               time_namespace='time:[1]', timens_offsets='monotonic 0 0\nboottime 0 0'))

    def test_same_namespace_with_aligned_brackets_matches(self):
        self.assertTrue(runner.clock_matches(self.clock_row()))

    def test_different_zero_offset_namespace_with_aligned_brackets_matches(self):
        row = self.clock_row()
        row['probe']['time_namespace'] = 'time:[2]'
        self.assertTrue(runner.clock_matches(row))

    def test_any_nonzero_offset_on_either_side_does_not_match(self):
        for namespace in ['time:[1]', 'time:[2]']:
            for side in ['host', 'probe']:
                for field in [1, 2, 4, 5]:
                    for value in ['1', '-1']:
                        with self.subTest(namespace=namespace, side=side, field=field, value=value):
                            row = self.clock_row()
                            row['probe']['time_namespace'] = namespace
                            offsets = ['monotonic', '0', '0', 'boottime', '0', '0']
                            offsets[field] = value
                            target = row if side == 'host' else row['probe']
                            target['host_timens_offsets' if side == 'host' else 'timens_offsets'] = ' '.join(offsets)
                            self.assertFalse(runner.clock_matches(row))

    def test_each_misaligned_bracket_does_not_match(self):
        for side, key, value in [('host', 'host_before_ns', '3'), ('probe', 'before_ns', '4'),
                                 ('probe', 'mono_ns', '5'), ('probe', 'after_ns', '6')]:
            with self.subTest(key=key):
                row = self.clock_row()
                (row if side == 'host' else row['probe'])[key] = value
                self.assertFalse(runner.clock_matches(row))
        row = self.clock_row()
        row.update(host_before_ns='3', host_after_ns='3')
        row['probe'].update(before_ns='3', mono_ns='3', after_ns='3')
        self.assertTrue(runner.clock_matches(row))

    def test_scratch_independent_of_working_directory(self):
        previous = Path.cwd()
        with tempfile.TemporaryDirectory(dir=SCRATCH) as elsewhere:
            try:
                os.chdir(elsewhere)
                with tempfile.TemporaryDirectory(dir=SCRATCH) as directory:
                    self.assertEqual(Path(directory).parent, SCRATCH)
                    self.assertTrue(Path(directory).is_absolute())
            finally:
                os.chdir(previous)

    def test_clock_mismatch_raises_before_suite(self):
        with tempfile.TemporaryDirectory(dir=SCRATCH) as directory, \
             patch.object(runner, 'mono_ns', side_effect=[1, 10] * 3), \
             patch.object(runner.os, 'readlink', return_value='time:[1]'), \
             patch.object(runner.subprocess, 'check_output', return_value=json.dumps(dict(
                 before_ns='2', mono_ns='11', after_ns='12', time_namespace='time:[2]',
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
        with tempfile.TemporaryDirectory(dir=SCRATCH) as directory:
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


class WrapperRegressionTests(unittest.TestCase):
    def probe(self, version, mode, stdout=b'true\n', stderr=b'', admitted=True, tool='docker'):
        with tempfile.TemporaryDirectory(dir=SCRATCH) as directory:
            root = Path(directory)
            real, wrapped = root / 'real', root / 'wrapped'
            real.mkdir()
            wrapped.mkdir()
            hold = root / 'hold.sh'
            hold.write_text("#!/bin/sh\nprintf '" + ('free' if admitted else 'busy') + "\\n'\n" +
                            ('exit 0\n' if admitted else 'exit 125\n'))
            stub = real / tool
            stub.write_text('#!' + sys.executable + '\nimport os, sys\n' +
                            'os.write(1, ' + repr(stdout) + ')\nos.write(2, ' + repr(stderr) + ')\n')
            stub.chmod(0o755)
            wrapper = wrapped / tool
            wrapper.symlink_to(Path(__file__).with_name('followup' + str(version) + '_timing.py'))
            env = dict(os.environ, FVPN_HOLD_SH=str(hold))
            env.update({f'F{version}_TIMING_ROOT': str(root), f'F{version}_ORIGINAL_PATH': str(real),
                        f'F{version}_LEG': str(root / 'leg')})
            command = [sys.executable, '-B', str(wrapper), 'test' if tool == 'go' else 'inspect']
            if mode == 'shell':
                command = ['bash', '-c', 'value=$("$@"); status=$?; printf "%s" "$value"; exit "$status"',
                           'TEST ONLY', *command]
            got = subprocess.run(command, env=env, stdout=subprocess.PIPE,
                                 stderr=subprocess.STDOUT if mode == 'combined' else subprocess.PIPE)
            receipts = results.lines(root / 'leg/admission.jsonl')
            return got, receipts

    def test_actual_wrapper_each_frozen_read_mode_and_old_regression(self):
        for mode in ['combined', 'stdout', 'shell']:
            for payload in [b'true\n', b'{"ok":true}\n', b'one\ntwo\n']:
                expected = payload.rstrip(b'\n') if mode == 'shell' else payload
                with self.subTest(mode=mode, payload=payload):
                    new, receipts = self.probe(47, mode, payload)
                    self.assertEqual(new.returncode, 0)
                    self.assertEqual(new.stdout, expected)
                    self.assertIn(new.stderr, (None, b''))
                    self.assertEqual(len(receipts), 1)
                    receipt = receipts[0]
                    self.assertEqual(receipt['command'], ['bash', '$FVPN_HOLD_SH'])
                    self.assertEqual(receipt['exit'], 0)
                    self.assertEqual(receipt['stdout'], 'free\n')
                    self.assertEqual(receipt['stderr'], '')
                    self.assertLessEqual(receipt['started'], receipt['completed'])
                    old, _ = self.probe(45, mode, payload)
                    # 45 corrupts combined bytes. Separate stdout/shell data
                    # already passed, but their stderr still violates silence.
                    if mode == 'combined':
                        self.assertNotEqual(old.stdout, expected)
                        self.assertEqual(old.stdout, b'free\n' + payload)
                    else:
                        self.assertEqual(old.stdout, expected)
                        self.assertEqual(old.stderr, b'free\n')

    def test_refusal_is_receipted_nonzero_and_never_executes_tool(self):
        for tool in ['docker', 'make', 'go']:
            got, receipts = self.probe(47, 'combined', admitted=False, tool=tool)
            self.assertEqual(got.returncode, 125)
            self.assertEqual(got.stdout, b'')
            self.assertFalse(receipts[0]['admitted'])
            self.assertEqual(receipts[0]['stdout'], 'busy\n')

    def test_tool_stderr_is_preserved_without_admission_bytes(self):
        got, _ = self.probe(47, 'stdout', stderr=b'TEST ONLY tool diagnostic\n')
        self.assertEqual(got.stdout, b'true\n')
        self.assertEqual(got.stderr, b'TEST ONLY tool diagnostic\n')


class SuiteProgramTests(unittest.TestCase):
    def processes(self, commands):
        return {pid: dict(ppid=parent, comm=Path(argv[0]).name, ticks=7, start=pid, argv=argv)
                for pid, parent, argv in commands}

    def test_required_process_tables(self):
        shell = ['/bin/bash', '-lc', 'bash "$FVPN_HOLD_SH"; sleep 60; make -f mk/at-isolation.mk at-isolation']
        waiting = [(20, 1, shell), (21, 20, ['sleep', '60'])]
        cases = [
            (waiting, []),
            (waiting + [(22, 20, ['make', '-f', 'mk/at-isolation.mk', 'at-isolation'])], [22]),
            ([(20, 1, ['bash', 'bin/gates.sh', 'w4-at-members'])], [20]),
            ([(20, 1, ['go', 'test', './tests/handover/...'])], [20]),
            ([(20, 1, ['go', 'build', './...'])], []),
            ([(20, 1, ['grep', '-E', r'[g]ates\.sh|[m]k/at-']),
              (21, 1, ['pgrep', '-af', 'gates.sh|mk/at-'])], []),
            ([(20, 1, ['/bin/bash', '-c', 'python3 checks.py go test ./...'])], []),
            ([(20, 1, ['python3', '-c', 'go test']), (21, 1, ['bash', '-', 'gates.sh']),
              (22, 1, ['vim', 'gates.sh']), (23, 1, ['codex', 'go test'])], []),
            ([(20, 1, ['bash', '-e', 'bin/gates.sh'])], [20]),
        ]
        for commands, expected in cases:
            with self.subTest(commands=commands):
                table = self.processes(commands)
                suites = runner.suite_processes(table, {10}, 10)
                self.assertEqual([s['pid'] for s in suites], expected)
                self.assertTrue(all(set(s) == {'pid', 'comm', 'ticks'} for s in suites))

    def test_own_ancestors_and_descendants_do_not_count_but_cpu_stays_foreign(self):
        table = self.processes([(1, 0, ['bash', 'gates.sh']), (8, 1, ['go', 'test']),
            (9, 8, ['make', 'node-test-netlab']), (10, 9, ['python3', 'runner.py']),
            (11, 10, ['go', 'test']), (12, 11, ['make', 'netlab-test']),
            (20, 1, ['make', '-f', 'mk/at-isolation.mk'])])
        own = runner.descendants(table, 10, set())
        self.assertEqual([s['pid'] for s in runner.suite_processes(table, own, 10)], [20])
        self.assertEqual(runner.cpu_class(9, dict(ppid=8, comm='codex', cgroup=''), own, set()), 'foreign')


class PreflightTests(unittest.TestCase):
    def check(self, clear_at=None, bad=None, deadline=1800):
        tick = [0.0]
        def sleep(seconds):
            self.assertGreaterEqual(seconds, 0)
            tick[0] += seconds
        monitor = Mock()
        monitor.unreadable = []
        monitor.at = 0
        def sample_now():
            row = sample()
            row['seconds'] = 4
            row['foreign_suites'] = [] if clear_at is not None and tick[0] >= clear_at else [dict(pid=20, comm='make', ticks=7)]
            row['foreign_containers'] = [] if not row['foreign_suites'] else [dict(id='TEST ONLY', name='fvpn-test')]
            if bad:
                row.update(bad)
            runner.qualify(row)
            return row
        monitor.sample.side_effect = sample_now
        with patch.object(runner.time, 'monotonic', side_effect=lambda: tick[0]), \
             patch.object(runner.time, 'sleep', side_effect=sleep), \
             patch.object(runner, 'mono_ns', side_effect=lambda: int(tick[0] * 1e9)), \
             patch.object(runner, 'proc_snapshot', return_value=({}, [])), \
             patch.object(runner, 'save') as save:
            reason = runner.preflight_wait(monitor, SCRATCH, deadline)
        return reason, save.call_args.args[1]

    def test_wait_clears_and_preserves_all_present_activity(self):
        reason, row = self.check(clear_at=34)
        self.assertEqual(reason, '')
        self.assertEqual([r['elapsed_seconds'] for r in row['checks']], [4, 19, 34])
        self.assertTrue(row['waited'] and row['cleared'])
        self.assertTrue(row['checks'][0]['foreign_suites'])
        self.assertTrue(row['checks'][0]['foreign_containers'])
        self.assertFalse(row['checks'][-1]['foreign_suites'])

    def test_wait_expiry_is_not_started_and_bounded(self):
        reason, row = self.check()
        self.assertEqual(reason, 'not started: preflight wait expired')
        self.assertEqual(row['seconds'], 600)
        self.assertFalse(row['cleared'])
        self.assertTrue(row['checks'][-1]['foreign_suites'])

    def test_clear_preflight_does_not_wait_or_ignore_time_limit(self):
        reason, row = self.check(clear_at=0)
        self.assertEqual(reason, '')
        self.assertFalse(row['waited'])
        reason, _ = self.check(clear_at=34, deadline=450)
        self.assertIn('less than seven minutes', reason)

    def test_non_activity_failure_is_immediate(self):
        for bad in [dict(unreadable=['proc']), dict(seconds=5.1)]:
            reason, row = self.check(bad=bad)
            self.assertEqual(reason, 'preflight monitoring failed')
            self.assertEqual(row['seconds'], 4)

    def test_expired_preflight_does_not_launch_suite_or_use_round(self):
        with tempfile.TemporaryDirectory(dir=SCRATCH) as directory:
            root = Path(directory)
            with patch.object(runner, 'ROOT', root), patch.object(runner, 'OUT', root), \
                 patch.object(runner, 'admission', return_value=True), \
                 patch.object(runner, 'Monitor') as monitor, \
                 patch.object(runner, 'preflight_wait', return_value='not started: preflight wait expired'), \
                 patch.object(runner.subprocess, 'Popen') as launch, patch('builtins.print'):
                row = runner.run_leg('A1', {'A': 'TEST ONLY'}, 1, 1800)
                self.assertTrue(row['not_started'])
                self.assertNotIn('started', row)
                launch.assert_not_called()
                monitor.return_value.events.close.assert_called_once()


class DryTests(unittest.TestCase):
    def test_dry_requires_all_scenarios_spans_and_qualification(self):
        good = dict(qualified=True, exit=0, passed_scenarios=list(range(13)), retained_whole_steps=5)
        self.assertTrue(runner.dry_pass(good))
        for change in [dict(qualified=False), dict(exit=2), dict(passed_scenarios=list(range(12))),
                       dict(retained_whole_steps=0)]:
            self.assertFalse(runner.dry_pass(dict(good, **change)))
        m = dict(metrics={'whole_serving_step': {'count': 5}}, excluded_counts={'whole_serving_step': 0})
        self.assertFalse(results.pool_decision(dict(good, dry=True), m)['pooled'])

    def test_reservation_requires_current_hash_both_arms_and_receipts(self):
        with tempfile.TemporaryDirectory(dir=SCRATCH) as directory:
            root = Path(directory)
            for arm in ['A', 'B']:
                (root / arm).mkdir()
                (root / arm / 'admission.jsonl').write_text('{}\n')
            rows = [dict(arm=a, evidence=a, qualified=True, exit=0, passed_scenarios=list(range(13)),
                         retained_whole_steps=5) for a in ['A', 'B']]
            record = dict(legs=rows, passed=True, runner_sha256='current')
            with patch.object(runner, 'ROOT', root), patch.object(runner, 'OUT', root), \
                 patch.object(runner, 'file_hash', return_value='current'):
                def check(value):
                    (root / 'dry1.json').write_text(json.dumps(value))
                    runner.require_dry_pass()
                check(record)
                for bad in [dict(record, runner_sha256='old'), dict(record, passed=False),
                            dict(record, legs=rows[:1])]:
                    with self.assertRaises(RuntimeError):
                        check(bad)
                (root / 'A/admission.jsonl').unlink()
                with self.assertRaises(RuntimeError):
                    check(record)


class ReservationTests(unittest.TestCase):
    def test_dry_is_reserved_waits_60_and_uses_same_finite_deadline(self):
        with tempfile.TemporaryDirectory(dir=SCRATCH) as directory:
            root = Path(directory)
            def leg(label, window, number, deadline):
                self.assertEqual(deadline, 1900)
                self.assertEqual(number, 'dry1')
                return dict(arm=label[0], evidence=label, qualified=True, exit=0,
                            passed_scenarios=list(range(13)), retained_whole_steps=2)
            with patch.object(runner, 'ROOT', root), patch.object(runner, 'OUT', root), \
                 patch.object(runner, 'wait_after_release') as spacing, \
                 patch.object(runner, 'reserve', return_value=100) as reserve, \
                 patch.object(runner, 'release') as release, \
                 patch.object(runner.time, 'monotonic', return_value=160), \
                 patch.object(runner.time, 'sleep') as sleep, \
                 patch.object(runner, 'run_leg', side_effect=leg) as launch, \
                 patch.object(runner, 'save'), patch('builtins.print'):
                runner.run_dry({'A': 'a', 'B': 'b'}, 1)
                reserve.assert_called_once_with('dry1')
                sleep.assert_called_once_with(60)
                self.assertEqual([call.args[0] for call in launch.call_args_list], ['A-dry1', 'B-dry1'])
                release.assert_called_once_with('dry1')
                spacing.assert_called_once()
                with self.assertRaises(RuntimeError):
                    runner.run_dry({}, 3)

    def test_reservation_refusals_recorded_for_bounded_120_minutes(self):
        tick = [0.0]
        with patch.dict(runner.os.environ, {'FVPN_HOLD_SH': 'TEST ONLY'}), \
             patch.object(runner.time, 'monotonic', side_effect=lambda: tick[0]), \
             patch.object(runner.time, 'sleep', side_effect=lambda n: tick.__setitem__(0, tick[0] + n)), \
             patch.object(runner.subprocess, 'run', return_value=Mock(returncode=1, stdout='busy', stderr='')) as run, \
             patch.object(runner, 'append') as record, patch('builtins.print'):
            self.assertIsNone(runner.reserve('dry1'))
            self.assertEqual(tick[0], 7200)
            self.assertEqual(record.call_count, 240)
            self.assertEqual(run.call_args.args[0][-1], 'agent-core f47 D-150 dry pair')


class RefusedLegTests(unittest.TestCase):
    def test_any_wrapper_refusal_stops_leg_even_if_consumer_ignores_exit(self):
        with tempfile.TemporaryDirectory(dir=SCRATCH) as directory:
            path = Path(directory)
            self.assertFalse(runner.admission_refused(path))
            runner.append(path / 'admission.jsonl', {'admitted': True})
            self.assertFalse(runner.admission_refused(path))
            runner.append(path / 'admission.jsonl', {'admitted': False})
            self.assertTrue(runner.admission_refused(path))


if __name__ == '__main__':
    unittest.main()
