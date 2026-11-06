"""Offline D-164 exporter proofs; no archived output or live campaign writes."""
import contextlib
import copy
import hashlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

import producer_timing
from producer_timing_test import fixture as timing_fixture
from serving_read import export_read, recovery_source_matches, utc_instant, within_window

HERE = Path(__file__).resolve().parent
REMOVAL = '2026-11-06T00:00:20Z'


def evidence(**changes):
    result = dict(read_at='2026-11-06T00:00:20.000000001Z', source='live',
                  candidate_generation='18446744073709551615', verified=True,
                  diagnostic='x'*499+'{')
    result.update(changes)
    return result


def report():
    projection = dict(reads=0, objects=0, truncated=False, tc=[])
    digest = hashlib.sha256(json.dumps(projection, sort_keys=True, separators=(',', ':')).encode()).hexdigest()
    return dict(report_id='TEST ONLY report', node_id='TEST ONLY node', node_incarnation_id='TEST ONLY incarnation',
                observed_at='2026-11-06T00:00:22Z', applied_config_generation='18446744073709551615',
                checks=[dict(name='qos', detail='source=live verified=true')],
                tc_evidence=dict(read_at='2026-11-06T00:00:20.1Z', phase='verify', verified=True,
                    candidate_generation='18446744073709551615', reads_total=0, objects_total=0,
                    truncated=False, attachments=[], rejected=None, error_code=None, sha256=digest))


