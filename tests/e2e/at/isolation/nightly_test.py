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

    def test_unexecuted_foreign_redirect_writes_blocked_verdict_without_traceback(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            (root/'receiver-capture.json').write_text(json.dumps(dict(run_id='775bc692-e6ef-4490-8056-ba6cfa818b69',transactions=[])))
            result=subprocess.run(['python3','-B',str(Path(__file__).with_name('review_followup8.py')),tmp],text=True,capture_output=True)
            self.assertEqual(result.returncode,0,result.stderr)
            verdict=json.loads((root/'recovery-verdict.json').read_text())
            self.assertFalse(verdict['passed'])
            self.assertEqual(verdict['status'],'blocked')
