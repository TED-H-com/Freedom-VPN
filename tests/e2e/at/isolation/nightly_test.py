import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from nightly_summary import summarize
from suite_admission import classify


class Nightly(unittest.TestCase):
    def test_unexecuted_expiry_never_reads_inherited_evidence(self):
        for script,output in [('review_boundary.py','boundary-review.json'),('review_followup10.py','producer-timing-unavailable.json')]:
            with self.subTest(script=script), tempfile.TemporaryDirectory() as tmp:
                root=Path(tmp)
                (root/'AT13.json').write_text(json.dumps(dict(run_id='current',checks=[dict(name='short whole-node lease expiry',status='blocked',actual='route unavailable',owner='fixture')])))
                for name in ('deadline-verdict.json','boundary-schedule.json','producer-operations.jsonl'):
                    (root/name).write_text('invalid inherited data must never be read')
                result=subprocess.run(['python3','-B',str(Path(__file__).with_name(script)),tmp],text=True,capture_output=True)
                self.assertNotEqual(result.returncode,0)
                self.assertEqual((result.stdout+result.stderr).strip(),'blocked: expiry not executed; refusing inherited boundary evidence')
                record=json.loads((root/output).read_text())
                self.assertEqual(record['run_id'],'current')
                self.assertEqual(record['status'],'blocked')
                self.assertFalse(record['inherited_evidence_used'])

    def test_summary_preserves_blocks_unblocks_and_errors(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            for case in ('AT10','AT11','AT12','AT13'):
                (root/(case+'.json')).write_text(json.dumps(dict(run_id='fresh',checks=[dict(name='normal',status='pass')])))
            operator=dict(name='IPv6 platform matrix',status='blocked',actual='operator devices required',owner='acceptance')
            p=root/'AT11.json';p.write_text(json.dumps(dict(run_id='fresh',checks=[operator])))
            r=summarize(root,[], 'fresh')
            self.assertEqual(r['status'],'pass')
            self.assertEqual(r['rows'][1]['status'],'blocked')
            self.assertEqual(json.loads(p.read_text())['checks'][0]['status'],'blocked')
            operator['status']='pass';p.write_text(json.dumps(dict(run_id='fresh',checks=[operator])))
            self.assertIn('unblocked',summarize(root,[],'fresh')['rows'][1]['mapping'])
            self.assertEqual(summarize(root,[dict(name='traceback',exit_code=1)],'fresh')['status'],'fail')
            operator.update(name='short whole-node lease expiry',status='blocked');p.write_text(json.dumps(dict(run_id='fresh',checks=[operator])))
            self.assertEqual(summarize(root,[],'fresh')['status'],'fail')
            self.assertEqual(summarize(root,[],'different')['status'],'fail')

    def test_real_processes_only(self):
        self.assertIsNone(classify(['bash','-lc','make -f mk/at-members.mk at-members']))
        self.assertEqual(classify(['make','-f','mk/at-members.mk','at-members']),'members assembled')
        self.assertEqual(classify(['make','-f','mk/node.mk','node-test-netlab']),'agent-core netlab')
        self.assertEqual(classify(['go','test','-tags','netlab','./tests/netlab'],'/work/w1-netlab'),'netlab 27')
        self.assertIsNone(classify(['go','test','-tags','netlab','./tests/netlab'],'/work/w4-at-isolation'))

    def test_unexecuted_foreign_redirect_writes_unavailable_verdict_without_traceback(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            (root/'receiver-capture.json').write_text(json.dumps(dict(run_id='775bc692-e6ef-4490-8056-ba6cfa818b69',transactions=[])))
            result=subprocess.run(['python3','-B',str(Path(__file__).with_name('review_followup8.py')),tmp],text=True,capture_output=True)
            self.assertEqual(result.returncode,0,result.stderr)
            verdict=json.loads((root/'recovery-verdict.json').read_text())
            self.assertFalse(verdict['passed'])
            self.assertEqual(verdict['status'],'unavailable')
            self.assertEqual(verdict['reason'],'injection not reached')
            self.assertNotIn('injected_creator', verdict)

    def test_d163_retirement_rejects_recurrence_and_other_failures(self):
        import copy
        import shutil
        from nightly_summary import DRIVER_PHASE, HOLD_REVIEW_PHASE
        source=Path(__file__).resolve().parents[4]/'docs/e2e/at/isolation/history/follow-up-24-nightly-41836274140'
        original=json.loads((source/'scheduled-job.json').read_text())
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            for name in ('AT10.json','AT11.json','AT12.json','AT13.json','populated-holds-review.json'):
                shutil.copyfile(source/name,root/name)
            review_path=root/'populated-holds-review.json'
            raw=review_path.read_bytes()
            result=summarize(root,original['phases'],original['run_id'])
            self.assertEqual(result['status'],'fail')
            self.assertEqual(original['reviews'][0]['mapping'],'expected not exercised')
            self.assertEqual(result['errors'],['AT13 lease member expiry under control loss: blocked', HOLD_REVIEW_PHASE+': nonzero exit 1'])
            self.assertEqual(result['reviews'][0]['mapping'],'unexpected')
            self.assertEqual(review_path.read_bytes(),raw)
            self.assertEqual(result['reviews'][0]['items'],[])
            # A passing acceptance row also makes its aggregate driver exit 0.
            # An unexplained nonzero driver exit must still fail the job.
            at13=json.loads((root/'AT13.json').read_text())
            next(c for c in at13['checks'] if c['name']=='lease member expiry under control loss')['status']='pass'
            (root/'AT13.json').write_text(json.dumps(at13))
            self.assertEqual(summarize(root,original['phases'],original['run_id'])['status'],'fail')
            phases=copy.deepcopy(original['phases'])
            next(p for p in phases if p['name']==DRIVER_PHASE)['exit_code']=0
            # The former pure-stale exception cannot explain a fresh recurrence.
            recurrence=summarize(root,phases,original['run_id'])
            self.assertEqual(recurrence['status'],'fail')
            self.assertEqual(recurrence['errors'],[HOLD_REVIEW_PHASE+': nonzero exit 1'])
            self.assertEqual(review_path.read_bytes(),raw)
            initial=json.loads(raw)
            initial['coverage']['pure_stale_generation']='pass'
            initial['status']='pass'
            review_path.write_text(json.dumps(initial))
            next(p for p in phases if p['name']==HOLD_REVIEW_PHASE)['exit_code']=0
            self.assertEqual(summarize(root,phases,original['run_id'])['status'],'pass')
            # Each independent defect starts from complete valid coverage.
            for kind in ('another gap','fail','join problem','different run','missing item','attempt failure','wrong status','error','missing result','invalid result'):
                with self.subTest(kind=kind):
                    v=copy.deepcopy(initial)
                    if kind=='another gap':v['coverage']['independent_denial']='not exercised'
                    if kind=='fail':v['coverage']['populated_hold']='fail'
                    if kind=='join problem':v['join_problems']=[{'reason':'unmatched incident'}]
                    if kind=='different run':v['run_id']='older'
                    if kind=='missing item':del v['coverage']['matching_recovery']
                    if kind=='attempt failure':v['attempts'][0]['status']='fail'
                    if kind=='wrong status':v['status']='fail'
                    if kind=='error':v['errors']=['unexpected error']
                    review_path.write_text(json.dumps(v))
                    if kind=='missing result':review_path.unlink()
                    if kind=='invalid result':review_path.write_text('{')
                    self.assertEqual(summarize(root,phases,original['run_id'])['status'],'fail')
            review_path.write_text(json.dumps(initial))
            changed=copy.deepcopy(phases)
            next(p for p in changed if p['name']==HOLD_REVIEW_PHASE)['exit_code']=2
            self.assertEqual(summarize(root,changed,original['run_id'])['status'],'fail')
            changed=copy.deepcopy(phases)
            changed.append(dict(name='unknown review',exit_code=1))
            self.assertEqual(summarize(root,changed,original['run_id'])['status'],'fail')
            review_path.write_text(json.dumps(initial))
            r=summarize(root,phases,original['run_id'])
            self.assertEqual(r['status'],'pass')
            self.assertEqual(r['reviews'][0]['mapping'],'pass')
            self.assertEqual(r['reviews'][0]['items'],[])
            self.assertNotIn('remove from expected-not-exercised list',(root/'scheduled-job.md').read_text())
            self.assertEqual(json.loads(review_path.read_text())['status'],'pass')

    def test_d163_archived_gaps_remain_unchanged_and_are_no_longer_exempt(self):
        from nightly_summary import review_mapping, HOLD_REVIEW_PHASE
        history=Path(__file__).resolve().parents[4]/'docs/e2e/at/isolation/history'
        for workflow in ('41640196805','41836274140'):
            root=history/('follow-up-24-nightly-'+workflow)
            path=root/'populated-holds-review.json'
            raw=path.read_bytes();record=json.loads(raw)
            self.assertEqual(record['coverage']['pure_stale_generation'],'not exercised')
            result=review_mapping(root,dict(name=HOLD_REVIEW_PHASE,exit_code=1),record['run_id'])
            self.assertEqual(result['mapping'],'unexpected')
            self.assertEqual(sum(v=='pass' for v in result['coverage'].values()),8)
            self.assertEqual(path.read_bytes(),raw)
