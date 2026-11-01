"""Tests of the diagnostic's evidence/refusal boundary; no root or network."""
import copy
import importlib.util
from pathlib import Path
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location('analysis',Path(__file__).with_name('timing-analysis.py'))
analysis = importlib.util.module_from_spec(spec)
spec.loader.exec_module(analysis)

probe_spec = importlib.util.spec_from_file_location('probes', Path(__file__).with_name('timing-probes.py'))
probes = importlib.util.module_from_spec(probe_spec)
probe_spec.loader.exec_module(probes)


class RestartCollector(unittest.TestCase):
    def test_process_disappearing_during_proc_read_completes_restart(self):
        with mock.patch.object(probes, 'Path') as path, \
                mock.patch.object(probes.os, 'kill'), \
                mock.patch.object(probes.subprocess, 'Popen') as child, \
                mock.patch('builtins.open', mock.mock_open()):
            path.return_value.exists.return_value = True
            path.return_value.read_text.side_effect = ['123', ProcessLookupError()]
            child.return_value.pid = 456
            probes.restart_agent('TEST ONLY output')
            child.assert_called_once()

    def test_unreadable_proc_is_not_proof_of_exit(self):
        with mock.patch.object(probes, 'Path') as path, \
                mock.patch.object(probes.os, 'kill'), \
                mock.patch.object(probes.subprocess, 'Popen') as child:
            path.return_value.exists.return_value = True
            path.return_value.read_text.side_effect = ['123', PermissionError()]
            with self.assertRaises(PermissionError):
                probes.restart_agent('TEST ONLY output')
            child.assert_not_called()

    def test_shutdown_timeout_does_not_start_successor(self):
        with mock.patch.object(probes, 'Path') as path, \
                mock.patch.object(probes.os, 'kill') as kill, \
                mock.patch.object(probes.time, 'monotonic', side_effect=[0, 3]), \
                mock.patch.object(probes.subprocess, 'Popen') as child:
            path.return_value.exists.return_value = True
            path.return_value.read_text.return_value = '123'
            with self.assertRaisesRegex(RuntimeError, 'failed to stop gracefully'):
                probes.restart_agent('TEST ONLY output')
            kill.assert_called_once_with(123, probes.signal.SIGTERM)
            child.assert_not_called()


def fixtures():
    def row(sequence,source):
        return dict(run_id='TEST ONLY run',boot_id='TEST ONLY boot',sequence=str(sequence),source=source,
                    anchor=dict(mono_before_ns='500',mono_after_ns='501',wall='2026-10-31T00:00:00Z'))
    def cmd(start,end):
        return dict(start_ns=str(start),end_ns=str(end),ok=True)
    verification=dict(start_ns='180',end_ns='240',verified=True,tc_sha256='TEST ONLY digest',
                      nft_read=cmd(180,190),tc_read=cmd(190,210),
                      precision_read=dict(start_ns='210',end_ns='230',read_ok=True,verified=True))
    timing=dict(version=1,clock='CLOCK_MONOTONIC',operation='apply',start_ns='100',end_ns='250',succeeded=True,verified=True,
                verifications=[verification],deadline=dict(render_start_ns='110',install_start_ns='120',installed_by_ns='140',
                release_start_ns='150',release_end_ns='170',install_ok=True,guard_released=True,install_bound_ns='100000000'))
    apply=row(1,'apply');apply.update(timing=timing,tc_sha256='TEST ONLY digest')
    live=row(2,'live');live.update(timing=dict(timing,operation='verify'),tc_sha256='TEST ONLY digest')
    renewals=[]
    for sequence,start in [(3,250),(4,350)]:
        r=row(sequence,'renewal')
        r['renewal']=dict(new_lease=True,start_ns=str(start),end_ns=str(start+10))
        r['lease']=dict(node_deadline_ns='1000000000')
        renewals.append(r)
    def packet(sent,ingress,egress,result):
        p=dict(sent_ns=str(sent),ingress_ns=str(ingress),read_completion_ns=str(sent+80000000),result=result)
        if egress: p['egress_ns']=str(egress)
        return p
    renewal=dict(boot_id='TEST ONLY boot',packets=[packet(125,130,0,'timeout'),packet(400,401,402,'ok')],capture_stats=[dict(drops=0)])
    expiry=dict(boot_id='TEST ONLY boot',deadline_ns='1000000000',capture_stats=[dict(drops=0)],packets=[
        packet(799000000,799000001,800000000,'ok'),packet(849000000,849000001,850000000,'ok'),
        packet(899000000,899000001,900000000,'ok'),packet(1000000000,1000000001,0,'timeout'),
        packet(1050000000,1050000001,0,'timeout'),packet(1100000000,1100000001,0,'timeout')])
    return [apply,live]+renewals,renewal,expiry


class EvidenceBoundary(unittest.TestCase):
    def test_loss_inside_install_is_still_a_loss(self):
        result=analysis.analyze(*fixtures())
        self.assertEqual(result['renewal']['delivered'],1)
        self.assertEqual(len(result['renewal']['losses']),1)
        self.assertEqual(result['renewal']['loss_windows_overlapping_install'],1)
        self.assertEqual(result['expiry']['packet_predicate'],'pass')

    def test_loss_outside_guard_is_unresolved(self):
        ops,renewal,expiry=fixtures()
        renewal['packets'][0]['ingress_ns']='175'
        result=analysis.analyze(ops,renewal,expiry)
        self.assertEqual(result['renewal']['unresolved_ingress'],1)
        self.assertEqual(len(result['renewal']['losses']),1)

    def test_invalid_evidence_is_not_continuity(self):
        for kind in ['gap','boot','zero','numeric','digest','precision','release','deadline']:
            with self.subTest(kind=kind):
                ops,renewal,expiry=copy.deepcopy(fixtures())
                if kind=='gap': ops[1]['sequence']='3'
                elif kind=='boot': renewal['boot_id']='another boot'
                elif kind=='zero': ops[0]['timing']['deadline']['install_start_ns']='0'
                elif kind=='numeric': ops[0]['timing']['start_ns']=100
                elif kind=='digest': ops[0]['tc_sha256']='another digest'
                elif kind=='precision': ops[0]['timing']['verifications'][0]['precision_read']['verified']=False
                elif kind=='release': ops[0]['timing']['deadline']['install_ok']=False
                elif kind=='deadline': expiry['deadline_ns']='1000000001'
                with self.assertRaises((AssertionError,ValueError)):
                    analysis.analyze(ops,renewal,expiry)


if __name__=='__main__':
    unittest.main()
