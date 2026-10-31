"""Regression checks for attribution, incomplete evidence and clock domains."""
import copy
import unittest
import producer_timing as p


def fixture():
    def read(a,b):
        return dict(start_ns=str(a), end_ns=str(b), ok=True)
    v=dict(start_ns='150',end_ns='190',verified=True,tc_sha256='digest',
           nft_read=read(151,155),tc_read=read(165,180),
           precision_read=dict(start_ns='156',end_ns='160',read_ok=True,verified=True))
    d=dict(apply_start_ns='100',render_start_ns='110',install_start_ns='115',installed_by_ns='130',
           release_start_ns='132',release_end_ns='145',install_ok=True,guard_released=True,
           install_bound_ns='100000000',granularity_ns='1000000')
    return [dict(run_id='run',boot_id='boot',sequence='1',source='apply',tc_sha256='digest',
                 anchor=dict(mono_before_ns='201',mono_after_ns='203',wall='2026-10-31T00:00:00Z'),
                 timing=dict(version=1,clock='CLOCK_MONOTONIC',operation='apply',start_ns='100',end_ns='200',
                             succeeded=True,verified=True,deadline=d,verifications=[v]))]


def packet(at):
    return dict(sent_ns=str(at),read_completion_ns=str(at+80),ingress_ns=str(at+1),egress_ns='0',result='timeout',packet_id=1)


