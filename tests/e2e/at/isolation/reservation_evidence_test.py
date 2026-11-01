import unittest
from review_reservations import bracket,elapsed_from_wall


class ReservationEvidence(unittest.TestCase):
    def test_clock_interval_keeps_uncertainty(self):
        row={'anchor':{'wall':'2026-11-01T00:00:01Z','mono_before_ns':'1000','mono_after_ns':'1010'}}
        self.assertEqual(elapsed_from_wall(row,'2000','2026-11-01T00:00:01Z'),{'lower_ns':'990','upper_ns':'1000'})

    def test_preapply_cost_included_and_bad_order_rejected(self):
        r={'host':{'before_apply':{'ok':True,'end_ns':'100'}},'timing':{'start_ns':'1100'}}
        self.assertEqual(bracket(r)['duration_ns'],'1000')
        r['timing']['start_ns']='99'
        with self.assertRaises(ValueError):bracket(r)
        self.assertEqual(bracket({'timing':{}}),{'available':False})


class ParentReceiptIdentity(unittest.TestCase):
    def test_other_launch_pid_or_ticks_is_not_this_exit(self):
        from review_stops import matched
        i=dict(launch_id='TEST ONLY',pid=12,start_ticks='100')
        r=dict(id='TEST ONLY',pid=12,start_ticks='100',exit_code=0,signal=None)
        self.assertTrue(matched(i,r))
        for key,value in [('id','OTHER TEST ONLY'),('pid',13),('start_ticks','101')]:
            self.assertFalse(matched(i,dict(r,**{key:value})))
        self.assertFalse(matched({},{}))