class ServingRead(unittest.TestCase):
    def setUp(self):
        disk = HERE.parents[3]/'.agent/tmp'
        disk.mkdir(parents=True, exist_ok=True)
        temp = tempfile.TemporaryDirectory(dir=disk)
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)

    def write(self, name, value):
        (self.root/name).write_text(json.dumps(value))

    def review(self, reports):
        run = 'TEST ONLY run'
        self.write('AT10.json', dict(run_id=run))
        self.write('receiver-capture.json', dict(run_id=run, transactions=[
            dict(path='/api/v1/agent/observations', body=r, received_at=r.get('observed_at'), status=202) for r in reports]))
        self.write('foreign-redirect.json', dict(run_id=run, rows=[
            dict(label='before injection', observation=dict(applied_config_generation='1')),
            dict(label='inject clsact mirred', start='2026-11-06T00:00:10Z'),
            dict(label='remove all injected objects', start=REMOVAL, end=REMOVAL),
            dict(label='publish clean newer generation', at='2026-11-06T00:00:21Z')]))
        (self.root/'producer-operations.jsonl').write_text('')
        run = subprocess.run(['python3', '-B', str(HERE/'review_followup8.py'), str(self.root), run], capture_output=True, text=True)
        self.assertEqual(run.returncode, 0, run.stderr)
        return json.loads((self.root/'tc-evidence-index.json').read_text()), json.loads((self.root/'recovery-verdict.json').read_text())

    def test_structured_precedence_and_context_without_timestamp_equality(self):
        r = report()
        self.assertTrue(recovery_source_matches(r, 'live', REMOVAL))
        r['serving_read'] = None
        self.assertTrue(recovery_source_matches(r, 'live', REMOVAL))
        r['serving_read'] = evidence()
        self.assertTrue(recovery_source_matches(r, 'live', REMOVAL))  # different TC timestamp
        r['serving_read'] = evidence(source='apply')
        self.assertTrue(recovery_source_matches(r, 'apply', REMOVAL))
        self.assertFalse(recovery_source_matches(r, 'live', REMOVAL))
        for changes in [dict(verified=False), dict(candidate_generation='9007199254740993'),
                        dict(read_at='2026-11-06T00:00:19.999999999Z'), dict(verified='true'),
                        dict(verified=1), dict(source='invented'), dict(diagnostic='x'*501),
                        dict(read_at='2026-02-30T00:00:20Z'), dict(candidate_generation='18446744073709551616')]:
            with self.subTest(changes=changes):
                r['serving_read'] = evidence(**changes)
                self.assertFalse(recovery_source_matches(r, 'live', REMOVAL))
        for malformed in [{}, 'source=live verified=true', [], dict(evidence(), unexpected=True)]:
            r['serving_read'] = malformed
            self.assertFalse(recovery_source_matches(r, 'live', REMOVAL))
            self.assertEqual(export_read(dict(report=r, status=202))['interpretation_basis'], 'invalid_structured')
        r['serving_read'] = evidence(read_at='2026-11-06T00:00:20.000000001Z')
        self.assertFalse(recovery_source_matches(r, 'live', '2026-11-06T00:00:20.000000002Z'))
        self.assertEqual(utc_instant(REMOVAL), utc_instant('2026-11-06T00:00:20.000000000Z'))
        self.assertTrue(within_window(REMOVAL, REMOVAL))
        for invalid in (None, '', 'not-a-time', '2026-02-30T00:00:00Z'):
            self.assertFalse(within_window(REMOVAL, invalid))
            self.assertFalse(within_window(REMOVAL, REMOVAL, invalid))
        for checks in [None, {}, [None], [dict(name='qos', detail=1)]]:
            raw = dict(report(), checks=checks)
            self.assertEqual(export_read(dict(report=raw, status=422))['legacy_interpretation']['verified_sources'], [])

    def test_real_exporter_preserves_presence_rejection_no_tc_and_legacy(self):
        old = report()
        baseline, legacy_verdict = self.review([old])
        cases = [copy.deepcopy(old) for _ in range(7)]
        for i, r in enumerate(cases):
            r['report_id'] = str(i)
        cases[1]['serving_read'] = None
        cases[2]['serving_read'] = evidence(verified=False)
        cases[3]['serving_read'] = evidence(source='apply')
        cases[4]['serving_read'] = evidence(candidate_generation='9007199254740993', read_at='2026-11-05T00:00:00Z', verified=False)
        del cases[4]['tc_evidence']
        cases[5]['serving_read'] = {}
        cases[6]['serving_read'] = evidence()
        index, verdict = self.review(cases)
        self.assertEqual(len(index['reads']), 6)
        self.assertEqual(len(index['serving_reads']), 7)
        for read in index['reads']:
            self.assertIs(type(read['digest_matches']), bool)
            self.assertTrue(read['digest_matches'])
            self.assertEqual(read['tc_evidence'], baseline['reads'][0]['tc_evidence'])
        self.assertEqual(verdict['live_report_ids'], ['0', '1', '6'])
        self.assertEqual(verdict['apply_report_ids'], ['3'])
        self.assertEqual(verdict['assertions'], legacy_verdict['assertions'])
        self.assertFalse(verdict['passed'])  # a diagnostic alone supplies no recovery proof
        for r, exported in zip(cases, index['serving_reads']):
            self.assertEqual('serving_read' in r, 'serving_read' in exported)
            if 'serving_read' in r:
                self.assertEqual(exported['serving_read'], r['serving_read'])
            for name in ('report_id', 'node_id', 'node_incarnation_id'):
                self.assertEqual(exported[name], r[name])
            self.assertEqual(exported['status'], 202)
        self.assertEqual([r['presence'] for r in index['serving_reads'][:3]], ['absent', 'null', 'value'])
        self.assertEqual(index['serving_reads'][0]['interpretation_basis'], 'legacy_qos')
        self.assertEqual(index['serving_reads'][1]['interpretation_basis'], 'legacy_qos')
        self.assertEqual(index['serving_reads'][2]['interpretation_basis'], 'structured')
        self.assertEqual(index['serving_reads'][5]['interpretation_basis'], 'invalid_structured')
        # A rejected HTTP receipt remains part of the diagnostic export too.
        rejected = export_read(dict(report=cases[4], status=422))
        self.assertEqual(rejected['status'], 422)
        self.assertEqual(rejected['serving_read'], cases[4]['serving_read'])

    def test_actual_producer_timing_consumes_tc_only_index(self):
        index, _ = self.review([report()])
        digest = index['reads'][0]['tc_evidence']['sha256']
        rows = timing_fixture()
        rows[0]['tc_sha256'] = digest
        rows[0]['timing']['verifications'][0]['tc_sha256'] = digest
        for seq, start in [(2, 300), (3, 400)]:
            rows.append(dict(run_id='run', boot_id='boot', sequence=str(seq), source='renewal',
                anchor=dict(mono_before_ns=str(start+20), mono_after_ns=str(start+21), wall=REMOVAL),
                renewal=dict(start_ns=str(start), end_ns=str(start+10), new_lease=True),
                lease=dict(node_deadline_ns='1000')))
        (self.root/'producer-operations.jsonl').write_text(''.join(json.dumps(r)+'\n' for r in rows))
        (self.root/'agent.log.txt').write_text('')
        clock = dict(boot_id='boot', clock='CLOCK_MONOTONIC', time_namespace='TEST ONLY namespace',
                     offsets='monotonic 0 0\nboottime 0 0\n', measurement_run_id='run')
        self.write('producer-clock.json', clock)
        self.write('producer-clock-after-restart.json', clock)
        self.write('deadline-verdict.json', dict(deadline_mono_ns='1000'))
        self.write('AT10.json', dict(run_id='TEST ONLY run'))
        # Real packet validator consumes synthetic exact clock/bracket fixtures.
        def packets(name, at):
            probe = dict(packet_id=1, send=dict(mono_ns=str(at)), read_completion=dict(mono_ns=str(at+80)), result='ok', trace_clock=clock)
            capture = [dict(event='ready', trace_clock=clock),
                       dict(interface='fvmb0', packet_id=1, capture_read=dict(mono_ns=str(at+1))),
                       dict(interface='fvup0', packet_id=1, capture_read=dict(mono_ns=str(at+2))),
                       dict(event='complete', statistics_available=True, drops=0)]
            self.write(name+'-packets.json', capture)
            return probe
        renewal = [packets('renewal', 250), dict(packet_id=2, send=dict(mono_ns='450'), read_completion=dict(mono_ns='530'), result='ok', trace_clock=clock)]
        captures = json.loads((self.root/'renewal-packets.json').read_text())
        captures[1:1] = [dict(interface=i, packet_id=2, capture_read=dict(mono_ns=str(n))) for i,n in [('fvmb0',451),('fvup0',452)]]
        self.write('renewal-packets.json', captures)
        self.write('renewal-probes.json', renewal)
        self.write('whole-node-probes.json', dict(probes=[packets('whole-node', 1100)]))
        with contextlib.redirect_stdout(io.StringIO()):
            before = producer_timing.analyze(self.root)
            index['serving_reads'].append(export_read(dict(report=dict(report(), serving_read=evidence(verified=False)), status=422)))
            self.write('tc-evidence-index.json', index)
            after = producer_timing.analyze(self.root)
        self.assertEqual(before, after)
        self.assertEqual(after['wire_digests'], dict(reads=1, recompute_ok=1, matched_producer_digest=1))

    def test_real_recovery_windows_use_exact_lower_and_upper_boundaries(self):
        cases, expected_live, expected_rejected = [], [], []
        def add(label, observed=None, tc_at=None, rejected=False, missing=None):
            r = report()
            r['report_id'] = label
            if observed is not None:
                r['observed_at'] = observed
            if tc_at is not None:
                r['tc_evidence']['read_at'] = tc_at
            if rejected:
                r['tc_evidence']['error_code'] = 'TC_FOREIGN_FILTER'
            if missing == 'observed_at':
                del r['observed_at']
            if missing == 'tc_read_at':
                del r['tc_evidence']['read_at']
            cases.append(r)
        for label, value, qualifies in [
            ('before', '2026-11-06T00:00:20.999999999Z', False),
            ('equal', '2026-11-06T00:00:21Z', True),
            ('equal_fraction', '2026-11-06T00:00:21.000000000Z', True),
            ('after', '2026-11-06T00:00:21.000000001Z', True),
            ('invalid', 'not-a-time', False)]:
            add('observed_'+label, observed=value)
            if qualifies:
                expected_live.append('observed_'+label)
        for label, value, qualifies in [
            ('before', '2026-11-06T00:00:19.999999999Z', False),
            ('equal', REMOVAL, True),
            ('equal_fraction', '2026-11-06T00:00:20.000000000Z', True),
            ('after', '2026-11-06T00:00:20.000000001Z', True),
            ('invalid', '2026-02-30T00:00:20Z', False)]:
            add('tc_'+label, tc_at=value)
            if qualifies:
                expected_live.append('tc_'+label)
        for label, value, qualifies in [
            ('before_lower', '2026-11-06T00:00:09.999999999Z', False),
            ('equal_lower', '2026-11-06T00:00:10Z', True),
            ('after_lower', '2026-11-06T00:00:10.000000001Z', True),
            ('before_upper', '2026-11-06T00:00:19.999999999Z', True),
            ('equal_upper', REMOVAL, False),
            ('equal_upper_fraction', '2026-11-06T00:00:20.000000000Z', False),
            ('after_upper', '2026-11-06T00:00:20.000000001Z', False),
            ('invalid', 'not-a-time', False)]:
            add('rejected_'+label, observed=value, rejected=True)
            if qualifies:
                expected_rejected.append('rejected_'+label)
        add('missing_observed', missing='observed_at')
        add('missing_tc_time', missing='tc_read_at')
        _, verdict = self.review(cases)
        self.assertEqual(verdict['live_report_ids'], expected_live)
        self.assertEqual(verdict['rejected_report_ids'], expected_rejected)
        self.assertFalse(verdict['passed'])


if __name__ == '__main__':
    unittest.main()
