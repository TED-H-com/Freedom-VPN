"""Regression checks for attribution, incomplete evidence and clock domains."""
import copy
import json
from pathlib import Path
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


def host_fixture():
    def sample(a, b, total):
        return dict(start_ns=str(a), end_ns=str(b), ok=True, cpu_total_ticks=str(total),
                    cpu_idle_ticks='20', cpu_iowait_ticks='0', procs_running='96', procs_blocked='0')
    rows = fixture()
    rows[0]['host'] = dict(before_apply=sample(90,99,100), after_apply=sample(200,201,200))
    return rows


def failed_install():
    rows = host_fixture()
    t = rows[0]['timing']
    t.update(succeeded=False, verified=False, verifications=[])
    # Synthetic tiny bound exercises the same integer comparison; no product bound change.
    t['deadline'].update(install_ok=False, guard_released=False, install_bound_ns='10',
                         release_start_ns='0', release_end_ns='0')
    return rows


def command(shape='success', phase='install', kind='nft', base=115):
    def stamp(n=None):
        return dict(ns=str(n or 0), available=n is not None)
    def resources(n):
        return dict(bracket=dict(before=stamp(n), after=stamp(n+1)), cgroup_v2_available=False,
                    visible_ancestors_complete=False, host_ancestors_available=False, cgroups=None)
    c = dict(kind=kind, phase=phase, context_deadline=dict(before=stamp(1000), after=stamp(1001)),
             start_attempt=stamp(base+2), start_return=stamp(base+3), child_started_by=stamp(base+3),
             child_start_exact=stamp(), cancel_observed=stamp(), wait_return=stamp(base+8),
             context_before='none', context_after='none', context_at_cancel='unavailable',
             started=True, ok=True, exit_available=True, exit_code=0, signal_available=True, signal=0,
             cpu_available=True, user_cpu_ns='1', system_cpu_ns='2', switches_available=True,
             voluntary_switches='1', involuntary_switches='0', before=resources(base), after=resources(base+9))
    if shape == 'start_failure':
        c.update(started=False, ok=False, child_started_by=stamp(), wait_return=stamp(), exit_available=False,
                 signal_available=False, cpu_available=False, switches_available=False,
                 user_cpu_ns='0', system_cpu_ns='0', voluntary_switches='0')
    if shape in ('canceled', 'deadline'):
        state = 'canceled' if shape == 'canceled' else 'deadline_exceeded'
        c.update(ok=False, exit_code=-1, signal=9, cancel_observed=stamp(base+5),
                 context_at_cancel=state, context_after=state)
    if shape == 'exit_failure':
        c.update(ok=False, exit_code=1)
    if shape == 'unknown_exit':
        c.update(exit_available=False, signal_available=False, cpu_available=False, switches_available=False,
                 user_cpu_ns='0', system_cpu_ns='0', voluntary_switches='0')
    return c


