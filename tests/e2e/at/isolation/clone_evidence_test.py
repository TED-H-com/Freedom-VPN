"""TEST ONLY sanitized parser inputs; no Docker or alternate reviewer implementation."""
import copy
import json
from pathlib import Path
import tempfile
import unittest
from auxiliary_inputs import seal, inputs
from evidence_io import redact_error, redact_text
from review_clone import evaluate, heartbeat, log_rows, review as review_clone
from review_stops import review as review_stops
from candidate_evidence import wall_ns

STAMP = '2026-11-05T06:00:'
def at(seconds):
    return STAMP+f'{seconds:02d}Z'


def fixture():
    manifest = dict(run_id='TEST ONLY current', build='a'*40, source_dirty=False,
                    started_at=at(0), completed_at=at(30))
    proc = dict(pid=42, start_ticks='100', status='available', role='fvpn-agent',
                executable='/usr/local/bin/fvpn-agent', command_binary='/usr/local/bin/fvpn-agent',
                config_path='/etc/vpn-node/agent-run.json', stdout_inode=11)
    report = '00000000-0000-7000-8000-000000000001'
    text = (f'time={at(4)} level=INFO msg="observation enqueued" report_id={report}\n'
            f'time={at(5)} level=INFO msg="observation submission" process_id=42 report_ids=[{report}] accepted_report_ids=[{report}] outcome=accepted http_status=202\n')
    logs = {n:dict(status='available', text=text if n=='agent' else '', inode=11, read_start=at(6), read_end=at(7)) for n in ('agent', 'watchdog', 'resolver-proxy')}
    snap = dict(process_coherent=True, process_before=proc, process_after=proc,
                executable_sha256='c'*64, agent_process_id=dict(text='TEST ONLY launch'),
                agent_selected=dict(text='/usr/local'), read_start=at(6), read_end=at(7),
                configs={'/etc/vpn-node/agent-run.json':dict(stable_during_snapshot=True, status='available', fields=dict(control_url='https://127.0.0.1:9444'), datapath=dict(snat='198.51.100.23'))},
                control_host=dict(status='available', addresses=['198.51.100.10']), logs=logs, services=[dict(role='resolver-proxy', status='available', pid=43)],
                network=dict(transport_sockets=dict(status='available', text='ESTAB 0 0 127.0.0.1:10000 127.0.0.1:9444 users:(("fvpn-agent",pid=42,fd=6))\nESTAB 0 0 198.51.100.23:10001 198.51.100.10:9443 users:(("fvpn-e2e-nodeconfig",pid=43,fd=7))'),
                             proxy_listener=dict(status='available', text='LISTEN 0 0 127.0.0.1:9444 0.0.0.0:* users:(("fvpn-e2e-nodeconfig",pid=43,fd=6))')))
    snapshots = [dict(manifest, role='clone', container='fvpn-w4-isolation-node-clone', status='available', phase=phase,
                      read_start=at(6), read_end=at(7), container_identity=dict(id='d'*64, started_at=at(1)), snapshot=copy.deepcopy(snap))
                 for phase in ('after-clone-creation','before-terminal-stop','after-terminal-stop')]
    post=snapshots[2]['snapshot'];post.update(read_start=at(22),read_end=at(23),process_coherent=False)
    for log in post['logs'].values():log.update(read_start=at(22),read_end=at(23))
    for phase,seconds in [('after-clone-creation',6),('after-clone-observation',14)]:
        source=copy.deepcopy(snap);source.update(read_start=at(seconds),read_end=at(seconds+1))
        snapshots.append(dict(manifest,role='source',container='fvpn-w4-isolation-node',status='available',phase=phase,read_start=at(seconds),read_end=at(seconds+1),snapshot=source))
    fingerprint={k:'e'*64 for k in ('machine_id_sha256','agent_spki_sha256','netbird_public_key_sha256')}
    rows=[]
    for seconds, label in ((8,'clone acceptance window'),(12,'freeze latch held')):
        readback=('v1\nboot_id TEST-ONLY\nmono_ns '+str(seconds*1000000000)+'\nwall_unix_ns '+str(wall_ns(at(seconds)))+
                  '\npolicy_expires_mono_ns 0\nbaseline_ok 1\nserving 0\ngeneration 0\nreason identity_frozen\n')
        row=dict(label=label,read_start=at(seconds),read_end=at(seconds+1),readback=readback,forward_view=dict(Baseline=True,Policy='drop'))
        if seconds==8:
            row['clone_observation']=dict(Source=fingerprint,Clone=fingerprint,same_identity=True,conflicts=1,mapping_frozen=True)
        rows.append(row)
    data={'clone-runtime.json':dict(manifest,snapshots=snapshots), 'identity-freeze.json':dict(manifest,fixture_created=True,started_at=at(1),completed_at=at(2),rows=rows)}
    data['receiver-capture.json']=dict(manifest,transactions=[dict(path='/api/v1/agent/observations',status=202,received_at=at(7),response_at=at(8),body=dict(report_id=report,member_forwarding_enabled=False,applied_config_generation=None,errors=[dict(stage='identity',message='identity_frozen',code='HEALTH_CHECK_FAILED')]),response=dict(accepted=True,identity_frozen=True,report_id=report))])
    stops=dict(manifest,stops=[dict(label='direct_agent',container=c,final_stop_matched=True,closure_start=at(19),closure_end=at(21),
                                  parent_wait_receipt=dict(id='TEST ONLY launch',pid=42,start_ticks='100',started_at=at(3),wait_observed_at=at(20)))
                             for c in ('fvpn-w4-isolation-node','fvpn-w4-isolation-node-clone')])
    return manifest,data,stops


