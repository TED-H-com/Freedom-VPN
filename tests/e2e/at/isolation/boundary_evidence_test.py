import unittest
from review_boundary import timeline, qualify_scope


class BoundaryQualification(unittest.TestCase):
    def packet(self, n, ingress, egress):
        return dict(ingress_mono_ns=str(ingress),egress_mono_ns=str(egress),probe=dict(
            packet_id=n,phase='dense_boundary',result='ok' if egress else 'timeout',
            scheduled_mono_ns=str(ingress-2),send=dict(mono_ns=str(ingress-1))))

    def test_interleaving_positive_cannot_qualify(self):
        rows,errors=timeline(dict(deadline_mono_ns='1000',packets=[self.packet(1,100,120),self.packet(2,110,0)]))
        self.assertEqual(errors[0]['positive_packet'],1)
        self.assertEqual(errors[0]['next_packet'],2)
        self.assertEqual(rows[1]['result'],'timeout')

    def test_noninterleaving_preserves_drop_and_latency(self):
        rows,errors=timeline(dict(deadline_mono_ns='1000',packets=[self.packet(2,120,0),self.packet(1,100,110)]))
        self.assertEqual(errors,[])
        self.assertEqual(rows[0]['transit_ns'],'10')
        self.assertEqual(rows[0]['latency_ns'],'1')
        self.assertIsNone(rows[1]['transit_ns'])
        self.assertEqual(rows[1]['egress_mono_ns'],'0')

    def test_d142_earlier_positive_overlap_recorded_only(self):
        rows,pairs=timeline(dict(deadline_mono_ns='1000',packets=[
            self.packet(1,100,120),self.packet(2,110,125),self.packet(3,140,150),self.packet(4,160,0)]))
        start,last,first=qualify_scope(rows,pairs)
        self.assertEqual(start,2)
        self.assertFalse(pairs[0]['d142_judged'])
        self.assertEqual([p['packet_id'] for p in pairs[0]['packets']],[1,2])
        self.assertEqual((last['packet_id'],first['packet_id']),(3,4))

    def test_d142_drop_before_later_positive_is_in_scope(self):
        rows,pairs=timeline(dict(deadline_mono_ns='1000',packets=[
            self.packet(1,100,0),self.packet(2,110,140),self.packet(3,130,0)]))
        start,_,_=qualify_scope(rows,pairs)
        self.assertEqual(start,0)
        self.assertTrue(pairs[0]['d142_judged'])

    def test_d142_equal_boundary_is_not_before(self):
        rows,pairs=timeline(dict(deadline_mono_ns='1000',packets=[self.packet(1,100,120),self.packet(2,120,0)]))
        qualify_scope(rows,pairs)
        self.assertTrue(pairs[0]['d142_judged'])


