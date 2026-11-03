"""Retention needs an open incident, reset counters and the actual parent receipt."""
import copy
import unittest
from review_populated_holds import retention


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