class CloneEvidence(unittest.TestCase):
    def test_complete_required_chain(self):
        m,d,s=fixture();r=evaluate(m,d,s)
        self.assertEqual(r['status'],'pass',r)
        self.assertTrue(all(v['status']=='established' for v in r['criteria'].values()))

    def test_clone_not_created_and_early_exit(self):
        for case in ('absent','exit'):
            m,d,s=fixture()
            if case=='absent':
                d['identity-freeze.json']['fixture_created']=False;d['clone-runtime.json']['snapshots']=[]
            else:
                for row in d['clone-runtime.json']['snapshots']:
                    row['snapshot']['process_coherent']=False;row['snapshot']['process_before']={}
            r=evaluate(m,d,s);self.assertNotEqual(r['status'],'pass')
            self.assertEqual(r['criteria']['executing_release_config']['status'],'not established')

    def test_missing_rotated_and_incomplete_logs(self):
        for case in ('missing','rotated','incomplete','parse'):
            m,d,s=fixture()
            for row in d['clone-runtime.json']['snapshots']:
                log=row['snapshot']['logs']['agent']
                if case=='rotated':log['inode']=12
                elif case=='parse':log['text']+='not a structured log row\n'
                else:log['status']=case
            r=evaluate(m,d,s)
            self.assertEqual(r['criteria']['corrected_transport']['status'],'not established',case)
            self.assertNotEqual(r['status'],'pass')

    def test_inconsistent_process_config_or_parent(self):
        for case in ('process','config','parent','historical-log'):
            m,d,s=fixture()
            for row in d['clone-runtime.json']['snapshots']:
                snap=row['snapshot']
                if case=='process':snap['process_coherent']=False
                if case=='config':snap['configs']['/etc/vpn-node/agent-run.json']['stable_during_snapshot']=False
                if case=='historical-log':snap['logs']['agent']['text']=snap['logs']['agent']['text'].replace(at(4),at(0)).replace(at(5),at(0))
            if case=='parent':s['stops'][1]['parent_wait_receipt']['start_ticks']='101'
            self.assertEqual(evaluate(m,d,s)['criteria']['corrected_transport']['status'],'not established',case)

    def test_static_url_is_not_live_transport(self):
        m,d,s=fixture()
        for row in d['clone-runtime.json']['snapshots']:row['snapshot']['network']={}
        r=evaluate(m,d,s)
        self.assertEqual(r['criteria']['corrected_transport']['status'],'not established')
        self.assertNotEqual(r['status'],'pass')

    def test_frozen_mapping_requires_matching_consumer_latch(self):
        m,d,s=fixture();d['identity-freeze.json']['rows'][1]['readback']=d['identity-freeze.json']['rows'][1]['readback'].replace('reason identity_frozen','reason stale_generation')
        r=evaluate(m,d,s)
        self.assertEqual(r['criteria']['central_conflict_frozen']['status'],'established')
        self.assertEqual(r['criteria']['consumer_latch_held']['status'],'not established')
        self.assertEqual(r['status'],'fail')

    def test_heartbeat_and_log_parse_errors_are_explicit(self):
        with self.assertRaises(ValueError):heartbeat('identity_frozen')
        _,errors=log_rows(dict(status='available',text='broken line'))
        self.assertTrue(errors)

    def test_historical_or_mismatched_run_and_real_stop_join(self):
        m,d,_=fixture()
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory)
            d['AT13.json']=dict(m,checks=[])
            d['container-lifecycle-events.json']=dict(m,events=[])
            # Preserve unavailable earlier stages; direct rows are still reviewed.
            d.pop('identity-freeze.json')
            d['receiver-closure.json']=[]
            for c in ('fvpn-w4-isolation-node','fvpn-w4-isolation-node-clone'):
                r=dict(launch_id='TEST ONLY launch',pid=42,start_ticks='100',signal_ok=True,exit_observed=True,
                       parent_wait_receipt=dict(id='TEST ONLY launch',pid=42,start_ticks='100',started_at=at(3),wait_observed_at=at(20),exit_code=0,signal=None))
                d['receiver-closure.json'].append(dict(m,container=c,start=at(19),end=at(21),ok=True,output=json.dumps(r),stored_lease_survived_stop=True))
            for name,value in d.items():(root/name).write_text(json.dumps(value))
            seal(root,m)
            reviewed=review_stops(root,m['run_id'])
            self.assertTrue(all(r['final_stop_matched'] for r in reviewed['stops']))
            # A copied parent receipt with the right ID but old time fails.
            r['parent_wait_receipt']['wait_observed_at']='2026-11-04T06:00:20Z'
            d['receiver-closure.json'][1]['output']=json.dumps(r)
            (root/'receiver-closure.json').write_text(json.dumps(d['receiver-closure.json']));seal(root,m)
            self.assertFalse(review_stops(root,m['run_id'])['stops'][1]['final_stop_matched'])
            d['AT13.json']['run_id']='TEST ONLY historical'
            (root/'AT13.json').write_text(json.dumps(d['AT13.json']));seal(root,m)
            with self.assertRaisesRegex(ValueError,'run mismatch'):inputs(root,m['run_id'])
            self.assertFalse(review_stops(root,m['run_id'])['all_parent_receipts_matched'])

    def test_seal_records_actual_compressed_filename(self):
        m,_,_=fixture()
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory);(root/'clone-runtime.json').write_text(json.dumps(dict(m,padding='x'*(1<<20))))
            receipt=seal(root,m)
            self.assertEqual(next(r for r in receipt['files'] if r['logical_name']=='clone-runtime.json')['path'],'clone-runtime.json.gz')
            self.assertFalse((root/'clone-runtime.json').exists());inputs(root,m['run_id'])

    def test_error_cases_and_missing_inputs_are_redacted(self):
        m,_,_=fixture()
        home='/'+'home/'+'alice/secret'
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory)
            stop_out=review_stops(root,m['run_id'])
            clone_out=review_clone(root,m['run_id'])
            self.assertEqual(stop_out['status'],'unavailable')
            self.assertEqual(clone_out['status'],'not established')
            for doc in ((root/'stop-receipt-review.json').read_text(), (root/'clone-transport-review.json').read_text()):
                self.assertNotIn('/'+'home/',doc)
                self.assertNotIn(str(root),doc)
            err=FileNotFoundError(2,'No such file or directory',home+'/auxiliary-inputs.json')
            self.assertNotIn('alice',redact_error(err))
            self.assertIn('auxiliary-inputs.json',redact_error(err))
            self.assertNotIn('secretvalue',redact_text('path='+home+' token=secretvalue "setup_key":"secretvalue"'))


if __name__=='__main__':unittest.main()