class CorrelationTests(unittest.TestCase):
    def test_lifecycle_each_shape_is_joined_and_cause_not_inferred(self):
        for shape in ('success', 'start_failure', 'canceled', 'deadline', 'exit_failure', 'unknown_exit'):
            with self.subTest(shape=shape):
                rows = failed_install()
                rows[0]['timing']['commands'] = [command(shape)]
                a = p.validate_operations(rows)['failed_applies'][0]
                self.assertEqual(a['cause_status'], 'unproven')
                self.assertIn('nft: started=', a['install_phase_description'])
                joined = a['command_lifecycle']['records'][0]
                self.assertEqual(p.identity(joined), p.identity(rows[0]))
                self.assertEqual(joined['record'], command(shape))
                self.assertFalse(a['complete_guard'])
                other = copy.deepcopy(rows[0]); other.update(run_id='other', boot_id='other-boot')
                del other['timing']['commands']
                v = p.validate_operations(rows+[other])
                self.assertEqual(v['applies'][1]['install_phase_description'], 'unavailable')
                self.assertEqual(v['applies'][1]['command_lifecycle']['status'], 'unavailable')
                # Command records alone do not supply host samples or cause.
                del rows[0]['host']
                a = p.validate_operations(rows)['applies'][0]
                self.assertEqual(a['diagnostic_label'], 'unavailable')
                self.assertEqual(a['cause_status'], 'unproven')

    def test_lifecycle_all_phases_kinds_resources_and_missing_stamps(self):
        for phase in p.COMMAND_PHASES:
            for kind in ('nft', 'tc', 'conntrack', 'ip', 'other'):
                rows = fixture()
                c = command(phase=phase, kind=kind, base=132 if phase == 'release' else 115)
                c['before'].update(cgroup_v2_available=True, visible_ancestors_complete=True,
                    cgroups=[dict(depth=0, **{k:dict(available=True,value=v) for k,v in
                        [('cpu_max','max 100000'),('cpu_weight','100'),('cpu_stat','usage_usec 4\nnr_throttled 0'),
                         ('cpu_pressure','some avg10=0.00 avg60=0.00 avg300=0.00 total=0') ]})])
                rows[0]['timing']['commands'] = [c]
                self.assertEqual(p.validate_operations(rows)['applies'][0]['command_lifecycle']['status'], 'available')
                # Unavailable timestamps remain absent; no invented time bounds.
                for k in p.COMMAND_STAMPS:
                    c[k] = dict(ns='0', available=False)
                p.validate_operations(rows)

    def test_successful_install_command_over_100ms_still_fails_closed(self):
        rows = failed_install(); del rows[0]['host']
        t = rows[0]['timing']; t['commands'] = [command()]
        # Scale the synthetic mono clock, preserving every enclosing bracket.
        def scale(value):
            for k, v in value.items():
                if isinstance(v, dict): scale(v)
                elif isinstance(v, str) and (k.endswith('_ns') or k == 'ns'):
                    value[k] = str(int(v)*10000000)
        scale(rows[0]['anchor']); scale(t)
        for c in t['commands']: scale(c)
        t['deadline'].update(install_bound_ns='100000000', install_ok=True)
        a = p.validate_operations(rows)['failed_applies'][0]
        self.assertEqual(a['failure_stage'], 'install')
        self.assertTrue(a['install_overrun'])
        self.assertFalse(a['complete_guard'])
        self.assertIn('ok=True', a['install_phase_description'])
        self.assertEqual(a['cause_status'], 'unproven')

    def test_invalid_lifecycle_records_are_rejected(self):
        changes = [
            lambda c: c.update(kind='nft -f secret'), lambda c: c.update(phase='unknown'),
            lambda c: c.update(started=1), lambda c: c.update(ok='true'),
            lambda c: c.update(exit_code=True), lambda c: c.update(signal=65),
            lambda c: c.update(exit_available=False, exit_code=1),
            lambda c: c.update(exit_code=1), lambda c: c.update(context_after='raw error'),
            lambda c: c.update(output='sensitive'), lambda c: c['start_attempt'].update(ns=117),
            lambda c: c['start_return'].update(ns='0118'), lambda c: c['wait_return'].update(ns='90'),
            lambda c: c['start_attempt'].update(available=False),
            lambda c: c['child_started_by'].update(ns='119'), lambda c: c.update(started=False),
            lambda c: c['before']['bracket']['after'].update(ns='130'),
            lambda c: c['after']['bracket']['after'].update(ns='131'),
            lambda c: c['context_deadline']['after'].update(ns='999'),
            lambda c: c.update(user_cpu_ns='-1'), lambda c: c.update(voluntary_switches='9223372036854775808'),
            lambda c: c['after'].update(visible_ancestors_complete=True),
        ]
        for change in changes:
            with self.subTest(change=changes.index(change)):
                rows = fixture(); c = command(); change(c); rows[0]['timing']['commands'] = [c]
                with self.assertRaises(ValueError): p.validate_operations(rows)
        rows = fixture(); rows[0]['timing']['commands'] = 'unavailable'
        with self.assertRaises(ValueError): p.validate_operations(rows)

    def test_historical_followup11_replays_unchanged_except_additive_unavailable(self):
        root = Path(__file__).resolve().parents[4]/'docs/e2e/at/isolation/history/follow-up-11'
        rows = [json.loads(s) for s in (root/'producer-operations.jsonl').read_text().splitlines()]
        before = copy.deepcopy(rows)
        v = p.validate_operations(rows)
        old = json.loads((root/'producer-timing-analysis.json').read_text())
        self.assertEqual(rows, before)
        for key in ('snapshots', 'runs', 'source_counts', 'brackets', 'failed_operations', 'recovery_double_verifiers'):
            self.assertEqual(v[key], old[key])
        def historical(a):
            self.assertEqual(a['command_lifecycle']['status'], 'unavailable')
            self.assertEqual(a['install_phase_description'], 'unavailable')
            return {k:v for k,v in a.items() if k not in ('command_lifecycle', 'install_phase_description')}
        self.assertEqual([historical(a) for a in v['failed_applies']], old['failed_applies'])
        self.assertEqual([historical({k:v for k,v in a.items() if k not in ('operation','lease')}) for a in v['applies']], old['apply_contexts'])

    def test_host_join_overrun_is_not_causal_or_failed_verification(self):
        rows = failed_install()
        rows[0]['host'].update(failed_command='nft', exit_code=-1, signal=9)
        a = p.validate_operations(rows)['failed_applies'][0]
        self.assertEqual(p.identity(a), p.identity(rows[0]))
        self.assertEqual(a['diagnostic_label'], 'fail-closed install overrun with recorded host contention')
        self.assertEqual(a['cause_status'], 'unproven')
        self.assertEqual(a['failure_stage'], 'install')
        self.assertEqual(a['verification_status'], 'unavailable')
        self.assertEqual(a['host_context']['idle_percent'], 0)
        self.assertEqual(a['host_context']['command_status'], dict(failed_command='nft', exit_code=-1, signal=9))
        self.assertFalse(a['complete_guard'])
        self.assertNotIn('outer_guard', p.validate_operations(rows)['brackets'])
        # Same sequence, distinct process. Never inherit host state across runs.
        other = copy.deepcopy(rows[0]); other['run_id'] = 'other'; del other['host']
        v = p.validate_operations(rows + [other])
        self.assertEqual(v['failed_applies'][1]['diagnostic_label'], 'unavailable')
        self.assertIsNone(v['failed_applies'][1]['host_context']['command_status']['exit_code'])
        self.assertEqual(len(v['failed_applies']), 2)

    def test_host_missing_failed_zero_and_decreasing_samples_are_unavailable(self):
        for change in ('absent', 'failed', 'zero', 'decreasing'):
            with self.subTest(change=change):
                rows = failed_install(); h = rows[0]['host']
                if change == 'absent': del rows[0]['host']
                if change == 'failed': h['after_apply']['ok'] = False
                if change == 'zero': h['after_apply']['cpu_total_ticks'] = '100'
                if change == 'decreasing': h['after_apply']['cpu_total_ticks'] = '99'
                a = p.validate_operations(rows)['applies'][0]
                self.assertEqual(a['diagnostic_label'], 'unavailable')
                self.assertEqual(a['host_context']['status'], 'unavailable')
                self.assertIsNone(a['host_context']['idle_percent'])
                self.assertTrue(a['install_overrun'])
                self.assertEqual(a['cause_status'], 'unproven')

    def test_bound_and_host_must_both_support_label(self):
        for change in ('idle', 'iowait', 'one_runnable', 'within_bound', 'no_install'):
            with self.subTest(change=change):
                rows = failed_install(); h = rows[0]['host']; d = rows[0]['timing']['deadline']
                if change == 'idle': h['after_apply']['cpu_idle_ticks'] = '40'
                if change == 'iowait': h['after_apply']['cpu_iowait_ticks'] = '10'
                if change == 'one_runnable': h['before_apply']['procs_running'] = '1'
                if change == 'within_bound': d['install_bound_ns'] = '100000000'
                if change == 'no_install': d.update(render_start_ns='0', install_start_ns='0', installed_by_ns='0')
                a = p.validate_operations(rows)['applies'][0]
                self.assertNotIn('with recorded host contention', a['diagnostic_label'])
                self.assertEqual(a['cause_status'], 'unproven')
        rows = host_fixture()
        a = p.validate_operations(rows)['applies'][0]
        self.assertEqual(a['verification_status'], 'passed')
        self.assertEqual(a['host_context']['status'], 'available')
        self.assertEqual(a['host_context']['command_status'], dict(failed_command=None, exit_code=None, signal=None))
        rows[0]['timing'].update(succeeded=False, verified=False)
        rows[0]['timing']['verifications'][0]['verified'] = False
        a = p.validate_operations(rows)['failed_applies'][0]
        self.assertEqual(a['failure_stage'], 'verification')
        self.assertEqual(a['verification_status'], 'failed')
        self.assertTrue(a['complete_guard'])

    def test_invalid_host_or_command_status_rejected(self):
        for change in ('early', 'late', 'float', 'leading_zero', 'overflow', 'total', 'delta', 'command', 'output', 'exit', 'signal', 'success'):
            with self.subTest(change=change):
                rows = failed_install(); h = rows[0]['host']
                if change == 'early': h['after_apply']['start_ns'] = '199'
                if change == 'late': h['before_apply']['end_ns'] = '101'
                if change == 'float': h['before_apply']['cpu_total_ticks'] = 100.0
                if change == 'leading_zero': h['before_apply']['cpu_total_ticks'] = '0100'
                if change == 'overflow': h['before_apply']['cpu_total_ticks'] = '18446744073709551616'
                if change == 'total': h['before_apply']['cpu_idle_ticks'] = '101'
                if change == 'delta': h['after_apply'].update(cpu_total_ticks='110', cpu_idle_ticks='40')
                if change == 'command': h['failed_command'] = 'nft -f sensitive-input'
                if change == 'output': h['stderr'] = 'not permitted'
                if change == 'exit': h['exit_code'] = True
                if change == 'signal': h['signal'] = '9'
                if change == 'success': rows = host_fixture(); rows[0]['host']['failed_command'] = 'tc'
                with self.assertRaises(ValueError): p.validate_operations(rows)

    def test_historical_first_attempt_remains_unavailable_and_all_losses_counted(self):
        root = Path(__file__).resolve().parents[4]/'docs/e2e/at/isolation/history/follow-up-10-attempt-1'
        rows = [json.loads(s) for s in (root/'producer-operations.jsonl').read_text().splitlines()]
        v = p.validate_operations(rows)
        self.assertEqual(len(v['failed_applies']), 49)
        for a in v['failed_applies']:
            self.assertEqual(a['failure_stage'], 'install')
            self.assertEqual(a['verification_status'], 'unavailable')
            self.assertEqual(a['diagnostic_label'], 'unavailable')
            self.assertEqual(a['cause_status'], 'unproven')
            self.assertTrue(a['install_overrun'])
        self.assertEqual(v['failed_applies'][0]['sequence'], '712')
        self.assertEqual(v['failed_applies'][0]['render_to_installed_ns'], '215890796')
        self.assertEqual(v['brackets']['outer_guard']['count'], 678)
        packets = p.packet_trace(json.loads((root/'renewal-probes.json').read_text()),
                                 json.loads((root/'renewal-packets.json').read_text()),
                                 json.loads((root/'producer-clock.json').read_text()))
        losses = [p.correlate(x, v) for x in packets if x['result'] != 'ok']
        self.assertEqual(len(packets), 1396)
        self.assertEqual(len(losses), 279)
        self.assertEqual(sum(bool(x['ingress_in_outer_guard']) for x in losses), 29)

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
        loss=p.correlate(packet(260),valid)
        self.assertEqual(loss['baseline_window_overlap'][0]['baseline']['cause'],'policy_deadline')
        self.assertIsNone(loss['last_written_heartbeat'])
        loss=p.correlate(packet(310),valid)
        self.assertEqual(loss['last_completed_baseline']['sequence'],'2')
        self.assertEqual(loss['last_written_heartbeat']['sequence'],'3')
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
