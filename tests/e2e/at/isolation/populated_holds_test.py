"""Retention needs an open incident, reset counters and the actual parent receipt."""
import copy
import unittest
from review_populated_holds import retention, central_age_fallback


class Retention(unittest.TestCase):
    def setUp(self):
        incident = dict(id='incident',first_seen_at='2026-01-01T00:00:00Z',ended_at=None)
        self.attempt = dict(incident_id='incident',arm={'id':'arm'},
            before_stop={'incidents':[incident]},after_stop={'incidents':[incident]},after_restart={'incidents':[incident]},
            stop=dict(start='2026-01-01T00:00:10Z',output='{"launch_id":"launch","pid":1,"start_ticks":"4","exit_observed":true,"parent_wait_receipt":{"id":"launch","pid":1,"start_ticks":"4","signal":15}}'),
            restart={'start':'2026-01-01T00:00:12Z'},
            proxy_after_stop={'arm':{'id':'arm'},'hold_started_at':'2026-01-01T00:00:00Z'})
        self.samples = [dict(hold_id='incident',received_at=t,reported_steps=s,reported_elapsed_ms=e) for t,s,e in
            [('2026-01-01T00:00:09Z','9','9000'),('2026-01-01T00:00:13Z','1','0')]]

    def test_active_hold_retains_signal_receipt_without_claiming_exit_zero(self):
        result = retention(self.attempt,self.samples)
        self.assertEqual(result['status'],'pass')
        self.assertEqual(result['stop_receipt']['signal'],15)

    def test_ended_incident_reset_or_proxy_restart_cannot_pass(self):
        for kind in ('ended','counter','proxy','receipt'):
            attempt=copy.deepcopy(self.attempt);samples=copy.deepcopy(self.samples)
            if kind=='ended':attempt['after_stop']['incidents'][0]['ended_at']='2026-01-01T00:00:11Z'
            if kind=='counter':samples[1]['reported_steps']='10'
            if kind=='proxy':attempt['proxy_after_stop']['released_at']='2026-01-01T00:00:11Z'
            if kind=='receipt':attempt['stop']['output']='{}'
            with self.subTest(kind=kind):self.assertEqual(retention(attempt,samples)['status'],'fail')


class CentralFallback(unittest.TestCase):
    def test_original_age_audit_needs_no_alarm_and_one_audit(self):
        incident=dict(first_seen_at='2026-01-01T00:00:00Z')
        audit=dict(occurred_at='2026-01-01T00:00:30Z',evidence=dict(reported_alarm=False,alarm_report_id=None))
        self.assertEqual(central_age_fallback(incident,[audit]),[audit])
        for kind in ('early','reported_alarm','alarm_id','missing_alarm','duplicate'):
            a=copy.deepcopy(audit)
            if kind=='early':a['occurred_at']='2026-01-01T00:00:29.999999999Z'
            if kind=='reported_alarm':a['evidence']['reported_alarm']=True
            if kind=='alarm_id':a['evidence']['alarm_report_id']='report'
            if kind=='missing_alarm':del a['evidence']['reported_alarm']
            audits=[a,a] if kind=='duplicate' else [a]
            with self.subTest(kind=kind):self.assertEqual(central_age_fallback(incident,audits),[])