class F2401Review(unittest.TestCase):
    """Run the real reviewer against immutable evidence in disk scratch space."""

    @classmethod
    def setUpClass(cls):
        from pathlib import Path
        cls.module = Path(__file__).resolve().parent
        cls.root = cls.module.parents[3]
        cls.history = cls.root/'docs/e2e/at/isolation/history'
        (cls.root/'.agent/tmp').mkdir(parents=True, exist_ok=True)

    def copied(self, source):
        import tempfile
        import shutil
        from pathlib import Path
        temp = tempfile.TemporaryDirectory(dir=self.root/'.agent/tmp', prefix='boundary-review.')
        self.addCleanup(temp.cleanup)
        root = Path(temp.name)
        for name in ('AT13.json', 'deadline-verdict.json', 'boundary-schedule.json'):
            shutil.copyfile(self.history/source/name, root/name)
        return root

    def execute(self, root):
        import json
        import subprocess
        result = subprocess.run(['python3', '-B', str(self.module/'review_boundary.py'), str(root)], capture_output=True, text=True)
        self.assertNotIn('Traceback', result.stderr)
        review = json.loads((root/'boundary-review.json').read_text())
        self.assertEqual(result.returncode, 0 if review['status'] == 'pass' else 1)
        return review

    def test_original_reviews_and_f2401(self):
        import json
        import os
        from pathlib import Path
        results = []
        for source in ('follow-up-24-run', 'follow-up-23-run', 'follow-up-25-runner-replay'):
            with self.subTest(source=source):
                root = self.copied(source)
                result = self.execute(root)
                verdict = json.loads((root/'deadline-verdict.json').read_text())
                self.assertEqual(result['unchanged_evaluator_passed'], verdict['passed'])
                self.assertEqual(result['unchanged_evaluator_errors'], verdict['errors'])
                if source == 'follow-up-24-run':
                    self.assertEqual(result['status'], 'fail')
                    self.assertFalse(result['schedule_qualified'])
                    self.assertIn('malformed plan: added_scheduled_mono_ns is null (F24-01 empty Go slice)', result['errors'])
                    self.assertIn('planned dense probes missing or extra', result['errors'])
                    self.assertTrue(result['unchanged_evaluator_passed'])
                else:
                    original = json.loads((self.history/source/'boundary-review.json').read_text())
                    self.assertEqual(result['status'], original['status'])
                    self.assertEqual(result['errors'], original['errors'])
                    self.assertEqual(result['deciding_pair'], original['deciding_pair'])
                    self.assertEqual(result['interleavings'], original['interleavings'])
                results.append(dict(source=source, status=result['status'], errors=result['errors'], schedule_qualified=result['schedule_qualified']))
                if os.environ.get('FVPN_AT_BOUNDARY_REPLAY_DIR'):
                    Path(os.environ['FVPN_AT_BOUNDARY_REPLAY_DIR'], source+'-review.json').write_text(json.dumps(result, indent=2)+'\n')
        print(json.dumps(results, indent=2))

    def test_null_empty_missing_nonlist_and_false(self):
        import json
        from nightly_summary import summarize
        mutations = [('null', None, 'is null'), ('empty', [], 'is empty'),
                     ('missing', None, 'missing'), ('non-list', {}, 'must be a list'),
                     ('qualified-false', False, 'qualified is false')]
        for name, value, reason in mutations:
            with self.subTest(name=name):
                root = self.copied('follow-up-23-run')
                plan = json.loads((root/'boundary-schedule.json').read_text())
                if name == 'missing':
                    del plan['added_scheduled_mono_ns']
                elif name == 'qualified-false':
                    plan['qualified'] = value
                else:
                    plan['added_scheduled_mono_ns'] = value
                (root/'boundary-schedule.json').write_text(json.dumps(plan))
                result = self.execute(root)
                self.assertEqual(result['status'], 'fail')
                self.assertFalse(result['schedule_qualified'])
                self.assertTrue(any(reason in e for e in result['errors']), result['errors'])
                self.assertTrue(result['unchanged_evaluator_passed'])
                # D-163 is not a boundary exemption: a review exit 1 stays red
                # even with all acceptance checks passing.
                for case in ('AT10', 'AT11', 'AT12', 'AT13'):
                    (root/(case+'.json')).write_text(json.dumps(dict(run_id=result['run_id'], checks=[dict(name='synthetic pass', status='pass')])))
                phase = 'python3 -B tests/e2e/at/isolation/review_boundary.py'
                job = summarize(root, [dict(name=phase, exit_code=1)], result['run_id'])
                self.assertEqual(job['status'], 'fail')
                self.assertEqual(job['errors'], [phase+': nonzero exit 1'])
                self.assertEqual(job['reviews'], [])

    def test_every_plan_field_missing_or_wrong_type(self):
        import copy
        import json
        from review_boundary import review
        root = self.copied('follow-up-23-run')
        original = json.loads((root/'boundary-schedule.json').read_text())
        # Every original plan field, nested clock/sample fields, and invalid
        # top-level JSON must fail without losing the observed packet verdict.
        plans = [[], None, False, 7, 'not an object']
        for key, value in original.items():
            missing = copy.deepcopy(original); del missing[key]; plans.append(missing)
            wrong = copy.deepcopy(original); wrong[key] = [] if isinstance(value, (str, dict, bool)) else {}; plans.append(wrong)
        for key in original['planned_at']:
            for value in (None, []):
                wrong = copy.deepcopy(original); wrong['planned_at'][key] = value; plans.append(wrong)
        for key in original['samples'][0]:
            missing = copy.deepcopy(original); del missing['samples'][0][key]; plans.append(missing)
            wrong = copy.deepcopy(original); wrong['samples'][0][key] = []; plans.append(wrong)
        wrong = copy.deepcopy(original); wrong['added_scheduled_mono_ns'] = [None]; plans.append(wrong)
        for i, plan in enumerate(plans):
            with self.subTest(i=i):
                (root/'boundary-schedule.json').write_text(json.dumps(plan))
                result = review(root)
                self.assertEqual(result['status'], 'fail')
                self.assertFalse(result['schedule_qualified'])
                self.assertTrue(result['errors'])
                self.assertTrue(result['unchanged_evaluator_passed'])
        (root/'boundary-schedule.json').write_text('{')
        self.assertIn('invalid JSON', self.execute(root)['errors'][0])
        (root/'boundary-schedule.json').unlink()
        self.assertIn('missing', self.execute(root)['errors'][0])

    def test_explicit_empty_plan_and_diagnostic_field_types(self):
        import copy
        import json
        from review_boundary import review
        root = self.copied('follow-up-24-run')
        plan = json.loads((root/'boundary-schedule.json').read_text())
        plan.update(qualified=False, added_scheduled_mono_ns=[],
                    unqualified_reason='empty added schedule: every candidate excluded by original cadence sends',
                    discounted_samples=[], spacing_unrounded_ns='33125363',
                    cadence_gaps_ns=['37000000', '63000000'],
                    cadence_gap_rule='gap >= 2 * spacing',
                    candidates=[dict(scheduled_mono_ns='125728478043437', excluded_by_original_sends=[
                        dict(scheduled_mono_ns='125728447630414', phase='arbitrary_37ms')])])
        (root/'boundary-schedule.json').write_text(json.dumps(plan))
        result = self.execute(root)
        self.assertFalse(result['schedule_qualified'])
        self.assertIn('unqualified plan: '+plan['unqualified_reason'], result['errors'])
        self.assertTrue(result['unchanged_evaluator_passed'])
        for key in ('unqualified_reason', 'discounted_samples', 'spacing_unrounded_ns', 'cadence_gaps_ns', 'cadence_gap_rule', 'candidates'):
            for missing in (True, False):
                wrong = copy.deepcopy(plan)
                if missing:
                    del wrong[key]
                else:
                    wrong[key] = None
                (root/'boundary-schedule.json').write_text(json.dumps(wrong))
                result = review(root)
                self.assertIn('malformed plan: plan.'+key+' missing or wrong type', result['errors'])
        for candidate in (None, {}, dict(scheduled_mono_ns='1', excluded_by_original_sends=[None])):
            wrong = copy.deepcopy(plan); wrong['candidates'] = [candidate]
            (root/'boundary-schedule.json').write_text(json.dumps(wrong))
            self.assertTrue(any('candidates[0]' in e for e in review(root)['errors']))