class CorrelationTests(unittest.TestCase):
    def test_inside_and_outside_guard_remain_losses(self):
        validated=p.validate_operations(fixture())
        inside=p.correlate(packet(120),validated)
        self.assertTrue(inside['ingress_in_install'])
        self.assertIn('loss retained',inside['classification'])
        release=p.correlate(packet(135),validated)
        self.assertFalse(release['install_window_overlap'])
        self.assertTrue(release['ingress_in_release'])
        self.assertTrue(release['ingress_in_outer_guard'])
        outside=p.correlate(packet(155),validated)
        self.assertFalse(outside['ingress_in_outer_guard'])
        self.assertTrue(outside['verifier_window_overlap'])
        self.assertIn('unresolved',outside['classification'])

    def test_failed_release_does_not_invent_guard_end(self):
        rows=fixture(); t=rows[0]['timing'];t['succeeded']=False
        t['deadline'].update(guard_released=False,release_start_ns='0',release_end_ns='0')
        valid=p.validate_operations(rows)
        self.assertEqual(valid['failed_operations'],1)
        loss=p.correlate(packet(135),valid)
        self.assertFalse(loss['ingress_in_outer_guard'])
        self.assertFalse(loss['nearest_apply']['complete_guard'])

    def test_partial_failure_before_verification_retained(self):
        rows=fixture();partial=copy.deepcopy(rows[0]);partial['sequence']='2'
        t=partial['timing'];t.update(succeeded=False,verified=False,verifications=[])
        t['deadline'].update(install_ok=False,guard_released=False)
        for k in t['deadline']:
            if k.endswith('_ns') and k!='apply_start_ns':t['deadline'][k]='0'
        rows.append(partial)
        valid=p.validate_operations(rows)
        self.assertEqual(len(valid['applies']),2)
        self.assertEqual(valid['failed_operations'],1)

    def test_sequence_digest_and_bounds_rejected(self):
        for mutation in ('gap','digest','order','float','release','overrun'):
            with self.subTest(mutation=mutation):
                rows=fixture();t=rows[0]['timing'];d=t['deadline']
                if mutation=='gap':rows[0]['sequence']='2'
                if mutation=='digest':rows[0]['tc_sha256']='different'
                if mutation=='order':d['installed_by_ns']='114'
                if mutation=='float':d['release_end_ns']=145
                if mutation=='release':d['install_ok']=False
                if mutation=='overrun':d['install_bound_ns']='1'
                with self.assertRaises(ValueError):p.validate_operations(rows)

    def test_both_recovery_verifiers_preserved(self):
        rows=fixture();v=copy.deepcopy(rows[0]['timing']['verifications'][0])
        for r in [v,v['nft_read'],v['precision_read'],v['tc_read']]:
            r['start_ns']=str(int(r['start_ns'])+50);r['end_ns']=str(int(r['end_ns'])+50)
        rows[0]['timing']['verifications'].append(v)
        rows[0]['timing']['end_ns']='250'
        rows[0]['anchor'].update(mono_before_ns='251',mono_after_ns='253')
        valid=p.validate_operations(rows)
        self.assertEqual(len(valid['verifiers']),2)
        self.assertEqual(len(valid['recovery_double_verifiers']),1)

    def test_baseline_heartbeat_sources_and_nearest_rank(self):
        rows=fixture()
        anchor=dict(mono_before_ns='301',mono_after_ns='303',wall='2026-10-31T00:00:00Z')
        rows += [dict(run_id='run',boot_id='boot',sequence='2',source='baseline',anchor=anchor,
                      baseline=dict(start_ns='250',end_ns='280',cause='policy_deadline',succeeded=True)),
                 dict(run_id='run',boot_id='boot',sequence='3',source='heartbeat',anchor=anchor,
                      heartbeat=dict(mono_ns='290',policy_expires_mono_ns='0',generation='1',written=True))]
        valid=p.validate_operations(rows)
        self.assertEqual(valid['snapshots'],3)
        self.assertEqual(len(valid['producer_rows']),2)
        self.assertEqual(valid['source_counts'],{'apply':1,'baseline':1,'heartbeat':1})
        self.assertEqual(valid['brackets']['outer_guard']['p99_ns'],'30')
        self.assertEqual(valid['brackets']['completed_guard_install_command']['max_ns'],'15')
        for kind in ('baseline','heartbeat'):
            bad=copy.deepcopy(rows)
            bad[1 if kind=='baseline' else 2][kind]['end_ns' if kind=='baseline' else 'mono_ns']='400'
            with self.assertRaises(ValueError):p.validate_operations(bad)
        # Two completed guards: nearest rank p50 is the first, not their mean.
        second=copy.deepcopy(rows[0]);second['sequence']='4'
        second['timing']['deadline']['release_end_ns']='146'
        valid=p.validate_operations(rows+[second])
        self.assertEqual(valid['brackets']['outer_guard']['p50_ns'],'30')
        self.assertEqual(valid['brackets']['outer_guard']['p90_ns'],'31')
        second['timing']['succeeded']=False
        valid=p.validate_operations(rows+[second])
        self.assertEqual(len(valid['failed_applies']),1)
        self.assertEqual(valid['brackets']['outer_guard']['count'],2)

    def test_clock_namespace_mismatch_and_missing_ingress_rejected(self):
        clock=dict(boot_id='boot',time_namespace='time:1',offsets='monotonic 0 0\nboottime 0 0',clock='CLOCK_MONOTONIC')
        other=dict(clock,time_namespace='time:2')
        p.validate_clock(other,clock)  # Distinct inode, exactly equal clock origin.
        with self.assertRaises(ValueError):p.validate_clock(dict(other,boot_id='different'),clock)
        with self.assertRaises(ValueError):p.validate_clock(dict(other,offsets='monotonic 1 0\nboottime 0 0'),clock)
        captures=[dict(event='ready',trace_clock=clock),dict(event='complete',statistics_available=True,drops=0)]
        probe=dict(packet_id=1,trace_clock=clock,result='timeout',send=dict(mono_ns='100'),read_completion=dict(mono_ns='180'))
        with self.assertRaises(ValueError):p.packet_trace([probe],captures,clock)
        captures.insert(1,dict(interface='fvmb0',packet_id=1,capture_read=dict(mono_ns='101')))
        self.assertEqual(len(p.packet_trace([probe],captures,clock)),1)
        captures[-1]['drops']=1
        with self.assertRaises(ValueError):p.packet_trace([probe],captures,clock)


if __name__=='__main__':
    unittest.main()
